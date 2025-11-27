# Realtime Grid Client

A TypeScript/JavaScript client library for interacting with a real-time, Redis-backed N-dimensional grid engine. This client provides atomic cell claiming, deterministic conflict resolution, and real-time synchronization through WebSockets. It is designed to be fully domain-agnostic and compatible with the Realtime Grid Server (Go + Redis).



## Features

- Real-time updates using WebSockets  
- Atomic first-claim-wins cell operations  
- Conflict-free concurrency across multiple clients  
- Supports 1D, 2D, 3D, and arbitrary N-dimensional grids  
- Lightweight and minimal API surface  
- Fully typed with TypeScript definitions  
- Works in browsers and Node.js (18+)  



## Installation

```bash
npm install realtime-grid-client
```

## Server Implementation

This client expects a compatible Realtime Grid Server.

**Reference implementation (Go + Redis):**  
📦 [GitHub Repository](https://github.com/ManadaHerath/realtime-grid/tree/main/server)

The server README includes instructions for:
- Running locally with Go and Redis
- Running via Docker
- Running via Docker Compose (server + Redis included)

---

## Quick Start

```ts
import { createGridClient } from "realtime-grid-client";

// Initialize client
const client = createGridClient({
  baseUrl: "http://localhost:8080",
  gridId: "g_abc123"
});

// Fetch initial state
const state = await client.getInitialState();
console.log("Dimensions:", state.dimensions);

// Connect to real-time updates
await client.connect();
client.onCellUpdate((event) => {
  console.log("Update:", event);
});

// Claim a cell
const result = await client.claim([2, 5], "held:user123");
if (!result.success) {
  console.log("Claim failed:", result.error);
}

// Release a cell
await client.release([2, 5]);

// Disconnect
client.disconnect();
```

### Creating a Grid

Grids are created via the server API:

```ts
const res = await fetch("http://localhost:8080/grids", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    dimensions: [100, 100, 100],
    defaultValue: null
  })
});

const { id: gridId } = await res.json();
const client = createGridClient({ baseUrl: "http://localhost:8080", gridId });
```



## API Reference

### `createGridClient(options)`

```ts
interface Options {
  baseUrl: string;   // Server URL (e.g., "http://localhost:8080")
  gridId: string;    // Grid identifier returned from POST /grids
}
```

**Returns:**

| Method | Description |
|--------|-------------|
| `getInitialState()` | Fetch current grid state from server |
| `connect()` | Connect to WebSocket for real-time updates |
| `disconnect()` | Close WebSocket connection |
| `isConnected()` | Check WebSocket connection status |
| `claim(coord, value)` | Atomically claim a cell (first-wins) |
| `release(coord)` | Release a claimed cell |
| `onCellUpdate(handler)` | Subscribe to real-time events |

### Types

```ts
interface GridState {
  id: string;
  dimensions: number[];
  defaultValue: any;
  cells: { coord: number[]; value: any }[];
}

interface ClaimResult {
  success: boolean;
  error?: string;
}

type GridEvent =
  | { type: "hello"; gridId: string }
  | { type: "cell_claimed"; gridId: string; coord: number[]; value: any }
  | { type: "cell_released"; gridId: string; coord: number[] };
```

---

## How It Works

- **Atomic operations**: Cell claims use Redis for race-free concurrency
- **First-claim-wins**: Only the first client to claim a cell succeeds
- **Real-time sync**: WebSocket broadcasts propagate updates to all clients
- **Minimal overhead**: Only changes are transmitted, not full grid state
- **Server authority**: No client-side caching; server is source of truth



## Architecture

```
Client (Browser / Node)
    |
    | HTTP: state fetch, claim, release
    |
Realtime Grid Server (Go)
    |
    | Redis (atomic ops + pub/sub)
    |
Other clients receive update stream
```