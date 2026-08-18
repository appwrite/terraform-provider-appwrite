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
	_ datasource.DataSource              = &backupsDataSource{}
	_ datasource.DataSourceWithConfigure = &backupsDataSource{}
)

// backupsDataSource lists the backups taken of a dedicated database, which is
// how a backup ID is found for a restore performed outside Terraform.
type backupsDataSource struct {
	engine  Engine
	clients *common.AppwriteClients
}

type backupsDataSourceModel struct {
	DatabaseID types.String `tfsdk:"database_id"`
	ProjectID  types.String `tfsdk:"project_id"`
	Queries    types.List   `tfsdk:"queries"`
	Total      types.Int64  `tfsdk:"total"`
	Backups    []backupItem `tfsdk:"backups"`
}

type backupItem struct {
	ID             types.String `tfsdk:"id"`
	PolicyID       types.String `tfsdk:"policy_id"`
	Trigger        types.String `tfsdk:"trigger"`
	Type           types.String `tfsdk:"type"`
	RequestedType  types.String `tfsdk:"requested_type"`
	FallbackReason types.String `tfsdk:"fallback_reason"`
	Status         types.String `tfsdk:"status"`
	SizeBytes      types.Int64  `tfsdk:"size_bytes"`
	StartedAt      types.String `tfsdk:"started_at"`
	CompletedAt    types.String `tfsdk:"completed_at"`
	VerifiedAt     types.String `tfsdk:"verified_at"`
	ExpiresAt      types.String `tfsdk:"expires_at"`
	LogPosition    types.String `tfsdk:"log_position"`
	Error          types.String `tfsdk:"error"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

// NewBackupsDataSource returns a constructor for the backups data source of one
// engine.
func NewBackupsDataSource(engine Engine) func() datasource.DataSource {
	return func() datasource.DataSource {
		return &backupsDataSource{engine: engine}
	}
}

func (d *backupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_backups", req.ProviderTypeName, d.engine)
}

func (d *backupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Lists the backups taken of a dedicated Appwrite %s database. Restoring is not a Terraform operation, so this is how "+
				"a backup ID is found for a restore run through the Console or API.",
			d.engine.Label(),
		),
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
			"queries": schema.ListAttribute{
				Description: "Appwrite query strings used to filter the listing, for example `equal(\"status\", \"completed\")`.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"total": schema.Int64Attribute{
				Description: "The total number of backups matching the query.",
				Computed:    true,
			},
			"backups": schema.ListNestedAttribute{
				Description: "The backups that matched.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":              schema.StringAttribute{Description: "The backup ID.", Computed: true},
						"policy_id":       schema.StringAttribute{Description: "The policy that produced the backup, when it came from a schedule.", Computed: true},
						"trigger":         schema.StringAttribute{Description: "What started the backup: `manual` or `schedule`.", Computed: true},
						"type":            schema.StringAttribute{Description: "The backup type that ran: `full`, `incremental` or `wal`.", Computed: true},
						"requested_type":  schema.StringAttribute{Description: "The backup type that was asked for. Differs from `type` when the backend could not run it and fell back.", Computed: true},
						"fallback_reason": schema.StringAttribute{Description: "Why the backend ran a different type than requested. Empty when it ran as requested.", Computed: true},
						"status":          schema.StringAttribute{Description: "`pending`, `running`, `completed`, `failed` or `verified`.", Computed: true},
						"size_bytes":      schema.Int64Attribute{Description: "The backup size in bytes.", Computed: true},
						"started_at":      schema.StringAttribute{Description: "When the backup started, in ISO 8601 format.", Computed: true},
						"completed_at":    schema.StringAttribute{Description: "When the backup finished, in ISO 8601 format.", Computed: true},
						"verified_at":     schema.StringAttribute{Description: "When the backup was verified, in ISO 8601 format.", Computed: true},
						"expires_at":      schema.StringAttribute{Description: "When the backup expires, in ISO 8601 format.", Computed: true},
						"log_position":    schema.StringAttribute{Description: "The transaction-log position the backup anchors at, in the engine's own notation. Empty for backup types that carry none.", Computed: true},
						"error":           schema.StringAttribute{Description: "The error message when the backup failed.", Computed: true},
						"created_at":      schema.StringAttribute{Description: "The creation timestamp in ISO 8601 format.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *backupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *backupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config backupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	var queries []string
	if !config.Queries.IsNull() && !config.Queries.IsUnknown() {
		resp.Diagnostics.Append(config.Queries.ElementsAs(ctx, &queries, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	list, err := clientFor(d.clients, d.engine, projectID).ListBackups(config.DatabaseID.ValueString(), queries)
	if err != nil {
		resp.Diagnostics.AddError("Error listing dedicated database backups", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.Total = types.Int64Value(int64(list.Total))
	config.Backups = make([]backupItem, 0, len(list.Backups))
	for _, backup := range list.Backups {
		config.Backups = append(config.Backups, backupItem{
			ID:             types.StringValue(backup.Id),
			PolicyID:       types.StringValue(backup.PolicyId),
			Trigger:        types.StringValue(backup.Trigger),
			Type:           types.StringValue(backup.Type),
			RequestedType:  types.StringValue(backup.RequestedType),
			FallbackReason: types.StringValue(backup.FallbackReason),
			Status:         types.StringValue(backup.Status),
			SizeBytes:      types.Int64Value(int64(backup.SizeBytes)),
			StartedAt:      types.StringValue(backup.StartedAt),
			CompletedAt:    types.StringValue(backup.CompletedAt),
			VerifiedAt:     types.StringValue(backup.VerifiedAt),
			ExpiresAt:      types.StringValue(backup.ExpiresAt),
			LogPosition:    types.StringValue(backup.LogPosition),
			Error:          types.StringValue(backup.Error),
			CreatedAt:      types.StringValue(backup.CreatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
