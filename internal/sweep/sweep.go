// Package sweep deletes resources an interrupted acceptance run left behind.
//
// A test that is canceled, panics, or loses its credential mid-apply never
// reaches its destroy step, and what it created stays. Against a disposable CI
// instance that costs nothing, because the whole instance is thrown away. Against
// a shared or Cloud project it accumulates, and some of it is billable.
//
// This package is imported only from test files. The Makefile asserts that:
// sweeper-linked proves the symbols reach the test binary, sweeper-unlinked
// proves they do not reach the release binary. Code that deletes infrastructure
// without a Terraform plan in front of it has no business shipping to users.
package sweep

import (
	"fmt"
	"os"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
)

// Prefix is the name and ID prefix a sweepable test resource carries.
//
// Sweeping is prefix-scoped rather than "everything in the project", because
// the project a developer points the suite at is not guaranteed to contain
// nothing else. A resource without this prefix is never touched, which means a
// test whose resources are not named with it is not swept -- deliberately, since
// the alternative failure mode is deleting someone's data.
const Prefix = "tf-acc-test"

// Sweepable reports whether a resource may be deleted by a sweeper, given
// every identifier it is known by. Matching on any of them tolerates a resource
// whose name carries the prefix but whose server-assigned ID does not, and the
// reverse.
func Sweepable(identifiers ...string) bool {
	for _, id := range identifiers {
		if strings.Contains(id, Prefix) {
			return true
		}
	}
	return false
}

// Config is the environment a sweep run needs.
type Config struct {
	Endpoint  string
	APIKey    string
	ProjectID string
}

// LoadConfig reads the sweep configuration, refusing to run without an
// explicitly nominated project.
//
// APPWRITE_PROJECT_ID is deliberately not consulted. It is whatever the last
// terraform run or acceptance test happened to leave exported, and a sweeper
// that inherits it will one day inherit a production project. Naming the
// project again, in a variable that exists for no other purpose, is the
// confirmation step.
func LoadConfig() (Config, error) {
	cfg := Config{
		Endpoint:  os.Getenv("APPWRITE_ENDPOINT"),
		APIKey:    os.Getenv("APPWRITE_API_KEY"),
		ProjectID: os.Getenv("APPWRITE_SWEEP_PROJECT_ID"),
	}

	var missing []string
	if cfg.Endpoint == "" {
		missing = append(missing, "APPWRITE_ENDPOINT")
	}
	if cfg.APIKey == "" {
		missing = append(missing, "APPWRITE_API_KEY")
	}
	if cfg.ProjectID == "" {
		missing = append(missing, "APPWRITE_SWEEP_PROJECT_ID")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf(
			"sweeping needs %s; APPWRITE_SWEEP_PROJECT_ID names the project to sweep and is separate from "+
				"APPWRITE_PROJECT_ID on purpose, so a sweep cannot inherit whichever project was last exported",
			strings.Join(missing, ", "),
		)
	}
	return cfg, nil
}

// Client returns a client scoped to the project being swept.
func (c Config) Client() sdkclient.Client {
	return appwrite.NewClient(
		appwrite.WithEndpoint(c.Endpoint),
		appwrite.WithKey(c.APIKey),
		appwrite.WithProject(c.ProjectID),
	)
}

// errs accumulates per-resource failures so one undeletable resource does not
// hide the rest. A sweeper that stops at the first error leaves most of the
// mess in place and has to be run repeatedly.
type errs struct {
	swept  int
	failed []string
}

func (e *errs) ok() { e.swept++ }

func (e *errs) fail(kind, id string, err error) {
	e.failed = append(e.failed, fmt.Sprintf("%s %s: %v", kind, id, err))
}

func (e *errs) result(kind string) error {
	fmt.Printf("swept %d %s\n", e.swept, kind)
	if len(e.failed) > 0 {
		return fmt.Errorf("could not sweep %d %s:\n  - %s", len(e.failed), kind, strings.Join(e.failed, "\n  - "))
	}
	return nil
}
