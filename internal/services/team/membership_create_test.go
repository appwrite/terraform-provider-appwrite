package team

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// TestCreateKeepsInvitationInStateWhenRolesFail drives Create against a server
// that confirms a pending invitation and then refuses the roles update. The
// membership exists by then, so Create must record it in state alongside the
// error; Terraform then taints it rather than losing it.
func TestCreateKeepsInvitationInStateWhenRolesFail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method + " " + r.URL.Path {
		case "POST /v1/teams/team-id/memberships":
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, membershipJSON)
		case "PATCH /v1/teams/team-id/memberships/membership-id":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"message":"Invalid roles","code":400,"type":"general_argument_invalid"}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotImplemented)
		}
	}))
	defer srv.Close()

	r := &membershipResource{clients: &common.AppwriteClients{
		BaseOptions: []client.ClientOption{appwrite.WithEndpoint(srv.URL + "/v1"), appwrite.WithKey("standard_test")},
		ProjectID:   "project",
	}}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	sch := schemaResp.Schema
	null := tftypes.NewValue(sch.Type().TerraformType(ctx), nil)

	lead, _ := types.SetValueFrom(ctx, types.StringType, []string{"lead"})
	plan := tfsdk.Plan{Schema: sch, Raw: null}
	if diags := plan.Set(ctx, &membershipResourceModel{
		ID: types.StringUnknown(), TeamID: types.StringValue("team-id"), UserID: types.StringValue("user-id"),
		Email: types.StringNull(), Phone: types.StringNull(), Name: types.StringNull(), Roles: lead,
		Confirm: types.BoolUnknown(), JoinedAt: types.StringUnknown(), UserName: types.StringUnknown(),
		UserEmail: types.StringUnknown(), UserPhone: types.StringUnknown(), CreatedAt: types.StringUnknown(),
		UpdatedAt: types.StringUnknown(), ProjectID: types.StringUnknown(),
	}); diags.HasError() {
		t.Fatal(diags)
	}

	createResp := resource.CreateResponse{State: tfsdk.State{Schema: sch, Raw: null}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, &createResp)

	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected the roles update error")
	}
	if got := createResp.Diagnostics.Errors()[0].Summary(); got != "Error setting team membership roles" {
		t.Errorf("error summary = %q", got)
	}
	if createResp.State.Raw.IsNull() {
		t.Fatal("state is empty; the confirmed membership would be left unmanaged")
	}
	var state membershipResourceModel
	if diags := createResp.State.Get(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	if state.ID.ValueString() != "membership-id" || state.TeamID.ValueString() != "team-id" {
		t.Errorf("state id = %s, team_id = %s; want the confirmed membership", state.ID, state.TeamID)
	}
	var roles []string
	state.Roles.ElementsAs(ctx, &roles, false)
	if !sameRoles(roles, []string{"dev"}) {
		t.Errorf("state roles = %v, want the invitation's [dev]", roles)
	}
	if want := "POST /v1/teams/team-id/memberships,PATCH /v1/teams/team-id/memberships/membership-id"; strings.Join(requests, ",") != want {
		t.Errorf("requests = %v", requests)
	}
}
