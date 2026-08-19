package provider

import (
	"context"
	"os"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
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
	dedicatedsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/dedicated"
	docdbsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/docdb"
	filesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/file"
	functionsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/function"
	indexsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/index"
	projectsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/project"
	providersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/provider"
	proxysvc "github.com/appwrite/terraform-provider-appwrite/internal/services/proxy"
	rowsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/row"
	sitesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/site"
	subscribersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/subscriber"
	tablesvc "github.com/appwrite/terraform-provider-appwrite/internal/services/table"
	teamsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/team"
	topicsvc "github.com/appwrite/terraform-provider-appwrite/internal/services/topic"
	usersvc "github.com/appwrite/terraform-provider-appwrite/internal/services/user"
	webhooksvc "github.com/appwrite/terraform-provider-appwrite/internal/services/webhook"
)

// defaultHTTPTimeout is the per-request ceiling. It has to clear the slowest
// operation the server performs inline rather than asynchronously.
const defaultHTTPTimeout = 120 * time.Second

var _ provider.Provider = &appwriteProvider{}

type appwriteProvider struct {
	version string
}

type appwriteProviderModel struct {
	Endpoint           types.String `tfsdk:"endpoint"`
	ProjectID          types.String `tfsdk:"project_id"`
	OrganizationID     types.String `tfsdk:"organization_id"`
	APIKey             types.String `tfsdk:"api_key"`
	OrganizationAPIKey types.String `tfsdk:"organization_api_key"`
	SelfSigned         types.Bool   `tfsdk:"self_signed"`
	HTTPTimeoutSeconds types.Int64  `tfsdk:"http_timeout_seconds"`
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
			"organization_id": schema.StringAttribute{
				Description: "The default Appwrite organization ID for organization-scoped resources. Can also be set with the APPWRITE_ORGANIZATION_ID environment variable. Can be overridden per-resource.",
				Optional:    true,
			},
			"api_key": schema.StringAttribute{
				Description: "The project API key used for project-scoped resources. Can also be set with the APPWRITE_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"organization_api_key": schema.StringAttribute{
				Description: "The organization API key used for organization-scoped resources and project API key management. Defaults to api_key for backwards compatibility. Can also be set with the APPWRITE_ORGANIZATION_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"self_signed": schema.BoolAttribute{
				Description: "Accept self-signed SSL certificates. Useful for Appwrite Community Edition with self-signed certs. Defaults to false.",
				Optional:    true,
			},
			"http_timeout_seconds": schema.Int64Attribute{
				Description: "How long to wait for a single API response before giving up. Defaults to 120. The SDK's own default is 10 seconds, which is too short for operations the server completes inline, such as updating a connection pooler.",
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
	organizationID := stringValueOrEnv(config.OrganizationID, "APPWRITE_ORGANIZATION_ID")
	apiKey := stringValueOrEnv(config.APIKey, "APPWRITE_API_KEY")
	organizationAPIKey := stringValueOrEnv(config.OrganizationAPIKey, "APPWRITE_ORGANIZATION_API_KEY")
	if organizationAPIKey == "" {
		organizationAPIKey = apiKey
	}

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

	// The SDK defaults to a 10 second timeout, which some Appwrite operations
	// exceed while the server works inline -- updating a connection pooler
	// restarts the sidecar and routinely takes longer. A request that times out
	// after the server already applied the change leaves Terraform reporting a
	// failure for work that succeeded, so the ceiling is raised well clear of it.
	httpTimeout := defaultHTTPTimeout
	if !config.HTTPTimeoutSeconds.IsNull() && !config.HTTPTimeoutSeconds.IsUnknown() {
		httpTimeout = time.Duration(config.HTTPTimeoutSeconds.ValueInt64()) * time.Second
	}

	baseOpts := []client.ClientOption{
		appwrite.WithEndpoint(endpoint),
		appwrite.WithKey(apiKey),
		appwrite.WithTimeout(httpTimeout),
		common.WithUserAgent(p.version),
	}
	organizationBaseOpts := []client.ClientOption{
		appwrite.WithEndpoint(endpoint),
		appwrite.WithKey(organizationAPIKey),
		appwrite.WithTimeout(httpTimeout),
		common.WithUserAgent(p.version),
	}
	if !config.SelfSigned.IsNull() && config.SelfSigned.ValueBool() {
		baseOpts = append(baseOpts, appwrite.WithSelfSigned(true))
		organizationBaseOpts = append(organizationBaseOpts, appwrite.WithSelfSigned(true))
	}

	clients := &common.AppwriteClients{
		BaseOptions:                baseOpts,
		OrganizationBaseOptions:    organizationBaseOpts,
		ProjectCredentialType:      common.DetectCredentialType(apiKey),
		OrganizationCredentialType: common.DetectCredentialType(organizationAPIKey),
		ProjectID:                  projectID,
		OrganizationID:             organizationID,
	}

	resp.DataSourceData = clients
	resp.ResourceData = clients
}

func (p *appwriteProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		projectsvc.NewProjectResource,
		projectsvc.NewKeyResource,
		proxysvc.NewRuleResource,
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
		sitesvc.NewDeploymentResource,
		functionsvc.NewDeploymentResource,

		// Dedicated databases. Each engine is exposed as its own resource type
		// because Appwrite routes them separately and only some engines have a
		// pooler or extensions.
		dedicatedsvc.NewDatabaseResource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewDatabaseResource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewDatabaseResource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewBackupPolicyResource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewBackupPolicyResource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewBackupPolicyResource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewBackupStorageResource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewBackupStorageResource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewBackupStorageResource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewBranchResource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewBranchResource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewBranchResource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewPoolerResource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewPoolerResource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewExtensionResource(dedicatedsvc.EnginePostgresql),

		// DocumentsDB and VectorsDB. Two products, one implementation: only the
		// embedding dimension differs, and it is exposed on VectorsDB alone.
		docdbsvc.NewDatabaseResource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewDatabaseResource(docdbsvc.ProductVectorsDB),
		docdbsvc.NewCollectionResource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewCollectionResource(docdbsvc.ProductVectorsDB),
		docdbsvc.NewIndexResource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewIndexResource(docdbsvc.ProductVectorsDB),
		docdbsvc.NewDocumentResource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewDocumentResource(docdbsvc.ProductVectorsDB),
	}
}

