package bucket

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v3/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &bucketDataSource{}
	_ datasource.DataSourceWithConfigure = &bucketDataSource{}
)

type bucketDataSource struct {
	clients *common.AppwriteClients
}

type bucketDataSourceModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Enabled               types.Bool   `tfsdk:"enabled"`
	MaximumFileSize       types.Int64  `tfsdk:"maximum_file_size"`
	AllowedFileExtensions types.List   `tfsdk:"allowed_file_extensions"`
	FileSecurity          types.Bool   `tfsdk:"file_security"`
	Compression           types.String `tfsdk:"compression"`
	Encryption            types.Bool   `tfsdk:"encryption"`
	Antivirus             types.Bool   `tfsdk:"antivirus"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
	ProjectID             types.String `tfsdk:"project_id"`
}

func NewBucketDataSource() datasource.DataSource {
	return &bucketDataSource{}
}

func (d *bucketDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage_bucket"
}

func (d *bucketDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite storage bucket by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The bucket ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The bucket name.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the bucket is enabled.",
				Computed:    true,
			},
			"maximum_file_size": schema.Int64Attribute{
				Description: "Maximum file size in bytes.",
				Computed:    true,
			},
			"allowed_file_extensions": schema.ListAttribute{
				Description: "List of allowed file extensions.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"file_security": schema.BoolAttribute{
				Description: "Whether file-level security is enabled.",
				Computed:    true,
			},
			"compression": schema.StringAttribute{
				Description: "Compression algorithm.",
				Computed:    true,
			},
			"encryption": schema.BoolAttribute{
				Description: "Whether bucket encryption is enabled.",
				Computed:    true,
			},
			"antivirus": schema.BoolAttribute{
				Description: "Whether virus scanning is enabled.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The bucket creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The bucket last update timestamp.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (d *bucketDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	d.clients = clients
}

func (d *bucketDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config bucketDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	storageClient := appwrite.NewStorage(d.clients.ClientForProject(projectID))

	bucket, err := storageClient.GetBucket(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading bucket", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(bucket.Id)
	config.Name = types.StringValue(bucket.Name)
	config.Enabled = types.BoolValue(bucket.Enabled)
	config.MaximumFileSize = types.Int64Value(int64(bucket.MaximumFileSize))
	config.FileSecurity = types.BoolValue(bucket.FileSecurity)
	config.Compression = types.StringValue(bucket.Compression)
	config.Encryption = types.BoolValue(bucket.Encryption)
	config.Antivirus = types.BoolValue(bucket.Antivirus)
	config.CreatedAt = types.StringValue(bucket.CreatedAt)
	config.UpdatedAt = types.StringValue(bucket.UpdatedAt)

	extList, diags := types.ListValueFrom(ctx, types.StringType, bucket.AllowedFileExtensions)
	resp.Diagnostics.Append(diags...)
	config.AllowedFileExtensions = extList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
