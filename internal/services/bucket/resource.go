package bucket

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/storage"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &bucketResource{}
	_ resource.ResourceWithConfigure   = &bucketResource{}
	_ resource.ResourceWithImportState = &bucketResource{}
)

type bucketResource struct {
	storage *storage.Storage
}

type bucketResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	FileSecurity          types.Bool   `tfsdk:"file_security"`
	MaximumFileSize       types.Int64  `tfsdk:"maximum_file_size"`
	AllowedFileExtensions types.List   `tfsdk:"allowed_file_extensions"`
	Permissions           types.List   `tfsdk:"permissions"`
	Compression           types.String `tfsdk:"compression"`
	Encryption            types.Bool   `tfsdk:"encryption"`
	Antivirus             types.Bool   `tfsdk:"antivirus"`
	Transformations       types.Bool   `tfsdk:"transformations"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func NewBucketResource() resource.Resource {
	return &bucketResource{}
}

func (r *bucketResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bucket"
}

func (r *bucketResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite storage bucket.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The bucket ID. Must be unique within the project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The bucket name.",
				Required:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the bucket is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"file_security": schema.BoolAttribute{
				Description: "Whether file-level security is enabled. When enabled, users can access files for which they have been granted permissions. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"maximum_file_size": schema.Int64Attribute{
				Description: "Maximum file size in bytes.",
				Optional:    true,
				Computed:    true,
			},
			"allowed_file_extensions": schema.ListAttribute{
				Description: "List of allowed file extensions. An empty list allows all extensions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"permissions": schema.ListAttribute{
				Description: "Bucket-level permissions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm: none, gzip, or zstd. Defaults to none.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
			},
			"encryption": schema.BoolAttribute{
				Description: "Whether bucket encryption is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"antivirus": schema.BoolAttribute{
				Description: "Whether virus scanning is enabled. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"transformations": schema.BoolAttribute{
				Description: "Whether image transformations are enabled. Defaults to false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				Description: "The bucket creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The bucket last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *bucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.storage = clients.Storage
}

func (r *bucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []storage.CreateBucketOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.FileSecurity.IsNull() && !plan.FileSecurity.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketFileSecurity(plan.FileSecurity.ValueBool()))
	}
	if !plan.MaximumFileSize.IsNull() && !plan.MaximumFileSize.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketMaximumFileSize(int(plan.MaximumFileSize.ValueInt64())))
	}
	if !plan.AllowedFileExtensions.IsNull() && !plan.AllowedFileExtensions.IsUnknown() {
		var extensions []string
		resp.Diagnostics.Append(plan.AllowedFileExtensions.ElementsAs(ctx, &extensions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithCreateBucketAllowedFileExtensions(extensions))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithCreateBucketPermissions(perms))
	}
	if !plan.Compression.IsNull() && !plan.Compression.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketCompression(plan.Compression.ValueString()))
	}
	if !plan.Encryption.IsNull() && !plan.Encryption.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketEncryption(plan.Encryption.ValueBool()))
	}
	if !plan.Antivirus.IsNull() && !plan.Antivirus.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketAntivirus(plan.Antivirus.ValueBool()))
	}
	if !plan.Transformations.IsNull() && !plan.Transformations.IsUnknown() {
		opts = append(opts, r.storage.WithCreateBucketTransformations(plan.Transformations.ValueBool()))
	}

	bucket, err := r.storage.CreateBucket(plan.ID.ValueString(), plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating bucket", common.FormatError(err))
		return
	}

	mapBucketToModel(ctx, bucket, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state bucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bucket, err := r.storage.GetBucket(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading bucket", common.FormatError(err))
		return
	}

	mapBucketToModel(ctx, bucket, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *bucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan bucketResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []storage.UpdateBucketOption
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketEnabled(plan.Enabled.ValueBool()))
	}
	if !plan.FileSecurity.IsNull() && !plan.FileSecurity.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketFileSecurity(plan.FileSecurity.ValueBool()))
	}
	if !plan.MaximumFileSize.IsNull() && !plan.MaximumFileSize.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketMaximumFileSize(int(plan.MaximumFileSize.ValueInt64())))
	}
	if !plan.AllowedFileExtensions.IsNull() && !plan.AllowedFileExtensions.IsUnknown() {
		var extensions []string
		resp.Diagnostics.Append(plan.AllowedFileExtensions.ElementsAs(ctx, &extensions, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithUpdateBucketAllowedFileExtensions(extensions))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithUpdateBucketPermissions(perms))
	}
	if !plan.Compression.IsNull() && !plan.Compression.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketCompression(plan.Compression.ValueString()))
	}
	if !plan.Encryption.IsNull() && !plan.Encryption.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketEncryption(plan.Encryption.ValueBool()))
	}
	if !plan.Antivirus.IsNull() && !plan.Antivirus.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketAntivirus(plan.Antivirus.ValueBool()))
	}
	if !plan.Transformations.IsNull() && !plan.Transformations.IsUnknown() {
		opts = append(opts, r.storage.WithUpdateBucketTransformations(plan.Transformations.ValueBool()))
	}

	bucket, err := r.storage.UpdateBucket(plan.ID.ValueString(), plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating bucket", common.FormatError(err))
		return
	}

	mapBucketToModel(ctx, bucket, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *bucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bucketResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.storage.DeleteBucket(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting bucket", common.FormatError(err))
	}
}

func (r *bucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapBucketToModel(ctx context.Context, bucket *models.Bucket, model *bucketResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(bucket.Id)
	model.Name = types.StringValue(bucket.Name)
	model.Enabled = types.BoolValue(bucket.Enabled)
	model.FileSecurity = types.BoolValue(bucket.FileSecurity)
	model.MaximumFileSize = types.Int64Value(int64(bucket.MaximumFileSize))
	model.Compression = types.StringValue(bucket.Compression)
	model.Encryption = types.BoolValue(bucket.Encryption)
	model.Antivirus = types.BoolValue(bucket.Antivirus)
	model.Transformations = types.BoolValue(bucket.Transformations)
	model.CreatedAt = types.StringValue(bucket.CreatedAt)
	model.UpdatedAt = types.StringValue(bucket.UpdatedAt)

	permsList, diags := types.ListValueFrom(ctx, types.StringType, bucket.Permissions)
	diagnostics.Append(diags...)
	model.Permissions = permsList

	extList, diags := types.ListValueFrom(ctx, types.StringType, bucket.AllowedFileExtensions)
	diagnostics.Append(diags...)
	model.AllowedFileExtensions = extList
}
