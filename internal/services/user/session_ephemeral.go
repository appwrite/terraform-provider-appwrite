package user

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResource              = &sessionEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &sessionEphemeralResource{}
	_ ephemeral.EphemeralResourceWithClose     = &sessionEphemeralResource{}
)

// sessionPrivateKey names the private-state slot shared by Open and Close.
const sessionPrivateKey = "session"

type sessionEphemeralResource struct {
	clients *common.AppwriteClients
}

type sessionEphemeralModel struct {
	UserID    types.String `tfsdk:"user_id"`
	ID        types.String `tfsdk:"id"`
	Secret    types.String `tfsdk:"secret"`
	Expire    types.String `tfsdk:"expire"`
	ProjectID types.String `tfsdk:"project_id"`
}

// sessionPrivate carries what Close needs to delete the session. Close is
// handed nothing but this, so the user and project travel with the session ID.
type sessionPrivate struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

func NewSessionEphemeralResource() ephemeral.EphemeralResource {
	return &sessionEphemeralResource{}
}

func (r *sessionEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_session"
}

func (r *sessionEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Opens a session for an Appwrite user, and deletes it once Terraform is finished. " +
			"The session secret is never written to Terraform state.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The ID of the user to open a session for.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"id": schema.StringAttribute{
				Description: "The session ID.",
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The session secret.",
				Computed:    true,
				Sensitive:   true,
			},
			"expire": schema.StringAttribute{
				Description: "The session expiration timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: common.ProjectIDDescription,
				Optional:    true,
				// Computed as well, because Open resolves the provider-level
				// default into it; an Optional-only attribute may not come back
				// from Open with a different value than was configured.
				Computed: true,
			},
		},
	}
}

func (r *sessionEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Ephemeral Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.clients = clients
}

func (r *sessionEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config sessionEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	usersClient := appwrite.NewUsers(r.clients.ClientForProject(projectID))
	session, err := usersClient.CreateSession(config.UserID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating session",
			common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_auth_session", "sessions.write")),
		)
		return
	}

	config.ID = types.StringValue(session.Id)
	config.Secret = types.StringValue(session.Secret)
	config.Expire = types.StringValue(session.Expire)
	config.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	private, err := json.Marshal(sessionPrivate{
		SessionID: session.Id,
		UserID:    config.UserID.ValueString(),
		ProjectID: projectID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error recording session state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, sessionPrivateKey, private)...)

	// No RenewAt: Appwrite has no endpoint to extend a session, and Renew
	// cannot hand Terraform a replacement secret.
}

func (r *sessionEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	raw, diags := req.Private.GetKey(ctx, sessionPrivateKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || len(raw) == 0 {
		return
	}

	var private sessionPrivate
	if err := json.Unmarshal(raw, &private); err != nil {
		resp.Diagnostics.AddError("Error reading session state", err.Error())
		return
	}

	usersClient := appwrite.NewUsers(r.clients.ClientForProject(private.ProjectID))
	// A session outlives the run by default, so closing it is the whole point:
	// the secret handed to the configuration stops working when Terraform is
	// done with it.
	if _, err := usersClient.DeleteSession(private.UserID, private.SessionID); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddWarning(
			"Could not delete session",
			fmt.Sprintf("The session %q will expire on its own, but could not be deleted now: %s", private.SessionID, common.FormatError(err)),
		)
	}
}
