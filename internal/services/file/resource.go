package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/appwrite/sdk-for-go/v4/appwrite"
	appwritefile "github.com/appwrite/sdk-for-go/v4/file"
	"github.com/appwrite/sdk-for-go/v4/id"
	"github.com/appwrite/sdk-for-go/v4/models"
	"github.com/appwrite/sdk-for-go/v4/storage"
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
	clients *common.AppwriteClients
}

type fileResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ProjectID    types.String `tfsdk:"project_id"`
	BucketID     types.String `tfsdk:"bucket_id"`
	Name         types.String `tfsdk:"name"`
	FilePath     types.String `tfsdk:"file_path"`
	Permissions  types.List   `tfsdk:"permissions"`
	MimeType     types.String `tfsdk:"mime_type"`
	SizeOriginal types.Int64  `tfsdk:"size_original"`
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
			"id": schema.StringAttribute{
				Description:   "The file ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
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
	r.clients = clients
}

func (r *fileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	storageClient := appwrite.NewStorage(r.clients.ClientForProject(projectID))

	fileID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		fileID = id.Unique()
	}

	fileName := plan.Name.ValueString()
	if plan.Name.IsNull() || plan.Name.IsUnknown() {
		fileName = ""
	}
	filePath := plan.FilePath.ValueString()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		resp.Diagnostics.AddError(
			"File not found",
			fmt.Sprintf("The file at path %q does not exist. Please verify the file_path attribute points to a valid local file.", filePath),
		)
		return
	}
	inputFile := appwritefile.NewInputFile(filePath, fileName)

	var opts []storage.CreateFileOption
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, storageClient.WithCreateFilePermissions(perms))
	}

	f, err := storageClient.CreateFile(plan.BucketID.ValueString(), fileID, inputFile, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating file", common.FormatError(err))
		return
	}

	mapFileToModel(ctx, f, &plan, &resp.Diagnostics)
	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *fileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	storageClient := appwrite.NewStorage(r.clients.ClientForProject(projectID))

	f, err := storageClient.GetFile(state.BucketID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading file", common.FormatError(err))
		return
	}

	mapFileToModel(ctx, f, &state, &resp.Diagnostics)
	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *fileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan fileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	storageClient := appwrite.NewStorage(r.clients.ClientForProject(projectID))

	var opts []storage.UpdateFileOption
	if !plan.Name.IsNull() {
		opts = append(opts, storageClient.WithUpdateFileName(plan.Name.ValueString()))
	}
	if !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown() {
		var perms []string
		resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &perms, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, storageClient.WithUpdateFilePermissions(perms))
	}

	f, err := storageClient.UpdateFile(plan.BucketID.ValueString(), plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating file", common.FormatError(err))
		return
	}

	mapFileToModel(ctx, f, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *fileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state fileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	storageClient := appwrite.NewStorage(r.clients.ClientForProject(projectID))

	_, err = storageClient.DeleteFile(state.BucketID.ValueString(), state.ID.ValueString())
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

func mapFileToModel(ctx context.Context, f *models.File, model *fileResourceModel, diagnostics *diag.Diagnostics) {
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
