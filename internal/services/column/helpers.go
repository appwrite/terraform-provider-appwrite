package column

import (
	"fmt"
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Appwrite fills an omitted integer or bigint bound with PHP_INT_MIN or
// PHP_INT_MAX, and an omitted float bound with -PHP_FLOAT_MAX or PHP_FLOAT_MAX.
// Bounds arrive decoded as float64, where PHP_INT_MAX rounds up to 2^63, so the
// integer checks compare with <= and >= rather than for equality.

func isUnboundedIntMin(v float64) bool { return v <= math.MinInt64 }

func isUnboundedIntMax(v float64) bool { return v >= math.MaxInt64 }

func isUnboundedFloatMin(v float64) bool { return v <= -math.MaxFloat64 }

func isUnboundedFloatMax(v float64) bool { return v >= math.MaxFloat64 }

// int64Bound converts a bound read back as float64 into an int64 bound. A
// float64 cannot hold every int64 (math.MaxInt64 arrives as 2^63), so when the
// server reports the configured value, the configured digits are kept, and
// anything beyond the int64 range is clamped instead of overflowing.
func int64Bound(configured types.Int64, v float64) types.Int64 {
	if !configured.IsNull() && !configured.IsUnknown() && float64(configured.ValueInt64()) == v {
		return configured
	}
	switch {
	case v >= math.MaxInt64:
		return types.Int64Value(math.MaxInt64)
	case v <= math.MinInt64:
		return types.Int64Value(math.MinInt64)
	}
	return types.Int64Value(int64(v))
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
