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
	backupsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/backup"
	bucketsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/bucket"
	columnsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/column"
	databasesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/database"
	filesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/file"
	functionsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/function"
	indexsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/index"
	sitesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/site"
	providersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/provider"
	rowsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/row"
	subscribersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/subscriber"
	tablesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/table"
	teamsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/team"
	topicsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/topic"
	usersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/user"
	webhooksvc "github.com/appwrite/terraform-provider-appwrite/internal/services/webhook"
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
				Description: "The default Appwrite project ID for all resources. Can also be set with the APPWRITE_PROJECT_ID environment variable. Can be overridden per-resource.",
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
	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing Appwrite API Key",
			"The provider requires an api_key to be set either in the provider configuration or via the APPWRITE_API_KEY environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	baseOpts := []client.ClientOption{
		appwrite.WithEndpoint(endpoint),
		appwrite.WithKey(apiKey),
	}
	if !config.SelfSigned.IsNull() && config.SelfSigned.ValueBool() {
		baseOpts = append(baseOpts, appwrite.WithSelfSigned(true))
	}

	clients := &common.AppwriteClients{
		BaseOptions: baseOpts,
		ProjectID:   projectID,
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *appwriteProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		databasesvc.NewDatabaseResource,
		tablesvc.NewTableResource,
		columnsvc.NewColumnResource,
		indexsvc.NewIndexResource,
		bucketsvc.NewBucketResource,
		topicsvc.NewTopicResource,
		providersvc.NewProviderResource,
		usersvc.NewUserResource,
		teamsvc.NewTeamResource,
		backupsvc.NewPolicyResource,
		rowsvc.NewRowResource,
		webhooksvc.NewWebhookResource,
		subscribersvc.NewSubscriberResource,
		filesvc.NewFileResource,
		functionsvc.NewFunctionResource,
		functionsvc.NewVariableResource,
		sitesvc.NewSiteResource,
		sitesvc.NewVariableResource,
	}
}

func (p *appwriteProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		databasesvc.NewDatabaseDataSource,
	}
}

func stringValueOrEnv(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}
