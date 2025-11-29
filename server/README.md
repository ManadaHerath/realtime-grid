# Realtime Grid Server

Backend server for the Realtime Grid Engine.

This service exposes an HTTP + WebSocket API over a Redis-backed N-dimensional grid. It is designed to be used together with the `realtime-grid-client` npm package, but can also be consumed directly over HTTP/WebSocket.

---

## Features

- N-dimensional grid definition (1D, 2D, 3D, ND)
- Atomic cell claim/release using Redis
- First-claim-wins concurrency model
- Real-time events via WebSockets (Pub/Sub)
- Designed to be horizontally scalable

---

## Requirements

- **Go 1.21+** (if running from source)  
- **Redis 6+**  
- **Docker** (optional, for containerized usage)

### Environment Variables

- `REDIS_ADDR` (default: `localhost:6379`)
- `REDIS_PASSWORD` (default: empty)
- `REDIS_DB` (optional, usually default 0)

---

## Running Locally (Go + Redis)

### 1. Start Redis

You can use Docker or a local Redis install.

Using Docker:

```bash
docker run -d --name redis-grid -p 6379:6379 redis:7
```

### 2. Run server from source

Clone the repo:

```bash
git clone https://github.com/ManadaHerath/realtime-grid.git
cd realtime-grid/server
```

Install dependencies:

```bash
go mod tidy
```

Run:

```bash
go run ./cmd/api
```

By default this will:
- Connect to Redis at `localhost:6379`
- Start HTTP server on `:8080`

You can override Redis connection:

```bash
REDIS_ADDR=localhost:6379 REDIS_PASSWORD=somepassword go run ./cmd/api
```

---

## Docker Usage

### 1. Build the image

From the `server/` directory:

```bash
docker build -t manadaherath/realtime-grid-api .
```

### 2. Run with Docker (using host Redis)

If you already have Redis running on your host at `localhost:6379`:

**On macOS/Windows:**

```bash
docker run --rm \
  -p 8080:8080 \
  -e REDIS_ADDR=host.docker.internal:6379 \
  manadaherath/realtime-grid-api
```

**On Linux** (add host mapping):

```bash
docker run --rm \
  -p 8080:8080 \
  --add-host=host.docker.internal:host-gateway \
  -e REDIS_ADDR=host.docker.internal:6379 \
  manadaherath/realtime-grid-api
```

---

## Docker Compose (Server + Redis)

For a fully self-contained setup,

```bash
docker compose up --build
```

This will bring up:
- **Redis** at `redis:6379` (and on host `localhost:6379`)
- **API** at `http://localhost:8080`

---

## API Overview

### Create a grid

**POST** `/grids`

Request body:

```json
{
  "dimensions": [100, 100],
  "defaultValue": null
}
```

Response:

```json
{
  "id": "g_abc123...",
  "dimensions": [100, 100],
  "defaultValue": null
}
```

### Get grid state

**GET** `/grids/:id`

Response:

```json
{
  "id": "g_abc123...",
  "dimensions": [100, 100],
  "defaultValue": null,
  "cells": [
    { "coord": [1, 2], "value": "held:user123" }
  ]
}
```

### Claim a cell

**POST** `/grids/:id/claim`

Body:

```json
{
  "coord": [1, 2],
  "value": "held:user123"
}
```

Response on success:

```json
{ "success": true }
```

If cell is already set:

```json
{
  "success": false,
  "error": "cell already set"
}
```

### Release a cell

**POST** `/grids/:id/release`

Body:

```json
{
  "coord": [1, 2]
}
```

Response:

```json
{ "success": true }
```

### WebSocket events

**GET** `/grids/:id/ws`

Messages:

```json
{ "type": "hello", "gridId": "g_abc123..." }
{ "type": "cell_claimed", "gridId": "g_abc123...", "coord": [1, 2], "value": "held:user123" }
{ "type": "cell_released", "gridId": "g_abc123...", "coord": [1, 2] }
```

---

## Using with the NPM Client

Install the client:

```bash
npm install realtime-grid-client
```

Example:

```ts
import { createGridClient } from "realtime-grid-client";

const baseUrl = "http://localhost:8080";

// gridId is obtained by POST /grids or created by the backend (you can change the backend logic of the server as you like)
const client = createGridClient({ baseUrl, gridId: "g_abc123..." });

async function main() {
  const state = await client.getInitialState();
  console.log("Initial:", state);

  await client.connect();

  client.onCellUpdate((ev) => {
    console.log("Update:", ev);
  });

  await client.claim([1, 2], "held:user1");
}
```
