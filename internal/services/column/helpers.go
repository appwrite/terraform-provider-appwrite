package column

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Appwrite fills an omitted integer or bigint bound with PHP_INT_MIN or
// PHP_INT_MAX, and an omitted float bound with -PHP_FLOAT_MAX or PHP_FLOAT_MAX.

func isUnboundedFloatMin(v float64) bool { return v <= -math.MaxFloat64 }

func isUnboundedFloatMax(v float64) bool { return v >= math.MaxFloat64 }

// Update keeps the stored bound when one is omitted, so a bound removed from
// the configuration would stay in place on the server. Update therefore sends
// the unbounded default for a bound the plan leaves null, the value Create
// leaves it at.

func updateIntBounds(minV, maxV types.Int64) (int64, int64) {
	lo, hi := int64(math.MinInt64), int64(math.MaxInt64)
	if !minV.IsNull() {
		lo = minV.ValueInt64()
	}
	if !maxV.IsNull() {
		hi = maxV.ValueInt64()
	}
	return lo, hi
}

// updateIntegerColumn updates an integer or bigint column with a raw call and
// returns the response body. The SDK takes the bounds as int and parses them
// back into an *int, and neither holds PHP_INT_MIN/PHP_INT_MAX on the 32-bit
// builds this provider ships: resetting a removed bound would instead narrow
// the column to the int32 range.
func updateIntegerColumn(c client.Client, columnType, databaseID, tableID, key string, required bool, def, minVal, maxVal int64) ([]byte, error) {
	path := fmt.Sprintf("/tablesdb/%s/tables/%s/columns/%s/%s",
		client.EncodePath(databaseID), client.EncodePath(tableID), columnType, client.EncodePath(key))
	resp, err := c.Call("PATCH", path, map[string]interface{}{
		// client.Call does not add the project header; see common.GetColumnRawJSON.
		"X-Appwrite-Project": c.Config["project"],
		"content-type":       "application/json",
		"accept":             "application/json",
	}, map[string]interface{}{
		"required": required,
		"min":      minVal,
		"max":      maxVal,
		"default":  def,
	})
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resp.Type, "application/json") {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	return client.ResponseBody(resp)
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
