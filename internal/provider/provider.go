package provider

import (
	"context"
	"os"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	"github.com/appwrite/sdk-for-go/v2/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/bucket"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/column"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/database"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/index"
	messagingprovider "github.com/appwrite/terraform-provider-appwrite/internal/services/provider"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/table"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/team"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/topic"
	"github.com/appwrite/terraform-provider-appwrite/internal/services/user"
)

var _ provider.Provider = &appwriteProvider{}

type appwriteProvider struct {
	version string
}

type appwriteProviderModel struct {
	Endpoint   types.String `tfsdk:"endpoint"`
	ProjectID  types.String `tfsdk:"project_id"`
	APIKey     types.String `tfsdk:"api_key"`
	SelfSigned types.Bool   `tfsdk:"self_signed"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &appwriteProvider{version: version}
	}
}

func (p *appwriteProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "appwrite"
	resp.Version = p.version
}

func (p *appwriteProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with Appwrite Cloud or Community Edition.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The Appwrite API endpoint (e.g. https://cloud.appwrite.io/v1). Can also be set with the APPWRITE_ENDPOINT environment variable.",
				Optional:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Can also be set with the APPWRITE_PROJECT_ID environment variable.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The Appwrite API key. Can also be set with the APPWRITE_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"self_signed": schema.BoolAttribute{
				Description: "Accept self-signed SSL certificates. Useful for Appwrite Community Edition with self-signed certs. Defaults to false.",
				Optional:    true,
			},
		},
	}
}

func (p *appwriteProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config appwriteProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := stringValueOrEnv(config.Endpoint, "APPWRITE_ENDPOINT")
	projectID := stringValueOrEnv(config.ProjectID, "APPWRITE_PROJECT_ID")
	apiKey := stringValueOrEnv(config.APIKey, "APPWRITE_API_KEY")

	if endpoint == "" {
		resp.Diagnostics.AddError(
			"Missing Appwrite Endpoint",
			"The provider requires an endpoint to be set either in the provider configuration or via the APPWRITE_ENDPOINT environment variable.",
		)
	}
	if projectID == "" {
		resp.Diagnostics.AddError(
			"Missing Appwrite Project ID",
			"The provider requires a project_id to be set either in the provider configuration or via the APPWRITE_PROJECT_ID environment variable.",
		)
	}
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing Appwrite API Key",
			"The provider requires an api_key to be set either in the provider configuration or via the APPWRITE_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	opts := []client.ClientOption{
		appwrite.WithEndpoint(endpoint),
		appwrite.WithProject(projectID),
		appwrite.WithKey(apiKey),
	}
	if !config.SelfSigned.IsNull() && config.SelfSigned.ValueBool() {
		opts = append(opts, appwrite.WithSelfSigned(true))
	}

	c := appwrite.NewClient(opts...)

	clients := &common.AppwriteClients{
		TablesDB:  appwrite.NewTablesDB(c),
		Storage:   appwrite.NewStorage(c),
		Messaging: appwrite.NewMessaging(c),
		Users:     appwrite.NewUsers(c),
		Teams:     appwrite.NewTeams(c),
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *appwriteProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		database.NewDatabaseResource,
		table.NewTableResource,
		column.NewColumnResource,
		index.NewIndexResource,
		bucket.NewBucketResource,
		topic.NewTopicResource,
		messagingprovider.NewProviderResource,
		user.NewUserResource,
		team.NewTeamResource,
	}
}

func (p *appwriteProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		database.NewDatabaseDataSource,
	}
}

func stringValueOrEnv(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}
