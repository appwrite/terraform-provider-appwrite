package dedicated

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/models"
)

func TestSplitImportID(t *testing.T) {
	for _, tc := range []struct {
		name       string
		importID   string
		wantFirst  string
		wantSecond string
		wantOK     bool
	}{
		{"two parts", "db/policy", "db", "policy", true},
		{"no separator", "db", "", "", false},
		{"empty first", "/policy", "", "", false},
		{"empty second", "db/", "", "", false},
		{"empty", "", "", "", false},
		// Appwrite IDs never contain a slash, so a third segment is a typo.
		// Folding it into the second part would import a different object.
		{"three parts", "postgresql/db/policy", "", "", false},
		{"many parts", "a/b/c/d", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first, second, ok := splitImportID(tc.importID)
			if ok != tc.wantOK || first != tc.wantFirst || second != tc.wantSecond {
				t.Errorf("splitImportID(%q) = (%q, %q, %v), want (%q, %q, %v)",
					tc.importID, first, second, ok, tc.wantFirst, tc.wantSecond, tc.wantOK)
			}
		})
	}
}

func TestWaitForStable(t *testing.T) {
	restore := pollInterval
	pollInterval = time.Millisecond
	t.Cleanup(func() { pollInterval = restore })

	for _, tc := range []struct {
		name        string
		statuses    []string
		dbError     string
		wantStatus  string
		wantErr     bool
		wantErrText string
	}{
		{name: "already ready", statuses: []string{"ready"}, wantStatus: "ready"},
		{name: "settles after provisioning", statuses: []string{"provisioning", "provisioning", "ready"}, wantStatus: "ready"},
		{name: "settles after scaling", statuses: []string{"scaling", "ready"}, wantStatus: "ready"},
		{name: "paused is a resting state", statuses: []string{"paused"}, wantStatus: "paused"},
		{name: "inactive is a resting state", statuses: []string{"inactive"}, wantStatus: "inactive"},
		{
			name: "failed reports the server error", statuses: []string{"provisioning", "failed"},
			dbError: "no capacity in region", wantErr: true, wantErrText: "no capacity in region",
		},
		{name: "deleted is a failure", statuses: []string{"deleted"}, wantErr: true, wantErrText: "deleted"},
		// An unrecognized status must resolve rather than poll to the timeout.
		{name: "unknown status resolves", statuses: []string{"unhealthy"}, wantStatus: "unhealthy"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls int
			get := func() (*models.DedicatedDatabase, error) {
				status := tc.statuses[min(calls, len(tc.statuses)-1)]
				calls++
				return &models.DedicatedDatabase{Id: "db", Status: status, Error: tc.dbError}, nil
			}

			db, err := waitForStable(t.Context(), get, "db", time.Minute)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got status %q", db.Status)
				}
				if !strings.Contains(err.Error(), tc.wantErrText) {
					t.Errorf("error = %q, want it to contain %q", err, tc.wantErrText)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if db.Status != tc.wantStatus {
				t.Errorf("status = %q, want %q", db.Status, tc.wantStatus)
			}
		})
	}
}

func TestWaitForStablePropagatesReadError(t *testing.T) {
	get := func() (*models.DedicatedDatabase, error) {
		return nil, errors.New("network is unreachable")
	}
	if _, err := waitForStable(t.Context(), get, "db", time.Minute); err == nil {
		t.Fatal("expected an error")
	} else if !strings.Contains(err.Error(), "network is unreachable") {
		t.Errorf("error = %q, want it to wrap the read failure", err)
	}
}

// A context canceled mid-provisioning has to stop the loop rather than hold
// the apply open for the full timeout.
func TestWaitForStableHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	get := func() (*models.DedicatedDatabase, error) {
		return &models.DedicatedDatabase{Id: "db", Status: "provisioning"}, nil
	}
	if _, err := waitForStable(ctx, get, "db", time.Hour); err == nil {
		t.Fatal("expected an error once the context is canceled")
	}
}
