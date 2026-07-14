package function

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
	_ datasource.DataSource              = &functionDataSource{}
	_ datasource.DataSourceWithConfigure = &functionDataSource{}
)

type functionDataSource struct {
	clients *common.AppwriteClients
}

type functionDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Runtime      types.String `tfsdk:"runtime"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Logging      types.Bool   `tfsdk:"logging"`
	Entrypoint   types.String `tfsdk:"entrypoint"`
	Commands     types.String `tfsdk:"commands"`
	Schedule     types.String `tfsdk:"schedule"`
	Timeout      types.Int64  `tfsdk:"timeout"`
	Execute      types.List   `tfsdk:"execute"`
	Events       types.List   `tfsdk:"events"`
	Scopes       types.List   `tfsdk:"scopes"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	ProjectID    types.String `tfsdk:"project_id"`
}

func NewFunctionDataSource() datasource.DataSource {
	return &functionDataSource{}
}

func (d *functionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function"
}

func (d *functionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite function by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The function ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The function name.",
				Computed:    true,
			},
			"runtime": schema.StringAttribute{
				Description: "The function execution runtime.",
				Computed:    true,
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the function is enabled.",
				Computed:    true,
			},
			"logging": schema.BoolAttribute{
				Description: "Whether execution logs are enabled.",
				Computed:    true,
			},
			"entrypoint": schema.StringAttribute{
				Description: "The entrypoint file.",
				Computed:    true,
			},
			"commands": schema.StringAttribute{
				Description: "The build command.",
				Computed:    true,
			},
			"schedule": schema.StringAttribute{
				Description: "Function execution schedule in CRON format.",
				Computed:    true,
			},
			"timeout": schema.Int64Attribute{
				Description: "Function execution timeout in seconds.",
				Computed:    true,
			},
			"execute": schema.ListAttribute{
				Description: "Execution permissions.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"events": schema.ListAttribute{
				Description: "Events that trigger the function.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"scopes": schema.ListAttribute{
				Description: "Function scopes.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"deployment_id": schema.StringAttribute{
				Description: "The active deployment ID.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The function creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The function last update timestamp.",
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

func (d *functionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *functionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config functionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	functionsClient := appwrite.NewFunctions(d.clients.ClientForProject(projectID))

	fn, err := functionsClient.Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading function", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(fn.Id)
	config.Name = types.StringValue(fn.Name)
	config.Runtime = types.StringValue(fn.Runtime)
	config.Enabled = types.BoolValue(fn.Enabled)
	config.Logging = types.BoolValue(fn.Logging)
	config.Timeout = types.Int64Value(int64(fn.Timeout))
	config.DeploymentID = types.StringValue(fn.DeploymentId)
	config.CreatedAt = types.StringValue(fn.CreatedAt)
	config.UpdatedAt = types.StringValue(fn.UpdatedAt)

	if fn.Entrypoint != "" {
		config.Entrypoint = types.StringValue(fn.Entrypoint)
	}
	if fn.Commands != "" {
		config.Commands = types.StringValue(fn.Commands)
	}
	if fn.Schedule != "" {
		config.Schedule = types.StringValue(fn.Schedule)
	}

	execList, diags := types.ListValueFrom(ctx, types.StringType, fn.Execute)
	resp.Diagnostics.Append(diags...)
	config.Execute = execList

	eventsList, diags := types.ListValueFrom(ctx, types.StringType, fn.Events)
	resp.Diagnostics.Append(diags...)
	config.Events = eventsList

	scopesList, diags := types.ListValueFrom(ctx, types.StringType, fn.Scopes)
	resp.Diagnostics.Append(diags...)
	config.Scopes = scopesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
