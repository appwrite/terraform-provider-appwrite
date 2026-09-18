package user

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/users"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResource              = &jwtEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &jwtEphemeralResource{}
)

type jwtEphemeralResource struct {
	clients *common.AppwriteClients
}

type jwtEphemeralModel struct {
	UserID          types.String `tfsdk:"user_id"`
	SessionID       types.String `tfsdk:"session_id"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
	JWT             types.String `tfsdk:"jwt"`
	ProjectID       types.String `tfsdk:"project_id"`
}

func NewJWTEphemeralResource() ephemeral.EphemeralResource {
	return &jwtEphemeralResource{}
}

func (r *jwtEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_jwt"
}

func (r *jwtEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mints a short-lived JWT for an Appwrite user. The token is never written to Terraform state.",
		Attributes: map[string]schema.Attribute{
			"user_id": schema.StringAttribute{
				Description: "The ID of the user to mint the token for.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"session_id": schema.StringAttribute{
				Description: "Scope the token to a specific session. Defaults to the user's most recent session.",
				Optional:    true,
			},
			"duration_seconds": schema.Int64Attribute{
				Description: "How long the token remains valid, in seconds. Defaults to the server's own default.",
				Optional:    true,
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"jwt": schema.StringAttribute{
				Description: "The encoded JWT.",
				Computed:    true,
				Sensitive:   true,
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

func (r *jwtEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *jwtEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config jwtEphemeralModel
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

	var opts []users.CreateJWTOption
	if !config.SessionID.IsNull() && !config.SessionID.IsUnknown() {
		opts = append(opts, usersClient.WithCreateJWTSessionId(config.SessionID.ValueString()))
	}
	if !config.DurationSeconds.IsNull() && !config.DurationSeconds.IsUnknown() {
		opts = append(opts, usersClient.WithCreateJWTDuration(int(config.DurationSeconds.ValueInt64())))
	}

	jwt, err := usersClient.CreateJWT(config.UserID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating JWT",
			common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_auth_jwt", "users.read", "sessions.write")),
		)
		return
	}

	config.JWT = types.StringValue(jwt.Jwt)
	config.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &config)...)

	// Neither Renew nor Close is implemented. A JWT cannot be revoked, and
	// Renew cannot hand Terraform a replacement token -- RenewResponse carries
	// no result data. Size duration_seconds to cover the run instead.
}
