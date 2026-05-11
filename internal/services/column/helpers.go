package column

import (
	"fmt"
	"strconv"
)

func parseIntDefault(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseBigIntDefault(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid bigint default %q: %w", s, err)
	}
	return int(v), nil
}

func parseFloatDefault(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
