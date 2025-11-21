package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ManadaHerath/realtime-grid-server/internal/grid"
)

// API holds dependencies for HTTP handlers.
type API struct {
	Store *grid.Store
}

// NewAPI creates an API with the given store.
func NewAPI(store *grid.Store) *API {
	return &API{Store: store}
}

// ===== Helper functions =====

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

// ===== Request/response DTOs =====

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

// ===== Handlers =====

// POST /grids
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

// GET /grids/{id}
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

// POST /grids/{id}/claim
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
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	resp := ClaimCellResponse{Success: true}
	writeJSON(w, http.StatusOK, resp)
}

// Router for /grids and /grids/{id}...
func (api *API) RegisterRoutes(mux *http.ServeMux) {
	// POST /grids
	mux.HandleFunc("/grids", api.HandleCreateGrid)

	// /grids/{id} and /grids/{id}/claim
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

		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	})
}
