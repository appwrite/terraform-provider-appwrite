package column

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Appwrite keeps the stored bound when an update omits it. Removing a bound
// from the configuration used to send nothing, so the server kept the old
// bound and returned it into an argument the plan left null: an inconsistent
// result after apply.
func TestUpdateResetsRemovedBound(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		colType string
		apiType string
		path    string
		prior   func(*columnResourceModel)
		wantMin string
		wantMax string
	}{
		{
			name:    "integer",
			colType: colTypeInteger,
			apiType: "integer",
			path:    "/v1/tablesdb/db/tables/tbl/columns/integer/n",
			prior:   func(m *columnResourceModel) { m.Min = types.Int64Value(0); m.Max = types.Int64Value(100) },
			wantMin: "-9223372036854775808",
			wantMax: "9223372036854775807",
		},
		{
			name:    "bigint",
			colType: colTypeBigInt,
			apiType: "bigint",
			path:    "/v1/tablesdb/db/tables/tbl/columns/bigint/n",
			prior:   func(m *columnResourceModel) { m.Max = types.Int64Value(100) },
			wantMin: "-9223372036854775808",
			wantMax: "9223372036854775807",
		},
		{
			name:    "float",
			colType: colTypeFloat,
			apiType: "double",
			path:    "/v1/tablesdb/db/tables/tbl/columns/float/n",
			prior: func(m *columnResourceModel) {
				m.FloatMin = types.Float64Value(0.5)
				m.FloatMax = types.Float64Value(60)
			},
			wantMin: "-1.7976931348623157e+308",
			wantMax: "1.7976931348623157e+308",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			type bounds struct {
				Method string          `json:"-"`
				Path   string          `json:"-"`
				Min    json.RawMessage `json:"min"`
				Max    json.RawMessage `json:"max"`
			}
			requests := make(chan bounds, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				body, _ := io.ReadAll(req.Body)
				var sent bounds
				_ = json.Unmarshal(body, &sent)
				sent.Method, sent.Path = req.Method, req.URL.Path
				requests <- sent
				// Like Appwrite, return the bounds the update stored.
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"key":"n","type":"` + tc.apiType + `","status":"available","required":false,"array":false,` +
					`"$createdAt":"2026-01-01T00:00:00.000+00:00","$updatedAt":"2026-01-01T00:00:00.000+00:00",` +
					`"min":` + string(sent.Min) + `,"max":` + string(sent.Max) + `}`))
			}))
			defer srv.Close()

			r := &columnResource{clients: &common.AppwriteClients{
				BaseOptions: []client.ClientOption{appwrite.WithEndpoint(srv.URL + "/v1"), appwrite.WithKey("standard_test")},
				ProjectID:   "project",
			}}
			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			sch := schemaResp.Schema
			null := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

			unbounded := columnResourceModel{
				DatabaseID: types.StringValue("db"), TableID: types.StringValue("tbl"), Key: types.StringValue("n"),
				Type: types.StringValue(tc.colType), Required: types.BoolValue(false), Array: types.BoolValue(false),
				Size: types.Int64Null(), Min: types.Int64Null(), Max: types.Int64Null(),
				FloatMin: types.Float64Null(), FloatMax: types.Float64Null(), Elements: types.ListNull(types.StringType),
				DefaultStr: types.StringNull(), Encrypt: types.BoolNull(), RelatedTableID: types.StringNull(),
				RelationType: types.StringNull(), TwoWay: types.BoolNull(), TwoWayKey: types.StringNull(),
				OnDelete: types.StringNull(), CreatedAt: types.StringValue("2026-01-01T00:00:00.000+00:00"),
				UpdatedAt: types.StringUnknown(), ProjectID: types.StringValue("project"),
			}
			bounded := unbounded
			bounded.UpdatedAt = types.StringValue("2026-01-01T00:00:00.000+00:00")
			tc.prior(&bounded)

			state := tfsdk.State{Schema: sch, Raw: null}
			if diags := state.Set(ctx, &bounded); diags.HasError() {
				t.Fatal(diags)
			}
			plan := tfsdk.Plan{Schema: sch, Raw: null}
			if diags := plan.Set(ctx, &unbounded); diags.HasError() {
				t.Fatal(diags)
			}

			resp := resource.UpdateResponse{State: state}
			r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatal(resp.Diagnostics)
			}
			var sent bounds
			select {
			case sent = <-requests:
			default:
				t.Fatal("no update request reached the server")
			}
			if sent.Method != http.MethodPatch || sent.Path != tc.path {
				t.Errorf("request = %s %s, want PATCH %s", sent.Method, sent.Path, tc.path)
			}
			if string(sent.Min) != tc.wantMin || string(sent.Max) != tc.wantMax {
				t.Errorf("bounds sent = [%s, %s], want [%s, %s]", sent.Min, sent.Max, tc.wantMin, tc.wantMax)
			}
			var got columnResourceModel
			resp.State.Get(ctx, &got)
			for name, v := range map[string]interface{ IsNull() bool }{
				"min": got.Min, "max": got.Max, "float_min": got.FloatMin, "float_max": got.FloatMax,
			} {
				if !v.IsNull() {
					t.Errorf("%s = %v after update, want null for a bound the configuration removed", name, v)
				}
			}
		})
	}
}
