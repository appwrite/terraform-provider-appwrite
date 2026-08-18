package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &extensionsDataSource{}
	_ datasource.DataSourceWithConfigure = &extensionsDataSource{}
)

type extensionsDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type extensionsDataSourceModel struct {
	DatabaseID types.String    `tfsdk:"database_id"`
	ProjectID  types.String    `tfsdk:"project_id"`
	Installed  types.Set       `tfsdk:"installed"`
	Available  types.Set       `tfsdk:"available"`
	Metadata   []extensionMeta `tfsdk:"metadata"`
}

type extensionMeta struct {
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
}

// NewExtensionsDataSource returns a constructor for the extensions data source.
func NewExtensionsDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &extensionsDataSource{engine: engine}
	}
}

func (d *extensionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_extensions", req.ProviderTypeName, d.engine)
}

func (d *extensionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Lists the extensions installed on, and available to, a dedicated Appwrite %s database.", d.engine.Label()),
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{
				Description: "The dedicated database ID.",
				Required:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
			"installed": schema.SetAttribute{
				Description: "The extensions currently installed.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"available": schema.SetAttribute{
				Description: "The extensions that can be installed.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"metadata": schema.ListNestedAttribute{
				Description: "Curated metadata for each available extension.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":         schema.StringAttribute{Description: "The extension key, as passed to `name` on the extension resource.", Computed: true},
						"name":        schema.StringAttribute{Description: "The human readable extension name.", Computed: true},
						"description": schema.StringAttribute{Description: "What the extension provides.", Computed: true},
						"category":    schema.StringAttribute{Description: "The category the extension belongs to.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *extensionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *extensionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config extensionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	api, ok := newExtensionAPI(d.engine, d.clients.ClientForProject(projectID))
	if !ok {
		resp.Diagnostics.AddError("Unsupported engine", fmt.Sprintf("%s databases do not support extensions.", d.engine.Label()))
		return
	}

	extensions, err := api.ListExtensions(config.DatabaseID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing extensions", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)

	installed, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(extensions.Installed))
	resp.Diagnostics.Append(diags...)
	config.Installed = installed

	available, diags := types.SetValueFrom(ctx, types.StringType, nonNilStrings(extensions.Available))
	resp.Diagnostics.Append(diags...)
	config.Available = available

	config.Metadata = make([]extensionMeta, 0, len(extensions.Metadata))
	for _, meta := range extensions.Metadata {
		config.Metadata = append(config.Metadata, extensionMeta{
			Key:         types.StringValue(meta.Key),
			Name:        types.StringValue(meta.Name),
			Description: types.StringValue(meta.Description),
			Category:    types.StringValue(meta.Category),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
