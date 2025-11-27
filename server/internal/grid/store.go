package grid

import (
	"errors"
	"sync"
)

var (
	ErrGridNotFound      = errors.New("grid not found")
	ErrDimensionMismatch = errors.New("coord dimension mismatch")
	ErrOutOfBounds       = errors.New("coord out of bounds")
	ErrCellAlreadySet    = errors.New("cell already set") 
)
type Store interface {
    CreateGrid(dimensions []int, defaultVal interface{}) (*Grid, error)
    GetGrid(id string) (*Grid, error)
    SetCell(gridID string, coord []int, value interface{}) error
    ListCells(gridID string) ([]CellView, error)
    ReleaseCell(gridID string, coord []int) error 
}

type MemStore struct {
	mu    sync.RWMutex
	grids map[string]*Grid
}

func NewMemStore() Store {
	return &MemStore{
		grids: make(map[string]*Grid),
	}
}

func (s *MemStore) ReleaseCell(gridID string, coord []int) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    g, ok := s.grids[gridID]
    if !ok {
        return ErrGridNotFound
    }

    if len(coord) != len(g.Dimensions) {
        return ErrDimensionMismatch
    }

    for i, c := range coord {
        if c < 0 || c >= g.Dimensions[i] {
            return ErrOutOfBounds
        }
    }

    key := CoordKey(coord)
    delete(g.Cells, key)
    return nil
}

func (s *MemStore) CreateGrid(dimensions []int, defaultVal interface{}) (*Grid, error) {
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

func (s *MemStore) GetGrid(id string) (*Grid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.grids[id]
	if !ok {
		return nil, ErrGridNotFound
	}
	return g, nil
}

func (s *MemStore) SetCell(gridID string, coord []int, value interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grids[gridID]
	if !ok {
		return ErrGridNotFound
	}

	if len(coord) != len(g.Dimensions) {
		return ErrDimensionMismatch
	}


	for i, c := range coord {
		if c < 0 || c >= g.Dimensions[i] {
			return ErrOutOfBounds
		}
	}

	key := CoordKey(coord)

	if _, exists := g.Cells[key]; exists {
		return ErrCellAlreadySet
	}

	g.Cells[key] = value
	return nil
}

func (s *MemStore) ListCells(gridID string) ([]CellView, error) {
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
