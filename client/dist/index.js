export function createGridClient(options) {
    const { baseUrl, gridId } = options;
    const normalizedBase = baseUrl.replace(/\/+$/, "");
    const httpBase = normalizedBase;
    const wsBase = normalizedBase.replace(/^http/, "ws");
    let ws = null;
    const listeners = new Set();
    function notifyListeners(ev) {
        for (const l of listeners) {
            try {
                l(ev);
            }
            catch (err) {
                console.error("GridClient listener error:", err);
            }
        }
    }
    async function getInitialState() {
        const res = await fetch(`${httpBase}/grids/${encodeURIComponent(gridId)}`);
        if (!res.ok) {
            throw new Error(`Failed to get grid: ${res.status} ${res.statusText}`);
        }
        const data = await res.json();
        const cells = (data.cells || []).map((c) => ({
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
    async function connect() {
        if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
            return;
        }
        const wsUrl = `${wsBase}/grids/${encodeURIComponent(gridId)}/ws`;
        ws = new WebSocket(wsUrl);
        await new Promise((resolve, reject) => {
            if (!ws) {
                reject(new Error("WebSocket not created"));
                return;
            }
            const handleOpen = () => {
                if (!ws)
                    return;
                ws.removeEventListener("open", handleOpen);
                ws.removeEventListener("error", handleError);
                resolve();
            };
            const handleError = (ev) => {
                if (!ws)
                    return;
                ws.removeEventListener("open", handleOpen);
                ws.removeEventListener("error", handleError);
                reject(new Error("WebSocket connection error"));
            };
            ws.addEventListener("open", handleOpen);
            ws.addEventListener("error", handleError);
        });
        if (!ws)
            return;
        ws.onmessage = (ev) => {
            try {
                const data = JSON.parse(ev.data);
                notifyListeners(data);
            }
            catch (err) {
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
    function isConnected() {
        return !!ws && ws.readyState === WebSocket.OPEN;
    }
    async function claim(coord, value) {
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
    async function release(coord) {
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
    function onCellUpdate(listener) {
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
