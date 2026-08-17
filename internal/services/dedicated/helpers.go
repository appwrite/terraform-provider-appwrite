package dedicated

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// Provisioning a dedicated database allocates real infrastructure, so the waits
// are minutes rather than seconds. Creation is the slowest path; resizes and
// restores reuse the shorter budget.
const (
	createStableTimeout = 45 * time.Minute
	updateStableTimeout = 30 * time.Minute
)

// pollInterval is a variable so tests can drive the wait loop without sleeping
// through real provisioning delays.
var pollInterval = 5 * time.Second

// transitionalStatuses are the states the server passes through on its own.
// Anything else is a resting state that a dependent resource can build on --
// including a status this provider does not know about, which is reported
// rather than polled until the timeout expires.
var transitionalStatuses = map[string]bool{
	"provisioning": true,
	"scaling":      true,
	"restoring":    true,
}

// terminalFailureStatuses are resting states that mean the database is not
// usable. Writing one of these to state would hand downstream resources a
// database that will never accept a connection.
var terminalFailureStatuses = map[string]bool{
	"failed":  true,
	"deleted": true,
}

// waitForStable polls a dedicated database until it settles into a resting
// state, so that dependent resources are not handed a half-built database. A
// database that reaches "failed" is reported with the server's own error text.
func waitForStable(ctx context.Context, get func() (*models.DedicatedDatabase, error), databaseID string, timeout time.Duration) (*models.DedicatedDatabase, error) {
	deadline := time.After(timeout)
	for {
		db, err := get()
		if err != nil {
			return nil, fmt.Errorf("error checking status of database %q: %w", databaseID, err)
		}

		switch {
		case terminalFailureStatuses[db.Status]:
			if db.Error != "" {
				return nil, fmt.Errorf("database %q entered the %s state: %s", databaseID, db.Status, db.Error)
			}
			return nil, fmt.Errorf("database %q entered the %s state", databaseID, db.Status)
		case !transitionalStatuses[db.Status]:
			return db, nil
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for database %q to finish %s", databaseID, db.Status)
		case <-deadline:
			return nil, fmt.Errorf("database %q was still %s after %s", databaseID, db.Status, timeout)
		case <-time.After(pollInterval):
		}
	}
}

// clientFor resolves the project and returns an engine adapter bound to it.
func clientFor(clients *common.AppwriteClients, engine Engine, projectID string) databaseAPI {
	return newDatabaseAPI(engine, clients.ClientForProject(projectID))
}

// splitImportID splits an import ID into exactly two non-empty parts. Appwrite
// IDs never contain a slash, so a third segment means the user mistyped the
// import ID; rejecting it beats silently folding the extra segments into the
// second part and importing some other object.
func splitImportID(importID string) (first, second string, ok bool) {
	first, second, ok = strings.Cut(importID, "/")
	if !ok || first == "" || second == "" || strings.Contains(second, "/") {
		return "", "", false
	}
	return first, second, true
}
