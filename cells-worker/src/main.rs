use std::env;
use std::sync::{Arc, Mutex};
use std::time::Duration;
use tokio::time;
use tonic::{transport::Server, Request, Response, Status};
use kube::{Client, Api, api::{Patch, PatchParams}};
use k8s_openapi::api::core::v1::Pod;
use serde_json::json;

pub mod cell {
    tonic::include_proto!("cell");
}

use cell::cell_service_server::{CellService, CellServiceServer};
use cell::{Empty, Status as CellStatus};
use cell::cell_service_client::CellServiceClient;

#[derive(Debug, Clone)]
struct CellState {
    alive: bool,
    generation: i32,
}

#[derive(Debug)]
struct MyCell {
    state: Arc<Mutex<CellState>>,
}

#[tonic::async_trait]
impl CellService for MyCell {
    async fn get_status(&self, _request: Request<Empty>) -> Result<Response<CellStatus>, Status> {
        let state = self.state.lock().unwrap();
        Ok(Response::new(CellStatus {
            alive: state.alive,
            generation: state.generation,
        }))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Cell Worker Starting...");

    // 1. Identity
    let hostname = env::var("HOSTNAME").unwrap_or_else(|_| "cell-0".to_string());
    // hostnames are like cell-0, cell-1...
    let id_str = hostname.strip_prefix("cell-").unwrap_or("0");
    let id: i32 = id_str.parse().unwrap_or(0);
    
    // Grid Configuration
    let width_str = env::var("GRID_WIDTH").unwrap_or_else(|_| "10".to_string());
    let width: i32 = width_str.parse().unwrap_or(10);
    
    let interval_ms_str = env::var("TICK_INTERVAL_MS").unwrap_or_else(|_| "1000".to_string());
    let interval_ms: u64 = interval_ms_str.parse().unwrap_or(1000);

    let x = id % width;
    let y = id / width;

    println!("Identity: ID={}, X={}, Y={} (Grid: Width={}, Interval={}ms)", id, x, y, width, interval_ms);

    // Initial State: Random or based on ID?
    // Let's make even IDs alive for initial entropy
    let initial_alive = id % 2 == 0;
    
    let state = Arc::new(Mutex::new(CellState {
        alive: initial_alive,
        generation: 0,
    }));

    // 2. Start gRPC Server
    let addr = "0.0.0.0:50051".parse()?;
    let cell_service = MyCell { state: state.clone() };

    println!("Starting gRPC server on {}", addr);
    
    let _server_handle = tokio::spawn(async move {
        Server::builder()
            .add_service(CellServiceServer::new(cell_service))
            .serve(addr)
            .await
            .unwrap();
    });

    // 3. Kubernetes Client
    // In-cluster config is assumed
    let client = Client::try_default().await; 
    let namespace = env::var("NAMESPACE").unwrap_or_else(|_| "cellular-automaton".to_string());

    // 4. Game Loop
    let mut interval = time::interval(Duration::from_millis(interval_ms));
    
    loop {
        interval.tick().await;

        // SKIP gathering if client failed to init (e.g. running locally without k8s context)
        // But for production logic we assume it works.
        
        // Calculate neighbors
        let neighbors = get_neighbors(id, width); 
        let mut alive_neighbors = 0;

        for neighbor_id in neighbors {
            // Address: cell-{id}.cell.cellular-automaton.svc.cluster.local:50051
            // Use short DNS: cell-{id}.cell
            let url = format!("http://cell-{}.cell.{}.svc.cluster.local:50051", neighbor_id, namespace);
            
            // Connect with timeout
            if let Ok(mut client) = CellServiceClient::connect(url.clone()).await {
                 let request = tonic::Request::new(Empty {});
                 if let Ok(response) = client.get_status(request).await {
                     if response.into_inner().alive {
                         alive_neighbors += 1;
                     }
                 }
            }
            // If unreachable, count as dead/0
        }

        // Apply Rules
        let mut s = state.lock().unwrap();
        let was_alive = s.alive;
        
        let next_alive = if was_alive {
            alive_neighbors == 2 || alive_neighbors == 3
        } else {
            alive_neighbors == 3
        };

        s.alive = next_alive;
        s.generation += 1;
        let gen = s.generation;
        let is_alive = s.alive;
        drop(s); // release lock

        println!("Tick {}: Alive={}, Neighbors={} -> Next={}", gen, was_alive, alive_neighbors, is_alive);

        // Update K8s Label
        if let Ok(c) = &client {
            let pods: Api<Pod> = Api::namespaced(c.clone(), &namespace);
            let patch = json!({
                "metadata": {
                    "labels": {
                        "game-status": if is_alive { "alive" } else { "dead" }
                    }
                }
            });
            let pp = PatchParams::default();
            let _ = pods.patch(&hostname, &pp, &Patch::Merge(&patch)).await;
        }
    }
}

fn get_neighbors(id: i32, width: i32) -> Vec<i32> {
    let size = width * width; // Assuming 10x10 = 100 total
    let x = id % width;
    let y = id / width;
    
    let mut neighbors = Vec::new();

    for dy in -1..=1 {
        for dx in -1..=1 {
            if dx == 0 && dy == 0 { continue; }
            
            let nx = x + dx;
            let ny = y + dy;

            if nx >= 0 && nx < width && ny >= 0 && ny < width { // No wrapping for now
                let nid = ny * width + nx;
                if nid >= 0 && nid < size {
                    neighbors.push(nid);
                }
            }
        }
    }
    neighbors
}
