package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	"github.com/appwrite/sdk-for-go/v2/backups"
	"github.com/appwrite/sdk-for-go/v2/client"
	"github.com/appwrite/sdk-for-go/v2/messaging"
	"github.com/appwrite/sdk-for-go/v2/storage"
	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/appwrite/sdk-for-go/v2/teams"
	"github.com/appwrite/sdk-for-go/v2/users"
	"github.com/appwrite/sdk-for-go/v2/webhooks"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppwriteClients holds the configured SDK service clients and base configuration
// for creating per-project clients when organization-level API keys are used.
type AppwriteClients struct {
	// Pre-configured service clients using the provider-level project_id.
	TablesDB  *tablesdb.TablesDB
	Storage   *storage.Storage
	Messaging *messaging.Messaging
	Users     *users.Users
	Teams     *teams.Teams
	Backups   *backups.Backups
	Webhooks  *webhooks.Webhooks

	// Base configuration for creating per-project clients (no project set).
	BaseOptions []client.ClientOption
	// ProjectID is the provider-level default project ID.
	ProjectID string
}

// ClientForProject creates a new SDK client targeting a specific project.
func (ac *AppwriteClients) ClientForProject(projectID string) client.Client {
	opts := make([]client.ClientOption, len(ac.BaseOptions))
	copy(opts, ac.BaseOptions)
	opts = append(opts, appwrite.WithProject(projectID))
	return appwrite.NewClient(opts...)
}

// ResolveProjectID returns the resource-level project_id if set, otherwise the provider default.
// Returns an error if neither is set.
func ResolveProjectID(clients *AppwriteClients, resourceProjectID types.String) (string, error) {
	if !resourceProjectID.IsNull() && !resourceProjectID.IsUnknown() && resourceProjectID.ValueString() != "" {
		return resourceProjectID.ValueString(), nil
	}
	if clients.ProjectID != "" {
		return clients.ProjectID, nil
	}
	return "", fmt.Errorf("project_id must be set either on the provider or the resource")
}

// GetTablesDB returns a TablesDB client for the resolved project.
func (ac *AppwriteClients) GetTablesDB(resourceProjectID types.String) (*tablesdb.TablesDB, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.TablesDB != nil {
		return ac.TablesDB, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewTablesDB(c), projectID, nil
}

// GetStorage returns a Storage client for the resolved project.
func (ac *AppwriteClients) GetStorage(resourceProjectID types.String) (*storage.Storage, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Storage != nil {
		return ac.Storage, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewStorage(c), projectID, nil
}

// GetMessaging returns a Messaging client for the resolved project.
func (ac *AppwriteClients) GetMessaging(resourceProjectID types.String) (*messaging.Messaging, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Messaging != nil {
		return ac.Messaging, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewMessaging(c), projectID, nil
}

// GetUsers returns a Users client for the resolved project.
func (ac *AppwriteClients) GetUsers(resourceProjectID types.String) (*users.Users, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Users != nil {
		return ac.Users, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewUsers(c), projectID, nil
}

// GetTeams returns a Teams client for the resolved project.
func (ac *AppwriteClients) GetTeams(resourceProjectID types.String) (*teams.Teams, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Teams != nil {
		return ac.Teams, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewTeams(c), projectID, nil
}

// GetBackups returns a Backups client for the resolved project.
func (ac *AppwriteClients) GetBackups(resourceProjectID types.String) (*backups.Backups, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Backups != nil {
		return ac.Backups, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewBackups(c), projectID, nil
}

// GetWebhooks returns a Webhooks client for the resolved project.
func (ac *AppwriteClients) GetWebhooks(resourceProjectID types.String) (*webhooks.Webhooks, string, error) {
	projectID, err := ResolveProjectID(ac, resourceProjectID)
	if err != nil {
		return nil, "", err
	}
	if projectID == ac.ProjectID && ac.Webhooks != nil {
		return ac.Webhooks, projectID, nil
	}
	c := ac.ClientForProject(projectID)
	return appwrite.NewWebhooks(c), projectID, nil
}

// ProjectIDAttribute returns the shared schema attribute for project_id on resources.
func ProjectIDAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description:   "The Appwrite project ID. Defaults to the provider-level project_id.",
		Optional:      true,
		Computed:      true,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
	}
}

// IsNotFoundError checks if an Appwrite SDK error is a 404.
func IsNotFoundError(err error) bool {
	if appErr, ok := err.(*client.AppwriteError); ok {
		return appErr.GetStatusCode() == 404
	}
	return false
}

// IsColumnNotAvailableError checks if the error is due to a column still being processed.
func IsColumnNotAvailableError(err error) bool {
	if appErr, ok := err.(*client.AppwriteError); ok {
		return appErr.GetStatusCode() == 400 && strings.Contains(appErr.GetMessage(), "not yet available")
	}
	return false
}

// FormatError returns a detailed error string including status code and response body.
func FormatError(err error) string {
	if appErr, ok := err.(*client.AppwriteError); ok {
		return fmt.Sprintf("%s (status: %d, response: %s)", appErr.GetMessage(), appErr.GetStatusCode(), appErr.GetResponse())
	}
	return err.Error()
}

// DecodeColumn decodes the raw *interface{} response from GetColumn into a typed model.
func DecodeColumn(raw *interface{}, target interface{}) error {
	if raw == nil {
		return fmt.Errorf("nil response from GetColumn")
	}
	switch v := (*raw).(type) {
	case string:
		return json.Unmarshal([]byte(v), target)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal GetColumn response: %w", err)
		}
		return json.Unmarshal(b, target)
	}
}

// GetColumnStatus extracts the status field from a raw GetColumn response.
func GetColumnStatus(raw *interface{}) (string, error) {
	if raw == nil {
		return "", fmt.Errorf("nil response")
	}
	var generic map[string]interface{}
	if err := DecodeColumn(raw, &generic); err != nil {
		return "", err
	}
	status, _ := generic["status"].(string)
	return status, nil
}

// ImportColumnState parses a "database_id/table_id/key" import ID into state.
func ImportColumnState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: database_id/table_id/key, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("table_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[2])...)
}
