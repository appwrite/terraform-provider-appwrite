// Package dedicateddatabase implements the appwrite_dedicated_database and
// appwrite_dedicated_database_backup_policy resources.
//
// It is built against the unreleased dedicated-databases SDK fork
// (github.com/aw-tests/sdk-for-go), whose module path differs from the released
// SDK. To avoid a module-identity conflict we do NOT touch the rest of the
// provider's imports; this package alone talks to the fork and builds its own
// client from the raw provider credentials. When the feature ships in the
// official SDK, swap the import prefix below back to appwrite/sdk-for-go and
// drop the fork require from go.mod.
//
// The three engines (postgresql, mysql, mongo) expose byte-identical Create/
// Get/Update/Delete + backup-policy APIs — same params, same response models —
// differing only in the URL segment. So instead of triplicating ~30 typed
// option builders per method, we issue the raw REST calls keyed by engine and
// decode into the shared SDK models. This mirrors the existing GetColumnRaw
// helper in internal/common.
package dedicateddatabase

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	fwappwrite "github.com/aw-tests/sdk-for-go/v6/appwrite"
	fwclient "github.com/aw-tests/sdk-for-go/v6/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// setStr/setInt/setBool add a param only when the attribute is explicitly set
// (known and non-null), so omitted optional attributes fall back to server
// defaults rather than being sent as zero values.
func setStr(params map[string]interface{}, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		params[key] = v.ValueString()
	}
}

func setInt(params map[string]interface{}, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		params[key] = int(v.ValueInt64())
	}
}

func setBool(params map[string]interface{}, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		params[key] = v.ValueBool()
	}
}

// splitTwo splits "a/b" into its two non-empty halves.
func splitTwo(s string) (string, string, bool) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// engines maps the terraform `engine` value to its REST path segment.
var engines = map[string]string{
	"postgresql": "postgresql",
	"mysql":      "mysql",
	"mongo":      "mongo",
}

// engineClient builds a fork SDK client scoped to the given project.
func engineClient(clients *common.AppwriteClients, projectID string) fwclient.Client {
	opts := []fwclient.ClientOption{
		fwappwrite.WithEndpoint(clients.Endpoint),
		fwappwrite.WithKey(clients.APIKey),
		fwappwrite.WithProject(projectID),
	}
	if clients.SelfSigned {
		opts = append(opts, fwappwrite.WithSelfSigned(true))
	}
	return fwappwrite.NewClient(opts...)
}

// apiCall issues a raw request and decodes the JSON response into T. Pass nil
// for out when the response body is not needed (e.g. deletes).
func apiCall[T any](c fwclient.Client, userAgent, method, path string, params map[string]interface{}, out *T) error {
	headers := map[string]interface{}{
		"X-Appwrite-Project": c.Config["project"],
		"content-type":       "application/json",
		"accept":             "application/json",
		"user-agent":         userAgent,
	}
	resp, err := c.Call(method, path, headers, params)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	body, ok := resp.Result.(string)
	if !ok {
		return fmt.Errorf("unexpected response result type: %T", resp.Result)
	}
	if err := json.Unmarshal([]byte(body), out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}

// waitForDatabaseReady polls the database until its status leaves
// "provisioning"/"scaling"/"restoring", returning an error on a failed status
// or when ctx is canceled. Provisioning a dedicated database is asynchronous;
// connection details are only populated once it is ready.
func waitForDatabaseReady(ctx context.Context, get func() (string, error), databaseID string) error {
	deadline := time.After(30 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("canceled while waiting for database %q to become ready", databaseID)
		case <-deadline:
			return fmt.Errorf("database %q did not become ready within 30m", databaseID)
		default:
		}

		status, err := get()
		if err != nil {
			return fmt.Errorf("error checking database %q status: %w", databaseID, err)
		}
		switch status {
		case "ready", "inactive", "paused":
			return nil
		case "failed":
			return fmt.Errorf("database %q entered failed state", databaseID)
		case "deleted":
			return fmt.Errorf("database %q was deleted during provisioning", databaseID)
		}

		time.Sleep(5 * time.Second)
	}
}

// validEngines is the sorted list used in validation and doc messages.
func validEngines() string {
	keys := make([]string, 0, len(engines))
	for k := range engines {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
