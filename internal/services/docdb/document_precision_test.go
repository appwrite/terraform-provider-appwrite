package docdb

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// The document body was decoded into interface{} before being sent, so an
// integer beyond 2^53 reached Appwrite already rounded.
func TestDocumentDataSentWithIntegerPrecision(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const configured = `{"big":9007199254740993}`
	sent := make(chan string, 1)
	updated := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := io.ReadAll(req.Body)
		var in struct {
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(body, &in)
		if req.Method == http.MethodPost && strings.HasSuffix(req.URL.Path, "/documents") {
			sent <- string(in.Data)
		}
		if req.Method == http.MethodPatch && strings.Contains(req.URL.Path, "/documents/") {
			updated <- string(in.Data)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"$id":"doc1","$sequence":"1","$collectionId":"col","$databaseId":"db",` +
			`"$createdAt":"2026-01-01T00:00:00.000+00:00","$updatedAt":"2026-01-01T00:00:00.000+00:00",` +
			`"$permissions":[],"big":9007199254740993}`))
	}))
	defer srv.Close()

	r := &documentResource{product: ProductDocumentsDB, clients: &common.AppwriteClients{
		BaseOptions: []client.ClientOption{appwrite.WithEndpoint(srv.URL + "/v1"), appwrite.WithKey("standard_test")},
		ProjectID:   "project",
	}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema
	null := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

	plan := tfsdk.Plan{Schema: sch, Raw: null}
	if diags := plan.Set(ctx, &documentResourceModel{
		ID: types.StringValue("doc1"), DatabaseID: types.StringValue("db"), CollectionID: types.StringValue("col"),
		Data: types.StringValue(configured), Permissions: types.SetUnknown(types.StringType),
		CreatedAt: types.StringUnknown(), UpdatedAt: types.StringUnknown(), ProjectID: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	resp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: null}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	select {
	case got := <-sent:
		if got != configured {
			t.Errorf("document data sent to Appwrite:\n got %s\nwant %s", got, configured)
		}
	default:
		t.Fatal("no create request reached the server")
	}

	// Update sends the data through a separate SDK option, so check it too.
	const changed = `{"big":9007199254740997}`
	var current documentResourceModel
	resp.State.Get(ctx, &current)
	current.Data = types.StringValue(changed)
	updatePlan := tfsdk.Plan{Schema: sch, Raw: null}
	if diags := updatePlan.Set(ctx, &current); diags.HasError() {
		t.Fatal(diags)
	}
	updateResp := resource.UpdateResponse{State: resp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: updatePlan, State: resp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatal(updateResp.Diagnostics)
	}
	select {
	case got := <-updated:
		if got != changed {
			t.Errorf("document data sent to Appwrite on update:\n got %s\nwant %s", got, changed)
		}
	default:
		t.Fatal("no update request reached the server")
	}
}
