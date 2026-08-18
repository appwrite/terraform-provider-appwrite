package docdb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func optString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func optInt(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int(v.ValueInt64())
	return &i
}

func optBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// splitImportID splits an import ID into exactly n non-empty parts. Appwrite
// IDs never contain a slash, so a different part count means the ID was
// mistyped; rejecting it beats silently mis-assigning segments and importing
// the wrong object.
func splitImportID(importID string, n int) ([]string, bool) {
	parts := strings.Split(importID, "/")
	if len(parts) != n {
		return nil, false
	}
	for _, part := range parts {
		if part == "" {
			return nil, false
		}
	}
	return parts, true
}

// The framework distinguishes a null list from an empty one, while the API
// returns an omitted list as nil. Normalising to empty stops a value flipping
// between null and [] on refresh.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilInts(values []int) []int {
	if values == nil {
		return []int{}
	}
	return values
}

// Provisioning a dedicated backing takes time, so the create call returns while
// the database is still coming up.
const (
	provisionTimeout = 30 * time.Minute
	pollInterval     = 3 * time.Second
)

// transitionalStatuses are the states the server moves through on its own.
// A database on a shared pool reports no status at all, which is terminal:
// there is no backing to wait for.
var transitionalStatuses = map[string]bool{
	"provisioning": true,
	"scaling":      true,
	"restoring":    true,
}

var terminalFailureStatuses = map[string]bool{
	"failed":  true,
	"deleted": true,
}

// waitForReady polls a database until its dedicated backing settles, so that a
// collection is never created against a database that is still coming up.
func waitForReady(ctx context.Context, get func() (*models.Database, error), databaseID string) (*models.Database, error) {
	deadline := time.After(provisionTimeout)
	for {
		database, err := get()
		if err != nil {
			return nil, fmt.Errorf("error checking status of database %q: %w", databaseID, err)
		}

		switch {
		case terminalFailureStatuses[database.Status]:
			return nil, fmt.Errorf("database %q entered the %s state", databaseID, database.Status)
		case !transitionalStatuses[database.Status]:
			return database, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for database %q to finish %s", databaseID, database.Status)
		case <-deadline:
			return nil, fmt.Errorf("database %q was still %s after %s", databaseID, database.Status, provisionTimeout)
		case <-time.After(pollInterval):
		}
	}
}
