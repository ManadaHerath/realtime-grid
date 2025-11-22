package grid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string, password string, db int) Store {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisStore{client: rdb}
}

func metaKey(gridID string) string {
	return "grid:" + gridID + ":meta"
}

func cellsKey(gridID string) string {
	return "grid:" + gridID + ":cells"
}

func (s *RedisStore) CreateGrid(dimensions []int, defaultVal interface{}) (*Grid, error) {
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
		Cells:      nil, // we don't keep all cells in memory here
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Encode dimensions as comma-separated: "10,20,3"
	dimParts := make([]string, len(dimensions))
	for i, d := range dimensions {
		dimParts[i] = fmt.Sprintf("%d", d)
	}
	dimStr := strings.Join(dimParts, ",")

	// defaultVal as JSON string (so it can be anything)
	var defStr string
	if defaultVal != nil {
		b, err := json.Marshal(defaultVal)
		if err != nil {
			return nil, err
		}
		defStr = string(b)
	}

	meta := map[string]interface{}{
		"dimensions": dimStr,
		"default":    defStr,
	}

	if err := s.client.HSet(ctx, metaKey(g.ID), meta).Err(); err != nil {
		return nil, err
	}

	return g, nil
}

// GetGrid loads grid meta from Redis.
func (s *RedisStore) GetGrid(id string) (*Grid, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vals, err := s.client.HGetAll(ctx, metaKey(id)).Result()
	if err != nil {
		return nil, err
	}
	if len(vals) == 0 {
		return nil, ErrGridNotFound
	}

	dimStr, ok := vals["dimensions"]
	if !ok || dimStr == "" {
		return nil, errors.New("invalid grid meta: missing dimensions")
	}

	dimParts := strings.Split(dimStr, ",")
	dims := make([]int, 0, len(dimParts))
	for _, p := range dimParts {
		if p == "" {
			continue
		}
		var d int
		_, err := fmt.Sscanf(p, "%d", &d)
		if err != nil {
			return nil, err
		}
		dims = append(dims, d)
	}

	var defaultVal interface{}
	if defStr, ok := vals["default"]; ok && defStr != "" {
		if err := json.Unmarshal([]byte(defStr), &defaultVal); err != nil {
			// not fatal, but log-worthy in real app
			defaultVal = nil
		}
	}

	return &Grid{
		ID:         id,
		Dimensions: dims,
		DefaultVal: defaultVal,
	}, nil
}

// SetCell atomically sets the cell value IF it is not already set.
// This enforces "only one successful claim per cell" across all clients.
func (s *RedisStore) SetCell(gridID string, coord []int, value interface{}) error {
	// First, we need the grid meta to do bounds checks.
	g, err := s.GetGrid(gridID)
	if err != nil {
		return err
	}
	if len(coord) != len(g.Dimensions) {
		return ErrDimensionMismatch
	}
	for i, c := range coord {
		if c < 0 || c >= g.Dimensions[i] {
			return ErrOutOfBounds
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := CoordKey(coord)
	cKey := cellsKey(gridID)

	// Value as JSON so we can support string/int/float/etc.
	valBytes, err := json.Marshal(value)
	if err != nil {
		return err
	}
	valStr := string(valBytes)

	// Use a Lua script for atomic "only if not exists" in the hash.
	script := redis.NewScript(`
local cKey = KEYS[1]
local field = ARGV[1]
local val = ARGV[2]

local existing = redis.call("HGET", cKey, field)
if existing ~= false and existing ~= nil then
  return 0
end

redis.call("HSET", cKey, field, val)
return 1
`)

	res, err := script.Run(ctx, s.client, []string{cKey}, key, valStr).Int()
	if err != nil {
		return err
	}
	if res == 0 {
		return ErrCellAlreadySet
	}

	return nil
}

// ListCells returns all explicitly set cells.
func (s *RedisStore) ListCells(gridID string) ([]CellView, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check grid exists
	_, err := s.GetGrid(gridID)
	if err != nil {
		return nil, err
	}

	entries, err := s.client.HGetAll(ctx, cellsKey(gridID)).Result()
	if err != nil {
		return nil, err
	}

	cells := make([]CellView, 0, len(entries))
	for k, v := range entries {
		coord, err := ParseCoordKey(k)
		if err != nil {
			continue
		}

		var val interface{}
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			// if fail, keep raw string
			val = v
		}

		cells = append(cells, CellView{
			Coord: coord,
			Value: val,
		})
	}
	return cells, nil
}

func (s *RedisStore) ReleaseCell(gridID string, coord []int) error {
    // Get grid meta for bounds check
    g, err := s.GetGrid(gridID)
    if err != nil {
        return err
    }
    if len(coord) != len(g.Dimensions) {
        return ErrDimensionMismatch
    }
    for i, c := range coord {
        if c < 0 || c >= g.Dimensions[i] {
            return ErrOutOfBounds
        }
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    key := CoordKey(coord)
    cKey := cellsKey(gridID)

    // Just delete the field. If it doesn't exist, that's fine.
    if err := s.client.HDel(ctx, cKey, key).Err(); err != nil {
        return err
    }

    return nil
}
