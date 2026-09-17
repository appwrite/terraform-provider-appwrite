package dedicated

import (
	"context"
	"fmt"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ action.Action              = &backupAction{}
	_ action.ActionWithConfigure = &backupAction{}
)

type backupAction struct {
	engine  Engine
	clients *common.AppwriteClients
}

type backupActionModel struct {
	DatabaseID types.String `tfsdk:"database_id"`
	Type       types.String `tfsdk:"type"`
	ProjectID  types.String `tfsdk:"project_id"`
}

// NewBackupAction builds the on-demand backup action for one engine, following
// the same one-implementation-registered-three-times shape as the resources.
func NewBackupAction(engine Engine) func() action.Action {
	return func() action.Action {
		return &backupAction{engine: engine}
	}
}

func (a *backupAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = fmt.Sprintf("%s_%s_backup", req.ProviderTypeName, a.engine)
}

func (a *backupAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf(
			"Takes an on-demand backup of a dedicated %s database. Scheduled backups belong in "+
				"appwrite_%s_backup_policy; this is for the backup you want taken right now, before a "+
				"migration or a risky change.", a.engine.Label(), a.engine),
		Attributes: map[string]schema.Attribute{
			"database_id": schema.StringAttribute{
				Description: "The ID of the database to back up.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"type": schema.StringAttribute{
				Description: "The backup type. Defaults to the server's own default.",
				Optional:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
			},
		},
	}
}

func (a *backupAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Action Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	a.clients = clients
}

func (a *backupAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var config backupActionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(a.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	api := newDatabaseAPI(a.engine, a.clients.ClientForProject(projectID))

	var backupType *string
	if v := config.Type; !v.IsNull() && !v.IsUnknown() {
		s := v.ValueString()
		backupType = &s
	}

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Requesting backup of %s database %s", a.engine.Label(), config.DatabaseID.ValueString()),
	})

	backup, err := api.CreateBackup(config.DatabaseID.ValueString(), backupType)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error backing up %s database", a.engine.Label()),
			common.FormatError(err),
		)
		return
	}

	// The server runs the backup asynchronously, so this reports that it was
	// accepted rather than that it finished.
	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Backup %s requested with status %q", backup.Id, backup.Status),
	})
}
