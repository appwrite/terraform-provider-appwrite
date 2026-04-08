package column

import "strconv"

func parseIntDefault(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseFloatDefault(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
