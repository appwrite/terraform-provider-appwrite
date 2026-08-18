package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource              = &backupStorageResource{}
	_ resource.ResourceWithConfigure = &backupStorageResource{}
)

// backupStorageResource points a dedicated database's backups at a bucket the
// user owns, instead of Appwrite's default storage.
//
// The API exposes only a write route for this: there is no way to read the
// configured destination back. That shapes the whole resource. Read is a no-op
// because there is nothing to refresh against, drift is undetectable, and
// import is impossible, so the resource deliberately does not implement
// ImportState. Destroying it only forgets the configuration; backups keep going
// to the last destination that was applied.
type backupStorageResource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type backupStorageResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatabaseID      types.String `tfsdk:"database_id"`
	StorageProvider types.String `tfsdk:"storage_provider"`
	Bucket          types.String `tfsdk:"bucket"`
	AccessKey       types.String `tfsdk:"access_key"`
	SecretKey       types.String `tfsdk:"secret_key"`
	Region          types.String `tfsdk:"region"`
	Prefix          types.String `tfsdk:"prefix"`
	Endpoint        types.String `tfsdk:"endpoint"`
	ProjectID       types.String `tfsdk:"project_id"`
}

// NewBackupStorageResource returns a constructor for the backup storage
// resource of one engine.
func NewBackupStorageResource(engine Engine) func() resource.Resource {
	return func() resource.Resource {
		return &backupStorageResource{engine: engine}
	}
}

func (r *backupStorageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_backup_storage", req.ProviderTypeName, r.engine)
}

func (r *backupStorageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Sends the backups of a dedicated Appwrite %s database to a bucket you own rather than Appwrite's default storage.\n\n"+
				"The API offers no route to read this configuration back, so Terraform cannot detect drift, cannot verify what the "+
				"server currently has, and cannot import an existing configuration. Destroying this resource only removes it from "+
				"state; backups continue going to the last destination applied. Change the destination by applying a new one.",
			r.engine.Label(),
		),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The backup storage identifier, which is the database ID it belongs to.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database_id": schema.StringAttribute{
				Description:   "The dedicated database ID whose backups are redirected.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"storage_provider": schema.StringAttribute{
				// Named storage_provider rather than provider: `provider` is a
				// reserved root attribute name, since it is meta-argument syntax
				// on every resource block.
				Description: "The storage provider. One of `s3` (Amazon S3 or S3-compatible), `gcs` (Google Cloud Storage) or `azure` (Azure Blob Storage).",
				Required:    true,
				Validators:  []validator.String{stringvalidator.OneOf("s3", "gcs", "azure")},
			},
			"bucket": schema.StringAttribute{
				Description: "The bucket or container name to write backups into.",
				Required:    true,
			},
			"access_key": schema.StringAttribute{
				Description: "The access key used to authenticate against the bucket. Never returned by the API, so it is only ever what you configured.",
				Required:    true,
				Sensitive:   true,
			},
			"secret_key": schema.StringAttribute{
				Description: "The secret key used to authenticate against the bucket. Never returned by the API, so it is only ever what you configured.",
				Required:    true,
				Sensitive:   true,
			},
			"region": schema.StringAttribute{
				Description: "The storage region.",
				Optional:    true,
			},
			"prefix": schema.StringAttribute{
				Description: "The object key prefix to write backups under.",
				Optional:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "A custom endpoint, for S3-compatible storage that is not Amazon S3.",
				Optional:    true,
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *backupStorageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	r.clients = clients
}

func (r *backupStorageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupStorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics, &resp.State)
}

func (r *backupStorageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupStorageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics, &resp.State)
}

// apply pushes the destination. Create and Update are the same call.
func (r *backupStorageResource) apply(ctx context.Context, plan backupStorageResourceModel, diagnostics *diag.Diagnostics, state *tfsdk.State) {
	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	api := clientFor(r.clients, r.engine, projectID)

	storage, err := api.UpdateBackupStorage(
		plan.DatabaseID.ValueString(),
		plan.StorageProvider.ValueString(),
		plan.Bucket.ValueString(),
		plan.AccessKey.ValueString(),
		plan.SecretKey.ValueString(),
		BackupStorageOptions{
			Region:   optString(plan.Region),
			Prefix:   optString(plan.Prefix),
			Endpoint: optString(plan.Endpoint),
		},
	)
	if err != nil {
		diagnostics.AddError("Error configuring backup storage", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.ID = types.StringValue(plan.DatabaseID.ValueString())

	// The response echoes the non-secret fields. Only overwrite the optional
	// ones the user left unset, so an omitted attribute stays null rather than
	// flipping to a server default that the config does not mention. The keys
	// are never echoed, so they keep the configured values.
	plan.StorageProvider = types.StringValue(storage.Provider)
	plan.Bucket = types.StringValue(storage.Bucket)
	plan.Region = preserveNull(plan.Region, storage.Region)
	plan.Prefix = preserveNull(plan.Prefix, storage.Prefix)
	plan.Endpoint = preserveNull(plan.Endpoint, storage.Endpoint)

	diagnostics.Append(state.Set(ctx, &plan)...)
}

// Read cannot refresh anything: the API has no route that returns the
// configured backup destination. Leaving prior state untouched is the only
// honest option, and the schema documents that drift is undetectable.
func (r *backupStorageResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

// Delete forgets the configuration. There is no route to clear a destination,
// so the server keeps writing backups where it was last told to.
func (r *backupStorageResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Backup storage configuration not removed from the server",
		"Appwrite has no route to clear a custom backup destination, so removing this resource only drops it from Terraform state. "+
			"The database keeps sending backups to the bucket that was last applied. Point it somewhere else by applying a new "+
			"configuration, or change it in the Appwrite Console.",
	)
}

// preserveNull keeps an unconfigured optional attribute null instead of
// adopting whatever the server echoed back for it.
func preserveNull(configured types.String, serverValue string) types.String {
	if configured.IsNull() || configured.IsUnknown() {
		return types.StringNull()
	}
	return types.StringValue(serverValue)
}