func (p *appwriteProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		databasesvc.NewDatabaseDataSource,
		bucketsvc.NewBucketDataSource,
		usersvc.NewUserDataSource,
		teamsvc.NewTeamDataSource,
		functionsvc.NewFunctionDataSource,
		sitesvc.NewSiteDataSource,
		topicsvc.NewTopicDataSource,
		webhooksvc.NewWebhookDataSource,

		dedicatedsvc.NewDatabaseDataSource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewDatabaseDataSource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewDatabaseDataSource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewDatabasesDataSource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewDatabasesDataSource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewDatabasesDataSource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewSpecificationsDataSource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewSpecificationsDataSource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewSpecificationsDataSource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewStatusDataSource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewStatusDataSource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewStatusDataSource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewBackupsDataSource(dedicatedsvc.EnginePostgresql),
		dedicatedsvc.NewBackupsDataSource(dedicatedsvc.EngineMysql),
		dedicatedsvc.NewBackupsDataSource(dedicatedsvc.EngineMongo),
		dedicatedsvc.NewExtensionsDataSource(dedicatedsvc.EnginePostgresql),

		docdbsvc.NewDatabaseDataSource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewDatabaseDataSource(docdbsvc.ProductVectorsDB),
		docdbsvc.NewSpecificationsDataSource(docdbsvc.ProductDocumentsDB),
		docdbsvc.NewSpecificationsDataSource(docdbsvc.ProductVectorsDB),
	}
}

func stringValueOrEnv(val types.String, envVar string) string {
	if !val.IsNull() && !val.IsUnknown() {
		return val.ValueString()
	}
	return os.Getenv(envVar)
}
