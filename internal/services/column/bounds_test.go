package column

import (
	"context"
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Appwrite fills a bound the request left out with the widest value the type
// allows and returns it. Adopting that into a min or max the configuration
// leaves null is an inconsistent result after apply, which taints the column.
func TestReadResponseIntoStateKeepsUnsetBoundsNull(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		colType  string
		response string
	}{
		{
			name:     "integer",
			colType:  colTypeInteger,
			response: `{"key":"age","type":"integer","status":"available","required":false,"array":false,"min":-9223372036854775808,"max":9223372036854775807,"default":null}`,
		},
		{
			name:     "bigint",
			colType:  colTypeBigInt,
			response: `{"key":"views","type":"bigint","status":"available","required":false,"array":false,"min":-9223372036854775808,"max":9223372036854775807,"default":null}`,
		},
		{
			name:     "float",
			colType:  colTypeFloat,
			response: `{"key":"score","type":"double","status":"available","required":false,"array":false,"min":-1.7976931348623157e+308,"max":1.7976931348623157e+308,"default":null}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, importing := range []bool{false, true} {
				model := columnResourceModel{
					Type:     types.StringValue(tc.colType),
					Min:      types.Int64Null(),
					Max:      types.Int64Null(),
					FloatMin: types.Float64Null(),
					FloatMax: types.Float64Null(),
				}
				if importing {
					model.Type = types.StringNull()
				}
				var r columnResource
				var d diag.Diagnostics
				r.readResponseIntoState(context.Background(), []byte(tc.response), &model, &d)
				if d.HasError() {
					t.Fatalf("diagnostics: %v", d)
				}
				for name, v := range map[string]interface{ IsNull() bool }{
					"min": model.Min, "max": model.Max, "float_min": model.FloatMin, "float_max": model.FloatMax,
				} {
					if !v.IsNull() {
						t.Errorf("importing=%t: %s = %v, want null for a bound the configuration does not set", importing, name, v)
					}
				}
			}
		})
	}
}

func TestReadResponseIntoStateKeepsConfiguredBounds(t *testing.T) {
	t.Parallel()

	model := columnResourceModel{
		Type:     types.StringValue(colTypeInteger),
		Min:      types.Int64Value(math.MinInt64),
		Max:      types.Int64Value(100),
		FloatMin: types.Float64Null(),
		FloatMax: types.Float64Null(),
	}
	var r columnResource
	var d diag.Diagnostics
	r.readResponseIntoState(context.Background(),
		[]byte(`{"key":"age","type":"integer","min":-9223372036854775808,"max":100}`), &model, &d)
	if d.HasError() {
		t.Fatalf("diagnostics: %v", d)
	}
	if got := model.Min.ValueInt64(); model.Min.IsNull() || got != math.MinInt64 {
		t.Errorf("min = %v, want %d: a configured bound must be kept even at the type's limit", model.Min, int64(math.MinInt64))
	}
	if got := model.Max.ValueInt64(); model.Max.IsNull() || got != 100 {
		t.Errorf("max = %v, want 100", model.Max)
	}

	// math.MaxInt64 comes back as the float64 2^63, which does not fit in an
	// int64. A configured maximum at the type's limit must still be kept.
	maxModel := columnResourceModel{
		Type:     types.StringValue(colTypeBigInt),
		Min:      types.Int64Null(),
		Max:      types.Int64Value(math.MaxInt64),
		FloatMin: types.Float64Null(),
		FloatMax: types.Float64Null(),
	}
	r.readResponseIntoState(context.Background(),
		[]byte(`{"key":"big","type":"bigint","min":-9223372036854775808,"max":9223372036854775807}`), &maxModel, &d)
	if got := maxModel.Max.ValueInt64(); maxModel.Max.IsNull() || got != math.MaxInt64 {
		t.Errorf("max = %v, want %d: a configured bound must be kept even at the type's limit", maxModel.Max, int64(math.MaxInt64))
	}
	if !maxModel.Min.IsNull() {
		t.Errorf("min = %v, want null", maxModel.Min)
	}

	floatModel := columnResourceModel{
		Type:     types.StringValue(colTypeFloat),
		Min:      types.Int64Null(),
		Max:      types.Int64Null(),
		FloatMin: types.Float64Value(0.5),
		FloatMax: types.Float64Null(),
	}
	r.readResponseIntoState(context.Background(),
		[]byte(`{"key":"score","type":"double","min":0.5,"max":60}`), &floatModel, &d)
	if floatModel.FloatMin.ValueFloat64() != 0.5 {
		t.Errorf("float_min = %v, want 0.5", floatModel.FloatMin)
	}
	// A bound the server reports that is not its unbounded default is real
	// drift and must still surface, even when the configuration leaves it out.
	if floatModel.FloatMax.IsNull() || floatModel.FloatMax.ValueFloat64() != 60 {
		t.Errorf("float_max = %v, want 60", floatModel.FloatMax)
	}
}
