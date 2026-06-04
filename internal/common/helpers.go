package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v5/appwrite"
	"github.com/appwrite/sdk-for-go/v5/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AppwriteClients holds the base configuration for creating SDK clients.
type AppwriteClients struct {
	// BaseOptions contains the client options without a project (endpoint + key + self_signed).
	BaseOptions []client.ClientOption
	// ProjectID is the provider-level default project ID.
	ProjectID string
}

// WithUserAgent returns a ClientOption that sets the User-Agent header to identify
// Terraform provider traffic. This is required for HashiCorp partner providers.
func WithUserAgent(version string) client.ClientOption {
	return func(clt *client.Client) error {
		clt.Headers["user-agent"] = fmt.Sprintf("terraform-provider-appwrite/%s", version)
		return nil
	}
}

// ClientForProject creates a new SDK client targeting a specific project.
func (ac *AppwriteClients) ClientForProject(projectID string) client.Client {
	opts := make([]client.ClientOption, 0, len(ac.BaseOptions)+1)
	opts = append(opts, ac.BaseOptions...)
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
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return appErr.GetStatusCode() == 404
	}
	return false
}

// IsColumnNotAvailableError checks if the error is due to a column still being processed.
func IsColumnNotAvailableError(err error) bool {
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return appErr.GetStatusCode() == 400 && strings.Contains(appErr.GetMessage(), "not yet available")
	}
	return false
}

// FormatError returns a detailed error string including status code and response body.
func FormatError(err error) string {
	var appErr *client.AppwriteError
	if errors.As(err, &appErr) {
		return fmt.Sprintf("%s (status: %d, response: %s)", appErr.GetMessage(), appErr.GetStatusCode(), appErr.GetResponse())
	}
	return err.Error()
}

// GetColumnRaw fetches a column using a raw API call, bypassing the SDK's
// type-matching logic which doesn't handle all column types (e.g. text,
// longtext, mediumtext, varchar, point, line, polygon).
func GetColumnRaw(c client.Client, databaseID, tableID, key string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/tablesdb/%s/tables/%s/columns/%s", databaseID, tableID, key)
	resp, err := c.Call("GET", path, map[string]interface{}{}, map[string]interface{}{
		"databaseId": databaseID,
		"tableId":    tableID,
		"key":        key,
	})
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resp.Type, "application/json") {
		return nil, fmt.Errorf("unexpected response type: %s", resp.Type)
	}
	resultStr, ok := resp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected response result type: %T", resp.Result)
	}
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resultStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal column response: %w", err)
	}
	return result, nil
}

// AttrCheck holds the result of an attribute mismatch check.
type AttrCheck struct {
	Summary  string
	Detail   string
	Mismatch bool
}

// CheckBoolNotIgnored returns an error diagnostic if the planned bool value differs from
// the API response, indicating the server doesn't support this attribute.
func CheckBoolNotIgnored(planned types.Bool, actual bool, attrName string, resourceDesc string) AttrCheck {
	if planned.IsNull() || planned.IsUnknown() {
		return AttrCheck{}
	}
	if planned.ValueBool() != actual {
		return AttrCheck{
			Summary: fmt.Sprintf("Attribute %q not supported", attrName),
			Detail: fmt.Sprintf(
				"The server did not accept the %q setting for %s. "+
					"This feature may not be supported on this Appwrite server version. "+
					"Remove the %q attribute from your configuration or upgrade your server.",
				attrName, resourceDesc, attrName,
			),
			Mismatch: true,
		}
	}
	return AttrCheck{}
}

// CheckStringNotIgnored returns an error diagnostic if the planned string value differs from
// the API response, indicating the server doesn't support this attribute.
func CheckStringNotIgnored(planned types.String, actual string, attrName string, resourceDesc string) AttrCheck {
	if planned.IsNull() || planned.IsUnknown() {
		return AttrCheck{}
	}
	if planned.ValueString() != actual {
		return AttrCheck{
			Summary: fmt.Sprintf("Attribute %q not supported", attrName),
			Detail: fmt.Sprintf(
				"The server did not accept the %q setting for %s (sent %q, got %q). "+
					"This feature may not be supported on this Appwrite server version. "+
					"Remove the %q attribute from your configuration or upgrade your server.",
				attrName, resourceDesc, planned.ValueString(), actual, attrName,
			),
			Mismatch: true,
		}
	}
	return AttrCheck{}
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

// WaitForDeploymentReady polls a deployment until its status becomes "ready",
// "failed", or "canceled". Returns nil on "ready", error otherwise.
func WaitForDeploymentReady(ctx context.Context, getDeployment func() (string, error), deploymentID string) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for deployment %q to become ready", deploymentID)
		default:
		}

		status, err := getDeployment()
		if err != nil {
			return fmt.Errorf("error checking deployment %q status: %w", deploymentID, err)
		}

		switch status {
		case "ready":
			return nil
		case "failed":
			return fmt.Errorf("deployment %q failed", deploymentID)
		case "canceled":
			return fmt.Errorf("deployment %q was canceled", deploymentID)
		}

		time.Sleep(2 * time.Second)
	}
}

// WaitForColumnAvailable polls a column until its status becomes "available",
// with a maximum wait of 5 minutes.
func WaitForColumnAvailable(ctx context.Context, getColumn func() (interface{}, error), key string) error {
	deadline := time.After(5 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for column %q to become available", key)
		case <-deadline:
			return fmt.Errorf("column %q did not become available within 5m", key)
		default:
		}

		raw, err := getColumn()
		if err != nil {
			return fmt.Errorf("error checking column %q status: %w", key, err)
		}

		var status string
		if m, ok := raw.(map[string]interface{}); ok {
			status, _ = m["status"].(string)
		}
		switch status {
		case "available":
			return nil
		case "failed", "stuck":
			return fmt.Errorf("column %q is in %q state", key, status)
		}

		time.Sleep(1 * time.Second)
	}
}
