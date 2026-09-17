package project

import (
	"context"
	"encoding/json"
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
	_ ephemeral.EphemeralResourceWithClose     = &ephemeralKeyEphemeralResource{}
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

// ephemeralKeyPrivate carries what Close needs to revoke the key. Close is
// handed nothing but this, so the project has to travel with the key ID.
type ephemeralKeyPrivate struct {
	KeyID     string `json:"key_id"`
	ProjectID string `json:"project_id"`
}

func NewEphemeralKeyEphemeralResource() ephemeral.EphemeralResource {
	return &ephemeralKeyEphemeralResource{}
}

func (r *ephemeralKeyEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_ephemeral_key"
}

func (r *ephemeralKeyEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Mints a short-lived Appwrite project API key that is never written to Terraform state. " +
			"The key is revoked when Terraform finishes with it.",
		Attributes: map[string]schema.Attribute{
			"scopes": schema.SetAttribute{
				Description: "The permission scopes granted to the key. Grant only what the run needs.",
				Required:    true,
				ElementType: types.StringType,
				Validators:  []validator.Set{setvalidator.SizeAtLeast(1), setvalidator.SizeAtMost(100)},
			},
			"duration_seconds": schema.Int64Attribute{
				Description: fmt.Sprintf("How long the key remains valid, in seconds. Defaults to %d, which is also the maximum the server allows.", maxEphemeralKeyDuration),
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
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
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
	if resp.Diagnostics.HasError() {
		return
	}

	private, err := json.Marshal(ephemeralKeyPrivate{KeyID: key.Id, ProjectID: projectID})
	if err != nil {
		resp.Diagnostics.AddError("Error recording ephemeral key state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, ephemeralKeyPrivateKey, private)...)

	// No RenewAt is set. Appwrite has no endpoint to extend a key's life, and
	// Renew cannot hand Terraform a replacement secret -- RenewResponse carries
	// no result data. Choose duration_seconds to cover the whole run instead.
}

func (r *ephemeralKeyEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	raw, diags := req.Private.GetKey(ctx, ephemeralKeyPrivateKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || len(raw) == 0 {
		return
	}

	var private ephemeralKeyPrivate
	if err := json.Unmarshal(raw, &private); err != nil {
		resp.Diagnostics.AddError("Error reading ephemeral key state", err.Error())
		return
	}

	projectClient := appwrite.NewProject(r.clients.ClientForProject(private.ProjectID))
	// Revoking early is the point: the key would lapse on its own, but leaving
	// a usable credential alive after the run defeats the exercise.
	if _, err := projectClient.DeleteKey(private.KeyID); err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddWarning(
			"Could not revoke ephemeral API key",
			fmt.Sprintf("The key %q will expire on its own, but could not be revoked now: %s", private.KeyID, common.FormatError(err)),
		)
	}
}

// ephemeralKeyPrivateKey names the private-state slot shared by Open and Close.
const ephemeralKeyPrivateKey = "ephemeral_key"
