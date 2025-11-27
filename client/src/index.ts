export type Coord = number[];

export interface GridCell {
  coord: Coord;
  value: any;
}

export interface GridState {
  id: string;
  dimensions: number[];
  defaultValue?: any;
  cells: GridCell[];
}

export interface ClaimResult {
  success: boolean;
  error?: string;
}

export interface CellUpdateEvent {
  type: string;
  gridId: string;
  coord?: Coord;
  value?: any;
  [key: string]: any;
}

export interface GridClientOptions {
  baseUrl: string;
  gridId: string;
  userId?: string;
}

export interface GridClient {
  getInitialState(): Promise<GridState>;
  connect(): Promise<void>;
  disconnect(): void;
  isConnected(): boolean;
  claim(coord: Coord, value: any): Promise<ClaimResult>;
  release(coord: Coord): Promise<ClaimResult>;           // NEW
  onCellUpdate(listener: (ev: CellUpdateEvent) => void): () => void;
}

export function createGridClient(options: GridClientOptions): GridClient {
  const { baseUrl, gridId } = options;

  const normalizedBase = baseUrl.replace(/\/+$/, "");
  const httpBase = normalizedBase;
  const wsBase = normalizedBase.replace(/^http/, "ws");

  let ws: WebSocket | null = null;
  const listeners = new Set<(ev: CellUpdateEvent) => void>();

  function notifyListeners(ev: CellUpdateEvent) {
    for (const l of listeners) {
      try {
        l(ev);
      } catch (err) {
        console.error("GridClient listener error:", err);
      }
    }
  }

  async function getInitialState(): Promise<GridState> {
    const res = await fetch(`${httpBase}/grids/${encodeURIComponent(gridId)}`);
    if (!res.ok) {
      throw new Error(`Failed to get grid: ${res.status} ${res.statusText}`);
    }
    const data = await res.json();
    const cells: GridCell[] = (data.cells || []).map((c: any) => ({
      coord: c.coord,
      value: c.value,
    }));
    return {
      id: data.id,
      dimensions: data.dimensions,
      defaultValue: data.defaultValue,
      cells,
    };
  }

  async function connect(): Promise<void> {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    const wsUrl = `${wsBase}/grids/${encodeURIComponent(gridId)}/ws`;

    ws = new WebSocket(wsUrl);

    await new Promise<void>((resolve, reject) => {
      if (!ws) {
        reject(new Error("WebSocket not created"));
        return;
      }

      const handleOpen = () => {
        if (!ws) return;
        ws.removeEventListener("open", handleOpen);
        ws.removeEventListener("error", handleError);
        resolve();
      };

      const handleError = (ev: Event) => {
        if (!ws) return;
        ws.removeEventListener("open", handleOpen);
        ws.removeEventListener("error", handleError);
        reject(new Error("WebSocket connection error"));
      };

      ws.addEventListener("open", handleOpen);
      ws.addEventListener("error", handleError);
    });

    if (!ws) return;

    ws.onmessage = (ev: MessageEvent) => {
      try {
        const data = JSON.parse(ev.data as string);
        notifyListeners(data);
      } catch (err) {
        console.error("Failed to parse WS message:", err);
      }
    };

    ws.onclose = () => {
      ws = null;
    };
  }

  function disconnect() {
    if (ws) {
      ws.close();
      ws = null;
    }
  }

  function isConnected(): boolean {
    return !!ws && ws.readyState === WebSocket.OPEN;
  }

  async function claim(coord: Coord, value: any): Promise<ClaimResult> {
    const res = await fetch(`${httpBase}/grids/${encodeURIComponent(gridId)}/claim`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        coord,
        value,
      }),
    });

    const data = await res.json().catch(() => ({}));

    if (!res.ok) {
      return {
        success: false,
        error: data.error || `HTTP ${res.status}`,
      };
    }

    return {
      success: !!data.success,
      error: data.error,
    };
  }

  async function release(coord: Coord): Promise<ClaimResult> {
  const res = await fetch(`${httpBase}/grids/${encodeURIComponent(gridId)}/release`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ coord }),
  });

  const data = await res.json().catch(() => ({}));

  if (!res.ok) {
    return {
      success: false,
      error: data.error || `HTTP ${res.status}`,
    };
  }

  return {
    success: true,
  };
}

  function onCellUpdate(listener: (ev: CellUpdateEvent) => void): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  }

  return {
    getInitialState,
    connect,
    disconnect,
    isConnected,
    claim,
    release,
    onCellUpdate,
  };
}
