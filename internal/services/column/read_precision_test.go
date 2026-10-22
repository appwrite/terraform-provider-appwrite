package column

import (
	"context"
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

// Read used to decode the column response into a map and marshal it again,
// which rounds a bigint bound beyond 2^53 before it reaches state.
func TestReadKeepsBigintBoundsExact(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"big","type":"bigint","status":"available","required":false,"array":false,` +
			`"$createdAt":"2026-01-01T00:00:00.000+00:00","$updatedAt":"2026-01-01T00:00:00.000+00:00",` +
			`"min":-9007199254740993,"max":9007199254740993}`))
	}))
	defer srv.Close()

	r := &columnResource{clients: &common.AppwriteClients{
		BaseOptions: []client.ClientOption{appwrite.WithEndpoint(srv.URL + "/v1"), appwrite.WithKey("standard_test")},
		ProjectID:   "project",
	}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema

	state := tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &columnResourceModel{
		DatabaseID: types.StringValue("db"), TableID: types.StringValue("tbl"), Key: types.StringValue("big"),
		Type: types.StringValue(colTypeBigInt), Required: types.BoolValue(false), Array: types.BoolValue(false),
		Size: types.Int64Null(), Min: types.Int64Value(-9007199254740993), Max: types.Int64Value(9007199254740993),
		FloatMin: types.Float64Null(), FloatMax: types.Float64Null(), Elements: types.ListNull(types.StringType),
		DefaultStr: types.StringNull(), Encrypt: types.BoolNull(), RelatedTableID: types.StringNull(),
		RelationType: types.StringNull(), TwoWay: types.BoolNull(), TwoWayKey: types.StringNull(),
		OnDelete: types.StringNull(), CreatedAt: types.StringNull(), UpdatedAt: types.StringNull(),
		ProjectID: types.StringValue("project"),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	resp := resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	var got columnResourceModel
	resp.State.Get(ctx, &got)
	if got.Min.ValueInt64() != -9007199254740993 || got.Max.ValueInt64() != 9007199254740993 {
		t.Errorf("bounds after refresh = [%d, %d], want [-9007199254740993, 9007199254740993]",
			got.Min.ValueInt64(), got.Max.ValueInt64())
	}
}
