package api

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"

    "github.com/redis/go-redis/v9"
    "github.com/ManadaHerath/realtime-grid-server/internal/grid"
)

type API struct {
    Store grid.Store
    Redis *redis.Client
}

func NewAPI(store grid.Store, rdb *redis.Client) *API {
    return &API{Store: store, Redis: rdb}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func parseJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(dst)
}

type CreateGridRequest struct {
	Dimensions []int       `json:"dimensions"`
	DefaultVal interface{} `json:"defaultValue"`
}

type CreateGridResponse struct {
	ID         string `json:"id"`
	Dimensions []int  `json:"dimensions"`
}

type GetGridResponse struct {
	ID         string          `json:"id"`
	Dimensions []int           `json:"dimensions"`
	DefaultVal interface{}     `json:"defaultValue,omitempty"`
	Cells      []grid.CellView `json:"cells"`
}

type ClaimCellRequest struct {
	Coord []int       `json:"coord"`
	Value interface{} `json:"value"`
}

type ClaimCellResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type CellUpdateEvent struct {
    Type   string      `json:"type"`
    GridID string      `json:"gridId"`
    Coord  []int       `json:"coord,omitempty"`
    Value  interface{} `json:"value,omitempty"`
}

type ReleaseCellRequest struct {
    Coord []int `json:"coord"`
}

func (api *API) publishCellEvent(ctx context.Context, ev CellUpdateEvent) {
    if api.Redis == nil {
        return
    }
    data, err := json.Marshal(ev)
    if err != nil {
        return
    }
    channel := "grid:" + ev.GridID + ":events"
    api.Redis.Publish(ctx, channel, data)
}

func (api *API) publishCellUpdate(ctx context.Context, gridID string, coord []int, value interface{}) {
	if api.Redis == nil {
		return
	}

	ev := CellUpdateEvent{
		Type:   "cell_claimed",
		GridID: gridID,
		Coord:  coord,
		Value:  value,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return
	}

	channel := "grid:" + gridID + ":events"
	api.Redis.Publish(ctx, channel, data)
}

func (api *API) HandleCreateGrid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req CreateGridRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	g, err := api.Store.CreateGrid(req.Dimensions, req.DefaultVal)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp := CreateGridResponse{
		ID:         g.ID,
		Dimensions: g.Dimensions,
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (api *API) HandleGetGrid(w http.ResponseWriter, r *http.Request, gridID string) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	g, err := api.Store.GetGrid(gridID)
	if err == grid.ErrGridNotFound {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "grid not found"})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	cells, err := api.Store.ListCells(gridID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := GetGridResponse{
		ID:         g.ID,
		Dimensions: g.Dimensions,
		DefaultVal: g.DefaultVal,
		Cells:      cells,
	}

	writeJSON(w, http.StatusOK, resp)
}

func (api *API) HandleClaimCell(w http.ResponseWriter, r *http.Request, gridID string) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req ClaimCellRequest
	if err := parseJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	if len(req.Coord) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "coord required"})
		return
	}

	err := api.Store.SetCell(gridID, req.Coord, req.Value)
	if err == grid.ErrGridNotFound {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "grid not found"})
		return
	}
	if err == grid.ErrDimensionMismatch || err == grid.ErrOutOfBounds {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err == grid.ErrCellAlreadySet {
		resp := ClaimCellResponse{
			Success: false,
			Error:   "cell already set",
		}
		writeJSON(w, http.StatusConflict, resp)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}


	api.publishCellEvent(r.Context(), CellUpdateEvent{
		Type:   "cell_claimed",
		GridID: gridID,
		Coord:  req.Coord,
		Value:  req.Value,
	})

	resp := ClaimCellResponse{Success: true}
	writeJSON(w, http.StatusOK, resp)
}

func (api *API) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/grids", api.HandleCreateGrid)

    mux.HandleFunc("/grids/", func(w http.ResponseWriter, r *http.Request) {
        path := strings.TrimPrefix(r.URL.Path, "/grids/")
        parts := strings.Split(path, "/")
        if len(parts) == 0 || parts[0] == "" {
            writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing grid id"})
            return
        }
        gridID := parts[0]

        if len(parts) == 1 {
            api.HandleGetGrid(w, r, gridID)
            return
        }

        if len(parts) == 2 && parts[1] == "claim" {
            api.HandleClaimCell(w, r, gridID)
            return
        }

        if len(parts) == 2 && parts[1] == "release" {
            api.HandleReleaseCell(w, r, gridID)
            return
        }

        if len(parts) == 2 && parts[1] == "ws" {
            api.HandleGridWS(w, r, gridID)
            return
        }

        writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
    })
}

func (api *API) HandleReleaseCell(w http.ResponseWriter, r *http.Request, gridID string) {
    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
        return
    }

    var req ReleaseCellRequest
    if err := parseJSON(r, &req); err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
        return
    }

    if len(req.Coord) == 0 {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "coord required"})
        return
    }

    err := api.Store.ReleaseCell(gridID, req.Coord)
    if err == grid.ErrGridNotFound {
        writeJSON(w, http.StatusNotFound, map[string]string{"error": "grid not found"})
        return
    }
    if err == grid.ErrDimensionMismatch || err == grid.ErrOutOfBounds {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }
    if err != nil {
        writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
        return
    }

    api.publishCellEvent(r.Context(), CellUpdateEvent{
        Type:   "cell_released",
        GridID: gridID,
        Coord:  req.Coord,
    })

    writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
