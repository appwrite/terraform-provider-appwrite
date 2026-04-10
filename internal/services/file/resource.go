package file

import (
	"context"
	"fmt"
	"strings"

	appwritefile "github.com/appwrite/sdk-for-go/v2/file"
	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/models"
	"github.com/appwrite/sdk-for-go/v2/storage"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &fileResource{}
	_ resource.ResourceWithConfigure   = &fileResource{}
	_ resource.ResourceWithImportState = &fileResource{}
)

type fileResource struct {
	storage   *storage.Storage
	projectID string
}

type fileResourceModel struct {
	ID           types.String `tfsdk:"id"`
	BucketID     types.String `tfsdk:"bucket_id"`
	Name         types.String `tfsdk:"name"`
	FilePath     types.String `tfsdk:"file_path"`
	Permissions  types.List   `tfsdk:"permissions"`
	MimeType     types.String `tfsdk:"mime_type"`
	SizeOriginal types.Int64  `tfsdk:"size_original"`
	ProjectID    types.String `tfsdk:"project_id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewFileResource() resource.Resource {
	return &fileResource{}
}

func (r *fileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_file"
}

func (r *fileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file in an Appwrite storage bucket.",
		Attributes: map[string]schema.Attribute{
			"project_id": common.ProjectIDAttribute(),
			"id": schema.StringAttribute{
				Description:   "The file ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"bucket_id": schema.StringAttribute{
				Description:   "The bucket ID.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Description: "The file name.",
				Optional:    true,
				Computed:    true,
			},
			"file_path": schema.StringAttribute{
				Description:   "The local path to the file to upload.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"permissions": schema.ListAttribute{
				Description: "File permissions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"mime_type": schema.StringAttribute{
				Description: "The file MIME type.",
				Computed:    true,
			},
			"size_original": schema.Int64Attribute{
				Description: "The original file size in bytes.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The file creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The file last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *fileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.storage = clients.Storage
	r.projectID = clients.ProjectID
}

func (r *fileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fileID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		fileID = id.Unique()
	}

	fileName := plan.Name.ValueString()
	if plan.Name.IsNull() || plan.Name.IsUnknown() {
		fileName = ""
	}
	inputFile := appwritefile.NewInputFile(plan.FilePath.ValueString(), fileName)

	var opts []storage.CreateFileOption
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithCreateFilePermissions(perms))
	}

	f, err := r.storage.CreateFile(plan.BucketID.ValueString(), fileID, inputFile, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating file", common.FormatError(err))
		return
	}

	r.mapToState(ctx, f, &plan, &resp.Diagnostics)
	if plan.ProjectID.IsNull() || plan.ProjectID.IsUnknown() {
		plan.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *fileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	f, err := r.storage.GetFile(state.BucketID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading file", common.FormatError(err))
		return
	}

	r.mapToState(ctx, f, &state, &resp.Diagnostics)
	if state.ProjectID.IsNull() || state.ProjectID.IsUnknown() {
		state.ProjectID = types.StringValue(r.projectID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *fileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var opts []storage.UpdateFileOption
	if !plan.Name.IsNull() {
		opts = append(opts, r.storage.WithUpdateFileName(plan.Name.ValueString()))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, r.storage.WithUpdateFilePermissions(perms))
	}

	f, err := r.storage.UpdateFile(plan.BucketID.ValueString(), plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating file", common.FormatError(err))
		return
	}

	r.mapToState(ctx, f, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *fileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.storage.DeleteFile(state.BucketID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting file", common.FormatError(err))
	}
}

func (r *fileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: bucket_id/file_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: bucket_id/file_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("bucket_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *fileResource) mapToState(ctx context.Context, f *models.File, model *fileResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(f.Id)
	model.BucketID = types.StringValue(f.BucketId)
	model.Name = types.StringValue(f.Name)
	model.MimeType = types.StringValue(f.MimeType)
	model.SizeOriginal = types.Int64Value(int64(f.SizeOriginal))
	model.CreatedAt = types.StringValue(f.CreatedAt)
	model.UpdatedAt = types.StringValue(f.UpdatedAt)

	permsList, diags := types.ListValueFrom(ctx, types.StringType, f.Permissions)
	diagnostics.Append(diags...)
	model.Permissions = permsList
}
