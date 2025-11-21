package grid

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"strings"
)

// Grid represents an N-dimensional grid.
type Grid struct {
	ID         string                 `json:"id"`
	Dimensions []int                  `json:"dimensions"`
	DefaultVal interface{}            `json:"defaultValue,omitempty"`
	Cells      map[string]interface{} `json:"-"` // coordKey -> value
}

// CellView is how we expose cells via the API (coord as []int).
type CellView struct {
	Coord []int      `json:"coord"`
	Value interface{} `json:"value"`
}

// GenerateID creates a random grid ID.
func GenerateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "g_" + hex.EncodeToString(b)
}

// CoordKey turns [x,y,z] into "x:y:z".
func CoordKey(coord []int) string {
	parts := make([]string, len(coord))
	for i, c := range coord {
		parts[i] = strconv.Itoa(c)
	}
	return strings.Join(parts, ":")
}

// ParseCoordKey turns "x:y:z" back into []int.
func ParseCoordKey(key string) ([]int, error) {
	if key == "" {
		return nil, nil
	}
	parts := strings.Split(key, ":")
	res := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		res[i] = n
	}
	return res, nil
}
