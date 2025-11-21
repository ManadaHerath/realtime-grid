package grid

import (
	"errors"
	"sync"
)

var (
	ErrGridNotFound      = errors.New("grid not found")
	ErrDimensionMismatch = errors.New("coord dimension mismatch")
	ErrOutOfBounds       = errors.New("coord out of bounds")
)

// Store is a simple in-memory grid store (later replaced with Redis).
type Store struct {
	mu    sync.RWMutex
	grids map[string]*Grid
}

func NewStore() *Store {
	return &Store{
		grids: make(map[string]*Grid),
	}
}

// CreateGrid creates a new grid with given dimensions & default value.
func (s *Store) CreateGrid(dimensions []int, defaultVal interface{}) (*Grid, error) {
	if len(dimensions) == 0 {
		return nil, errors.New("dimensions required")
	}
	for _, d := range dimensions {
		if d <= 0 {
			return nil, errors.New("dimensions must be > 0")
		}
	}

	g := &Grid{
		ID:         GenerateID(),
		Dimensions: dimensions,
		DefaultVal: defaultVal,
		Cells:      make(map[string]interface{}),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.grids[g.ID] = g
	return g, nil
}

// GetGrid returns a grid by ID.
func (s *Store) GetGrid(id string) (*Grid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.grids[id]
	if !ok {
		return nil, ErrGridNotFound
	}
	return g, nil
}

// SetCell sets a value for a coord (for now: last write wins).
func (s *Store) SetCell(gridID string, coord []int, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grids[gridID]
	if !ok {
		return ErrGridNotFound
	}

	if len(coord) != len(g.Dimensions) {
		return ErrDimensionMismatch
	}

	// Bounds check
	for i, c := range coord {
		if c < 0 || c >= g.Dimensions[i] {
			return ErrOutOfBounds
		}
	}

	key := CoordKey(coord)
	g.Cells[key] = value
	return nil
}

// ListCells returns all cells that have explicit values set.
func (s *Store) ListCells(gridID string) ([]CellView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.grids[gridID]
	if !ok {
		return nil, ErrGridNotFound
	}

	cells := make([]CellView, 0, len(g.Cells))
	for k, v := range g.Cells {
		coord, err := ParseCoordKey(k)
		if err != nil {
			continue
		}
		cells = append(cells, CellView{
			Coord: coord,
			Value: v,
		})
	}
	return cells, nil
}
