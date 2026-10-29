package row

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeRowServer stores the row body it is sent, byte for byte, and echoes it
// back the way Appwrite does, with system fields added.
type fakeRowServer struct {
	mu   sync.Mutex
	sent string
	row  map[string]json.RawMessage
}

func (f *fakeRowServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(req.Body)
	isCreate := req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/rows")
	isUpdate := req.Method == http.MethodPatch && strings.Contains(req.URL.Path, "/rows/")
	if isCreate || isUpdate {
		var in struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(body, &in)
		f.sent = string(in.Data)
		f.row = map[string]json.RawMessage{}
		_ = json.Unmarshal(in.Data, &f.row)
		for key, value := range map[string]string{
			"$id": `"row1"`, "$sequence": `"1"`, "$databaseId": `"db"`, "$tableId": `"tbl"`,
			"$createdAt": `"2026-01-01T00:00:00.000+00:00"`, "$updatedAt": `"2026-01-01T00:00:00.000+00:00"`,
			"$permissions": `[]`,
		} {
			f.row[key] = json.RawMessage(value)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	out, _ := json.Marshal(f.row)
	_, _ = w.Write(out)
}

// Integers beyond 2^53 cannot be represented by a float64. Row data used to be
// decoded into interface{} on the way out and on the way back, so such a value
// was rounded before Appwrite ever received it, and the rounded value was then
// written to state.
func TestRowDataKeepsIntegerPrecision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	server := &fakeRowServer{}
	srv := httptest.NewServer(server)
	defer srv.Close()

	r := &rowResource{clients: &common.AppwriteClients{
		BaseOptions: []client.ClientOption{appwrite.WithEndpoint(srv.URL + "/v1"), appwrite.WithKey("standard_test")},
		ProjectID:   "project",
	}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema
	null := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

	// The compact, key-sorted form jsonencode produces.
	const configured = `{"big":9007199254740993,"max":9223372036854775807,"ratio":0.5,"tags":[9007199254740995]}`

	plan := tfsdk.Plan{Schema: sch, Raw: null}
	if diags := plan.Set(ctx, &rowResourceModel{
		ID: types.StringValue("row1"), DatabaseID: types.StringValue("db"), TableID: types.StringValue("tbl"),
		Data: types.StringValue(configured), Permissions: types.ListUnknown(types.StringType),
		CreatedAt: types.StringUnknown(), UpdatedAt: types.StringUnknown(), ProjectID: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: null}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatal(createResp.Diagnostics)
	}
	if server.sent != configured {
		t.Errorf("row data sent to Appwrite:\n got %s\nwant %s", server.sent, configured)
	}
	var created rowResourceModel
	createResp.State.Get(ctx, &created)
	if got := created.Data.ValueString(); got != configured {
		t.Errorf("data in state after create:\n got %s\nwant %s", got, configured)
	}

	readResp := resource.ReadResponse{State: createResp.State}
	r.Read(ctx, resource.ReadRequest{State: createResp.State}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatal(readResp.Diagnostics)
	}
	var read rowResourceModel
	readResp.State.Get(ctx, &read)
	if got := read.Data.ValueString(); got != configured {
		t.Errorf("data in state after refresh:\n got %s\nwant %s", got, configured)
	}

	// Update sends the data through a separate SDK option, so check it too.
	const updated = `{"big":9007199254740997,"max":9223372036854775806,"ratio":0.5,"tags":[9007199254740999]}`
	updatePlan := tfsdk.Plan{Schema: sch, Raw: null}
	read.Data = types.StringValue(updated)
	if diags := updatePlan.Set(ctx, &read); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: readResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: readResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	if server.sent != updated {
		t.Errorf("row data sent to Appwrite on update:\n got %s\nwant %s", server.sent, updated)
	}
	var afterUpdate rowResourceModel
	updateResp.State.Get(ctx, &afterUpdate)
	if got := afterUpdate.Data.ValueString(); got != updated {
		t.Errorf("data in state after update:\n got %s\nwant %s", got, updated)
	}
}
