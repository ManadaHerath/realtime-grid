import { createGridClient } from "../client/dist/index.js"; // ESM build

const baseUrl = "http://localhost:8080";

const rowsInput = document.getElementById("rowsInput");
const colsInput = document.getElementById("colsInput");
const userNameInput = document.getElementById("userNameInput");
const createGridBtn = document.getElementById("createGridBtn");

const gridIdInput = document.getElementById("gridIdInput");
const connectGridBtn = document.getElementById("connectGridBtn");

const gridIdLabel = document.getElementById("gridIdLabel");
const connectionStatus = document.getElementById("connectionStatus");
const gridContainer = document.getElementById("gridContainer");
const logEl = document.getElementById("log");

console.log("Demo script loaded");

let gridId = null;
let client = null;
let dimensions = [0, 0];
let userId = `user_${Math.random().toString(16).slice(2)}`;
// coordKey -> value
const cellState = new Map();

function log(message) {
  const line = document.createElement("div");
  line.className = "log-line";
  const time = new Date().toLocaleTimeString();
  line.innerHTML = `<span class="time">[${time}]</span> ${message}`;
  logEl.prepend(line);
}

function setConnected(isConnected) {
  connectionStatus.textContent = isConnected ? "Connected" : "Disconnected";
  connectionStatus.classList.toggle("status-connected", isConnected);
  connectionStatus.classList.toggle("status-disconnected", !isConnected);
}

function renderGrid(rows, cols) {
  gridContainer.innerHTML = "";
  gridContainer.style.setProperty("--cols", String(cols));

  for (let r = 0; r < rows; r++) {
    for (let c = 0; c < cols; c++) {
      const cell = document.createElement("div");
      cell.className = "cell cell-free";
      cell.dataset.row = String(r);
      cell.dataset.col = String(c);
      cell.title = `(${r}, ${c})`;

      cell.addEventListener("click", () => handleCellClick(r, c));

      gridContainer.appendChild(cell);
    }
  }
}


function coordKey(r, c) {
  return `${r}:${c}`;
}


function updateCellClass(r, c) {
  const key = coordKey(r, c);
  const value = cellState.get(key) || null;
  const cell = gridContainer.querySelector(`.cell[data-row="${r}"][data-col="${c}"]`);
  if (!cell) return;

  cell.classList.remove("cell-free", "cell-held-me", "cell-held-other");

  if (!value) {
    cell.classList.add("cell-free");
    return;
  }

  if (typeof value === "string" && value.startsWith("held:")) {
    const holder = value.slice("held:".length);
    if (holder === userId) {
      cell.classList.add("cell-held-me");
    } else {
      cell.classList.add("cell-held-other");
    }
    return;
  }
  cell.classList.add("cell-held-other");
}

async function handleCellClick(r, c) {
  if (!client || !gridId) return;
  const key = coordKey(r, c);
  const current = cellState.get(key) || null;

  if (!current) {
    const value = `held:${userId}`;
    const result = await client.claim([r, c], value);
    if (!result.success) {
      log(`Failed to claim (${r}, ${c}): ${result.error || "conflict"}`);
    } else {
      log(`Claimed (${r}, ${c})`);
      cellState.set(key, value);
      updateCellClass(r, c);
    }
    return;
  }

  if (typeof current === "string" && current.startsWith("held:") && current.slice("held:".length) === userId) {
    const result = await client.release([r, c]);
    if (!result.success) {
      log(`⚠️ Failed to release (${r}, ${c}): ${result.error || "error"}`);
    } else {
      log(`🔓 Released (${r}, ${c})`);
      cellState.delete(key);
      updateCellClass(r, c);
    }
    return;
  }

  log(`Cell (${r}, ${c}) is held by another user.`);
}


function applyInitialState(state) {
  dimensions = state.dimensions;
  cellState.clear();

  (state.cells || []).forEach((cell) => {
    const [r, c] = cell.coord;
    cellState.set(coordKey(r, c), cell.value);
  });

  const [rows, cols] = dimensions;
  renderGrid(rows, cols);

  for (const [key, value] of cellState.entries()) {
    const [rStr, cStr] = key.split(":");
    const r = parseInt(rStr, 10);
    const c = parseInt(cStr, 10);
    updateCellClass(r, c);
  }
}


function handleRealtimeUpdate(ev) {
  if (ev.type === "hello") {
    log("Connected to grid " + ev.gridId);
    setConnected(true);
    return;
  }

  if (!ev.coord || ev.coord.length !== 2) return;

  const [r, c] = ev.coord;
  const key = coordKey(r, c);

  if (ev.type === "cell_claimed") {
    cellState.set(key, ev.value);
    updateCellClass(r, c);
    log(`Seat (${r}, ${c}) claimed (${String(ev.value)})`);
  } else if (ev.type === "cell_released") {
    cellState.delete(key);
    updateCellClass(r, c);
    log(`Seat (${r}, ${c}) released`);
  }
}

/**
 * Core helper: set up client for an existing grid ID.
 */
async function setupClientForGrid(targetGridId) {
  if (!targetGridId) {
    log("No grid ID provided.");
    return;
  }

  gridId = targetGridId;
  gridIdLabel.innerHTML = `Grid ID: <code>${gridId}</code>`;
  gridIdInput.value = gridId;

  if (client) {
    client.disconnect();
  }

  client = createGridClient({
    baseUrl,
    gridId,
  });

  setConnected(false);
  log(`🔄 Loading grid ${gridId} ...`);

  const state = await client.getInitialState();
  applyInitialState(state);

  await client.connect();
  setConnected(true);

  client.onCellUpdate(handleRealtimeUpdate);

  log(`Connected to grid ${gridId}`);
}

/**
 * Create a new grid via API, then setup client for it.
 */
async function createGrid() {
  const rows = parseInt(rowsInput.value, 10) || 100;
  const cols = parseInt(colsInput.value, 10) || 100;
  const userName = userNameInput.value.trim();
  if (userName) {
    userId = `user_${userName}`;
  }

  log(`Creating ${rows} x ${cols} grid...`);

  const res = await fetch(`${baseUrl}/grids`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      dimensions: [rows, cols],
      defaultValue: null,
    }),
  });

  if (!res.ok) {
    log(`Failed to create grid: HTTP ${res.status}`);
    return;
  }

  const data = await res.json();
  const newGridId = data.id;
  log(`Created grid ${newGridId}`);

  gridIdInput.value = newGridId;

  await setupClientForGrid(newGridId);
}


async function connectToExistingGrid() {
  const userName = userNameInput.value.trim();
  if (userName) {
    userId = `user_${userName}`;
  }

  const existingId = gridIdInput.value.trim();
  if (!existingId) {
    log("Please enter a grid ID to connect.");
    return;
  }

  try {
    await setupClientForGrid(existingId);
  } catch (err) {
    console.error(err);
    log(`Error connecting: ${err.message}`);
  }
}

createGridBtn.addEventListener("click", () => {
  console.log("Create Grid button clicked");
  createGrid().catch((err) => {
    console.error(err);
    log(`Error: ${err.message}`);
  });
});

connectGridBtn.addEventListener("click", () => {
  console.log("Connect to Grid button clicked");
  connectToExistingGrid().catch((err) => {
    console.error(err);
    log(`Error: ${err.message}`);
  });
});

window.addEventListener("beforeunload", () => {
  if (client) {
    client.disconnect();
  }
});
