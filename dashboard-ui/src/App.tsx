import { useEffect, useState } from 'react';
import './App.css';

interface Cell {
  name: string;
  status: 'alive' | 'dead' | 'initializing' | 'terminating' | 'deleted' | 'unknown';
  namespace: string;
}

function App() {
  const [cells, setCells] = useState<Map<string, Cell>>(new Map());
  const [gridSize] = useState(10); // 10x10 hardcoded for now

  useEffect(() => {
    // WebSocket Connection
    // In dev: localhost:8080 (via proxy or CORS?) - user probably uses kubectl port-forward
    // In k8s: Ingress path

    // For local dev with vite proxy, we might need a setup. 
    // Let's assume relative path /ws if served by Go, or localhost:8080 if standalone
    const wsUrl = window.location.hostname === 'localhost'
      ? 'ws://localhost:8080/ws'
      : `ws://${window.location.host}/ws`;

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('Connected to Controller WS');
    };

    ws.onmessage = (event) => {
      try {
        const update = JSON.parse(event.data); // {name, status, namespace}
        setCells(prev => {
          const next = new Map(prev);
          next.set(update.name, update);
          return next;
        });
      } catch (e) {
        console.error('Failed to parse WS message', e);
      }
    };

    ws.onclose = () => {
      console.log('Disconnected from Controller WS');
    };

    return () => {
      ws.close();
    };
  }, []);

  const killPod = async (name: string) => {
    if (!confirm(`Kill pod ${name}?`)) return;

    try {
      const apiUrl = window.location.hostname === 'localhost'
        ? `http://localhost:8080/api/pods/${name}`
        : `/api/pods/${name}`;

      await fetch(apiUrl, { method: 'DELETE' });
    } catch (e) {
      console.error('Failed to kill pod', e);
      alert('Failed to kill pod');
    }
  };

  // Render Grid
  // We assume names are cell-0, cell-1...

  const renderCell = (index: number) => {
    const name = `cell-${index}`;
    const cell = cells.get(name);

    let color = 'bg-gray-200'; // Default/Unknown
    let statusText = '...';

    if (cell) {
      statusText = cell.status;
      switch (cell.status) {
        case 'alive': color = 'bg-green-500'; break;
        case 'dead': color = 'bg-gray-800'; break;
        case 'initializing': color = 'bg-blue-300 animate-pulse'; break;
        case 'terminating': color = 'bg-red-500 animate-pulse'; break;
        case 'deleted': color = 'bg-red-900 border-red-500 border-2'; break;
        default: color = 'bg-gray-400';
      }
    }

    return (
      <div
        key={index}
        className={`w-16 h-16 m-1 rounded flex items-center justify-center text-xs text-white font-mono cursor-pointer transition-colors duration-200 ${color}`}
        onClick={() => cell && killPod(name)}
        title={name}
      >
        {statusText}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gray-900 flex flex-col items-center justify-center p-4">
      <h1 className="text-3xl font-bold text-white mb-8">Cellular Automaton Control</h1>
      <div className="grid grid-cols-10 gap-2 p-4 bg-gray-800 rounded-lg shadow-xl">
        {Array.from({ length: gridSize * gridSize }).map((_, i) => renderCell(i))}
      </div>
      <div className="mt-8 text-gray-400">
        <p>Click a cell to kill its pod (Chaos Monkey).</p>
        <p>Green: Alive | Black: Dead | Blue: Init | Red: Terminating</p>
      </div>
    </div>
  );
}

export default App;
