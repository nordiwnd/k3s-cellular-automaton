package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan []byte)
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientsMu sync.Mutex
)

type CellUpdate struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Namespace string `json:"namespace"`
}

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Use in-cluster config if available, otherwise fallback to kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %s", err.Error())
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error building clientset: %s", err.Error())
	}

	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		namespace = "cellular-automaton"
	}

	// Start Informer
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Minute*10, informers.WithNamespace(namespace))
	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			handlePodUpdate(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			handlePodUpdate(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			handlePodDelete(obj)
		},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)
	factory.Start(stopCh)

	// Broadcaster
	go handleMessages()

	// HTTP Server
	http.HandleFunc("/ws", handleConnections)
	http.HandleFunc("/api/pods/", func(w http.ResponseWriter, r *http.Request) {
		handleChaos(w, r, clientset, namespace)
	})

	log.Println("Controller started on :8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func handlePodUpdate(obj interface{}) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return
	}

	// Check if it's a cell pod
	if app, ok := pod.Labels["app"]; !ok || app != "cell" {
		return
	}

	status := "unknown"
	if s, ok := pod.Labels["game-status"]; ok {
		status = s
	} else {
		// If no label, it might be initializing
		status = "initializing"
	}

	// Also consider DeletionTimestamp as "dying"
	if pod.DeletionTimestamp != nil {
		status = "terminating"
	}

	update := CellUpdate{
		Name:      pod.Name,
		Status:    status,
		Namespace: pod.Namespace,
	}

	msg, _ := json.Marshal(update)
	broadcast <- msg
}

func handlePodDelete(obj interface{}) {
	// When a pod is deleted, we might receive a DeletedFinalStateUnknown
	pod, ok := obj.(*v1.Pod)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			return
		}
		pod, ok = tombstone.Obj.(*v1.Pod)
		if !ok {
			return
		}
	}

	if app, ok := pod.Labels["app"]; !ok || app != "cell" {
		return
	}

	update := CellUpdate{
		Name:      pod.Name,
		Status:    "deleted",
		Namespace: pod.Namespace,
	}
	msg, _ := json.Marshal(update)
	broadcast <- msg
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Register client
	clientsMu.Lock()
	clients[ws] = true
	clientsMu.Unlock()

	log.Println("Client connected")
}

func handleMessages() {
	for {
		msg := <-broadcast
		clientsMu.Lock()
		for client := range clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("Websocket error: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
}

func handleChaos(w http.ResponseWriter, r *http.Request, clientset *kubernetes.Clientset, namespace string) {
	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")

	if r.Method == "OPTIONS" {
		return
	}

	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /api/pods/{name}
	name := filepath.Base(r.URL.Path)
	if name == "" || name == "/" {
		http.Error(w, "Pod name required", http.StatusBadRequest)
		return
	}

	log.Printf("Chaos: Deleting pod %s", name)

	err := clientset.CoreV1().Pods(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Pod deleted"))
}
