import { useEffect, useMemo, useState } from "react";
import { SpaceGridScene} from "./SpaceGridScene";
import type { ClaimedCell, Coord3 } from "./SpaceGridScene";
import { createGridClient } from "../../client/dist/index.js";

const API_BASE_URL = "http://localhost:8080";

interface GridState {
  id: string;
  dimensions: number[];
  defaultValue?: any;
  cells: { coord: number[]; value: any }[];
}

type ClientType = ReturnType<typeof createGridClient>;

function App() {
  const [gridId, setGridId] = useState<string | null>(null);
  const [existingGridId, setExistingGridId] = useState("");
  const [claimedCells, setClaimedCells] = useState<Map<string, ClaimedCell>>(new Map());
  const [dimensions, setDimensions] = useState<number[] | null>(null);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [xInput, setXInput] = useState(0);
  const [yInput, setYInput] = useState(0);
  const [zInput, setZInput] = useState(0);
  const [client, setClient] = useState<ClientType | null>(null);

  const [userName, setUserName] = useState("");
  const userId = useMemo(
    () => (userName.trim() ? `user_${userName.trim()}` : `user_${Math.random().toString(16).slice(2)}`),
    [userName]
  );

  const addLog = (msg: string) => {
    setLogLines((prev) => [`[${new Date().toLocaleTimeString()}] ${msg}`, ...prev].slice(0, 50));
  };

  const claimedCellsArray = useMemo(
    () => Array.from(claimedCells.values()),
    [claimedCells]
  );

  useEffect(() => {
    return () => {
      if (client) client.disconnect();
    };
  }, [client]);
  const applyInitialState = (state: GridState) => {
    setDimensions(state.dimensions);
    const next = new Map<string, ClaimedCell>();
    (state.cells || []).forEach((cell) => {
      if (cell.coord.length >= 3) {
        const coord: Coord3 = [cell.coord[0], cell.coord[1], cell.coord[2]];
        const key = coord.join(":");
        next.set(key, { coord, value: cell.value });
      }
    });
    setClaimedCells(next);
  };

  const setupClientForGrid = async (targetGridId: string) => {
    if (!targetGridId) {
      addLog("No grid ID provided");
      return;
    }

    if (client) {
      client.disconnect();
    }

    const newClient = createGridClient({
      baseUrl: API_BASE_URL,
      gridId: targetGridId,
    });
    setClient(newClient);

    addLog(`Loading grid ${targetGridId} ...`);

    const state = await newClient.getInitialState();
    applyInitialState(state);

    await newClient.connect();
    addLog(`Connected to grid ${targetGridId}`);

    newClient.onCellUpdate((ev) => {
      if (!ev.coord || ev.coord.length < 3) return;
      const [x, y, z] = ev.coord as Coord3;
      const key = `${x}:${y}:${z}`;

      setClaimedCells((prev) => {
        const next = new Map(prev);
        if (ev.type === "cell_claimed") {
          next.set(key, { coord: [x, y, z], value: ev.value });
        } else if (ev.type === "cell_released") {
          next.delete(key);
        }
        return next;
      });

      if (ev.type === "cell_claimed") {
        addLog(`Cell [${x},${y},${z}] claimed (${String(ev.value)})`);
      } else if (ev.type === "cell_released") {
        addLog(`Cell [${x},${y},${z}] released`);
      }
    });

    setGridId(targetGridId);
  };

  const handleCreateGrid = async () => {
    try {
      const dims = [100, 100, 100];
      addLog(`Creating new 3D grid ${dims.join("x")} ...`);

      const res = await fetch(`${API_BASE_URL}/grids`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          dimensions: dims,
          defaultValue: null,
        }),
      });

      if (!res.ok) {
        addLog(`Failed to create grid: HTTP ${res.status}`);
        return;
      }

      const data = await res.json();
      const newGridId = data.id as string;

      setExistingGridId(newGridId);
      addLog(`Created grid ${newGridId}`);
      await setupClientForGrid(newGridId);
    } catch (err: any) {
      addLog(`Error: ${err.message}`);
    }
  };

  const handleConnectExisting = async () => {
    if (!existingGridId.trim()) {
      addLog("Enter a grid ID to connect");
      return;
    }
    try {
      await setupClientForGrid(existingGridId.trim());
    } catch (err: any) {
      addLog(`Error: ${err.message}`);
    }
  };

  const handleToggleCell = async () => {
    if (!client || !gridId) {
      addLog("Connect to a grid first");
      return;
    }

    const x = xInput;
    const y = yInput;
    const z = zInput;

    if (x < 0 || x >= 100 || y < 0 || y >= 100 || z < 0 || z >= 100) {
      addLog("x,y,z must be between 0 and 99");
      return;
    }

    const key = `${x}:${y}:${z}`;
    const current = claimedCells.get(key);

    if (current) {
      const result = await client.release([x, y, z]);
      if (!result.success) {
        addLog(`Failed to release [${x},${y},${z}]: ${result.error || "error"}`);
      } else {
        addLog(`Released [${x},${y},${z}]`);
      }
      return;
    }

    // else claim
    const value = `held:${userId}`;
    const result = await client.claim([x, y, z], value);
    if (!result.success) {
      addLog(`Failed to claim [${x},${y},${z}]: ${result.error || "conflict"}`);
    } else {
      addLog(`Claimed [${x},${y},${z}]`);
    }
  };

  return (
    <div className="app-root">
      <div className="control-panel">
        <h1>3D Space Grid Demo</h1>
        <p className="subtitle">Go + Redis + WebSockets + React Three Fiber</p>

        <div className="field-row">
          <label>
            Your name:
            <input
              value={userName}
              onChange={(e) => setUserName(e.target.value)}
              placeholder="e.g. Alice"
            />
          </label>
        </div>

        <div className="field-row">
          <button onClick={handleCreateGrid}>Create new 100x100x100 grid</button>
        </div>

        <div className="field-row">
          <label>
            Existing Grid ID:
            <input
              value={existingGridId}
              onChange={(e) => setExistingGridId(e.target.value)}
              placeholder="Paste grid id to join"
            />
          </label>
          <button onClick={handleConnectExisting}>Connect</button>
        </div>

        <div className="field-row">
          <span>
            Grid:{" "}
            {gridId ? <code>{gridId}</code> : <em>none</em>}
          </span>
        </div>

        <hr />

        <h2>Claim / Release a voxel</h2>
        <div className="xyz-inputs">
          <label>
            x:
            <input
              type="number"
              min={0}
              max={99}
              value={xInput}
              onChange={(e) => setXInput(Number(e.target.value))}
            />
          </label>
          <label>
            y:
            <input
              type="number"
              min={0}
              max={99}
              value={yInput}
              onChange={(e) => setYInput(Number(e.target.value))}
            />
          </label>
          <label>
            z:
            <input
              type="number"
              min={0}
              max={99}
              value={zInput}
              onChange={(e) => setZInput(Number(e.target.value))}
            />
          </label>
          <button onClick={handleToggleCell}>Toggle voxel</button>
        </div>

        <div className="log-panel">
          <h3>Log</h3>
          <div className="log-scroll">
            {logLines.map((line, idx) => (
              <div key={idx} className="log-line">
                {line}
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="scene-panel">
        <SpaceGridScene cells={claimedCellsArray} userId={userId} />
      </div>
    </div>
  );
}

export default App;
