package column

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Appwrite fills an omitted integer or bigint bound with PHP_INT_MIN or
// PHP_INT_MAX, and an omitted float bound with -PHP_FLOAT_MAX or PHP_FLOAT_MAX.

func isUnboundedFloatMin(v float64) bool { return v <= -math.MaxFloat64 }

func isUnboundedFloatMax(v float64) bool { return v >= math.MaxFloat64 }

// Update keeps the stored bound when one is omitted, so a bound removed from
// the configuration would stay in place on the server. Update therefore sends
// the unbounded default for a bound the plan leaves null, the value Create
// leaves it at. The SDK takes an int, so this is the int64 limit on 64-bit
// builds.

func updateIntBounds(minV, maxV types.Int64) (int, int) {
	lo, hi := math.MinInt, math.MaxInt
	if !minV.IsNull() {
		lo = int(minV.ValueInt64())
	}
	if !maxV.IsNull() {
		hi = int(maxV.ValueInt64())
	}
	return lo, hi
}

func updateFloatBounds(minV, maxV types.Float64) (float64, float64) {
	lo, hi := -math.MaxFloat64, math.MaxFloat64
	if !minV.IsNull() {
		lo = minV.ValueFloat64()
	}
	if !maxV.IsNull() {
		hi = maxV.ValueFloat64()
	}
	return lo, hi
}

// exactInt64 reads an integer bound without going through float64, which
// cannot hold every int64 (math.MaxInt64 and math.MaxInt64-1 both round to
// 2^63). A value outside the int64 range is clamped instead of overflowing.
func exactInt64(n json.Number) (int64, bool) {
	if v, err := n.Int64(); err == nil {
		return v, true
	}
	f, err := n.Float64()
	if err != nil {
		return 0, false
	}
	switch {
	case f >= math.MaxInt64:
		return math.MaxInt64, true
	case f <= math.MinInt64:
		return math.MinInt64, true
	}
	return int64(f), true
}

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
