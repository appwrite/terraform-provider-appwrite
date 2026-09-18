package project

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ ephemeral.EphemeralResource              = &ephemeralKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &ephemeralKeyEphemeralResource{}
)

// maxEphemeralKeyDuration is the server's ceiling. Asking for more is rejected
// outright rather than clamped, so reject it here with a better message.
const maxEphemeralKeyDuration = 3600

type ephemeralKeyEphemeralResource struct {
	clients *common.AppwriteClients
}

type ephemeralKeyModel struct {
	Scopes          types.Set    `tfsdk:"scopes"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
	ID              types.String `tfsdk:"id"`
	Secret          types.String `tfsdk:"secret"`
	Expire          types.String `tfsdk:"expire"`
	ProjectID       types.String `tfsdk:"project_id"`
}

func NewEphemeralKeyEphemeralResource() ephemeral.EphemeralResource {
	return &ephemeralKeyEphemeralResource{}
}

func (r *ephemeralKeyEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_ephemeral_key"
}

func (r *ephemeralKeyEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mints a short-lived, scoped Appwrite project API key that is never written to Terraform state. " +
			"The key cannot be revoked -- it is a signed token the server does not store -- so it stays valid until " +
			"`duration_seconds` elapses. Ask for the shortest duration and the narrowest scopes the run needs.",
		Attributes: map[string]schema.Attribute{
			"scopes": schema.SetAttribute{
				Description: "The permission scopes granted to the key. Grant only what the run needs.",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1), setvalidator.SizeAtMost(100)},
			},
			"duration_seconds": schema.Int64Attribute{
				Description: fmt.Sprintf("How long the key remains valid, in seconds. Defaults to %d, which is also the maximum the server allows. The key cannot be revoked early, so prefer the shortest duration that covers the run.", maxEphemeralKeyDuration),
				Optional:    true,
				Validators:  []validator.Int64{int64validator.Between(1, maxEphemeralKeyDuration)},
			},
			"id": schema.StringAttribute{
				Description: "The key ID.",
				Computed:    true,
			},
			"secret": schema.StringAttribute{
				Description: "The key secret. Pass this to a downstream provider or write-only argument.",
				Computed:    true,
				Sensitive:   true,
			},
			"expire": schema.StringAttribute{
				Description: "The key expiration timestamp in ISO 8601 format.",
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

func (r *ephemeralKeyEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
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

func (r *ephemeralKeyEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var config ephemeralKeyModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	var scopes []string
	resp.Diagnostics.Append(config.Scopes.ElementsAs(ctx, &scopes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	duration := int64(maxEphemeralKeyDuration)
	if !config.DurationSeconds.IsNull() && !config.DurationSeconds.IsUnknown() {
		duration = config.DurationSeconds.ValueInt64()
	}

	projectClient := appwrite.NewProject(r.clients.ClientForProject(projectID))
	key, err := projectClient.CreateEphemeralKey(scopes, int(duration))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating ephemeral API key",
			common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_project_ephemeral_key", "keys.write")),
		)
		return
	}

	config.ID = types.StringValue(key.Id)
	config.Secret = types.StringValue(key.Secret)
	config.Expire = types.StringValue(key.Expire)
	config.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.Result.Set(ctx, &config)...)

	// Neither Renew nor Close is implemented, and neither can be.
	//
	// An ephemeral key is a signed token carrying its own scopes and expiry;
	// the server keeps no record of it. GET and DELETE on /project/keys/{id}
	// both return 404 for one, and the secret keeps working regardless, so a
	// Close that called DeleteKey would swallow a 404 and imply a revocation
	// that never happened. Renew is no use either: RenewResponse carries no
	// result data, so it cannot hand Terraform a replacement secret.
	//
	// What bounds the exposure is duration_seconds, capped by the server at an
	// hour, together with the scopes the caller asks for.
}
