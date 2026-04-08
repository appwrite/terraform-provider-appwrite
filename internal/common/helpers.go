package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/client"
	"github.com/appwrite/sdk-for-go/v2/tablesdb"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// AppwriteClients holds the configured SDK service clients.
type AppwriteClients struct {
	TablesDB *tablesdb.TablesDB
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
