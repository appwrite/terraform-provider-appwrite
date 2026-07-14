package site

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v6/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &siteDataSource{}
	_ datasource.DataSourceWithConfigure = &siteDataSource{}
)

type siteDataSource struct {
	clients *common.AppwriteClients
}

type siteDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Framework       types.String `tfsdk:"framework"`
	BuildRuntime    types.String `tfsdk:"build_runtime"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	Logging         types.Bool   `tfsdk:"logging"`
	Timeout         types.Int64  `tfsdk:"timeout"`
	InstallCommand  types.String `tfsdk:"install_command"`
	BuildCommand    types.String `tfsdk:"build_command"`
	OutputDirectory types.String `tfsdk:"output_directory"`
	DeploymentID    types.String `tfsdk:"deployment_id"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
	ProjectID       types.String `tfsdk:"project_id"`
}

func NewSiteDataSource() datasource.DataSource {
	return &siteDataSource{}
}

func (d *siteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (d *siteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite site by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The site ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The site name.",
				Computed:    true,
			},
			"framework": schema.StringAttribute{
				Description: "The site framework.",
				Computed:    true,
			},
			"build_runtime": schema.StringAttribute{
				Description: "The build runtime.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the site is enabled.",
				Computed:    true,
			},
			"logging": schema.BoolAttribute{
				Description: "Whether logging is enabled.",
				Computed:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "Site timeout in seconds.",
				Computed:    true,
			},
			"install_command": schema.StringAttribute{
				Description: "Custom install command.",
				Computed:    true,
			},
			"build_command": schema.StringAttribute{
				Description: "Custom build command.",
				Computed:    true,
			},
			"output_directory": schema.StringAttribute{
				Description: "Build output directory.",
				Computed:    true,
			},
			"deployment_id": schema.StringAttribute{
				Description: "The active deployment ID.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The site creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The site last update timestamp.",
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

func (d *siteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *siteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config siteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(d.clients.ClientForProject(projectID))

	site, err := sitesClient.Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading site", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(site.Id)
	config.Name = types.StringValue(site.Name)
	config.Framework = types.StringValue(site.Framework)
	config.BuildRuntime = types.StringValue(site.BuildRuntime)
	config.Enabled = types.BoolValue(site.Enabled)
	config.Logging = types.BoolValue(site.Logging)
	config.Timeout = types.Int64Value(int64(site.Timeout))
	config.DeploymentID = types.StringValue(site.DeploymentId)
	config.CreatedAt = types.StringValue(site.CreatedAt)
	config.UpdatedAt = types.StringValue(site.UpdatedAt)

	if site.InstallCommand != "" {
		config.InstallCommand = types.StringValue(site.InstallCommand)
	}
	if site.BuildCommand != "" {
		config.BuildCommand = types.StringValue(site.BuildCommand)
	}
	if site.OutputDirectory != "" {
		config.OutputDirectory = types.StringValue(site.OutputDirectory)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
