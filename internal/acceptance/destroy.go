package acceptance

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	sdkclient "github.com/appwrite/sdk-for-go/v7/client"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// fetchFunc asks the server for the resource an instance's state describes. It
// returns the SDK error unchanged, because the whole point is to tell a 404
// apart from a transport failure: the first means the delete worked, the second
// means the check could not be performed and must not be reported as a pass.
type fetchFunc func(t *testing.T, attrs map[string]string) error

// destroyChecks maps a resource type to the call that proves it is gone.
//
// Deliberately central rather than per-service: a registry each service package
// could append to is a registry a new service can forget to append to, and the
// gap would be invisible. Here the coverage test below can see the whole set at
// once and compare it against what the provider actually registers.
var destroyChecks = map[string]fetchFunc{
	"appwrite_webhook": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewWebhooks(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_auth_user": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewUsers(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_auth_team": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewTeams(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_messaging_topic": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewMessaging(projectScoped(t, attrs)).GetTopic(attrs["id"])
		return err
	},
	"appwrite_messaging_provider": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewMessaging(projectScoped(t, attrs)).GetProvider(attrs["id"])
		return err
	},
	"appwrite_messaging_subscriber": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewMessaging(projectScoped(t, attrs)).GetSubscriber(attrs["topic_id"], attrs["id"])
		return err
	},
	"appwrite_storage_bucket": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewStorage(projectScoped(t, attrs)).GetBucket(attrs["id"])
		return err
	},
	"appwrite_storage_file": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewStorage(projectScoped(t, attrs)).GetFile(attrs["bucket_id"], attrs["id"])
		return err
	},
	"appwrite_function": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewFunctions(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_function_variable": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewFunctions(projectScoped(t, attrs)).GetVariable(attrs["function_id"], attrs["id"])
		return err
	},
	"appwrite_function_deployment": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewFunctions(projectScoped(t, attrs)).GetDeployment(attrs["function_id"], attrs["id"])
		return err
	},
	"appwrite_site": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewSites(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_site_variable": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewSites(projectScoped(t, attrs)).GetVariable(attrs["site_id"], attrs["id"])
		return err
	},
	"appwrite_site_deployment": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewSites(projectScoped(t, attrs)).GetDeployment(attrs["site_id"], attrs["id"])
		return err
	},
	"appwrite_tablesdb": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewTablesDB(projectScoped(t, attrs)).Get(attrs["id"])
		return err
	},
	"appwrite_tablesdb_table": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewTablesDB(projectScoped(t, attrs)).GetTable(attrs["database_id"], attrs["id"])
		return err
	},
	"appwrite_tablesdb_index": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewTablesDB(projectScoped(t, attrs)).GetIndex(attrs["database_id"], attrs["table_id"], attrs["key"])
		return err
	},
	"appwrite_tablesdb_row": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewTablesDB(projectScoped(t, attrs)).GetRow(attrs["database_id"], attrs["table_id"], attrs["id"])
		return err
	},
	// Columns are read through the raw API rather than the SDK, because the
	// SDK's type matching does not cover every column type. The destroy check
	// has to use the same path or it would report a 404 for types the SDK
	// simply cannot decode.
	"appwrite_tablesdb_column": func(t *testing.T, attrs map[string]string) error {
		_, err := common.GetColumnRaw(projectScoped(t, attrs), attrs["database_id"], attrs["table_id"], attrs["key"])
		return err
	},
	"appwrite_proxy_rule": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewProxy(projectScoped(t, attrs)).GetRule(attrs["id"])
		return err
	},
	"appwrite_backup_policy": func(t *testing.T, attrs map[string]string) error {
		_, err := appwrite.NewBackups(projectScoped(t, attrs)).GetPolicy(attrs["id"])
		return err
	},
}

const (
	reasonBillable  = "billable; tests skipped in CI"
	reasonCloudOnly = "Cloud-only; tests skipped in CI"
)

// destroyCheckGaps lists resource types with no destroy check, each with the
// reason it is absent. Present so the coverage test can tell a deliberate
// omission from an oversight: a new resource type appears in neither map and
// fails the test, which is the point.
//
// Shrink this list; do not grow it.
var destroyCheckGaps = map[string]string{
	// The console project a test creates is also the scope every other check
	// authenticates against. Checking its absence needs the organization
	// credential rather than the project one, so it takes its own path.
	"appwrite_project": "needs the organization credential, not the project one",
	// Project keys are already covered by their own test asserting the key
	// stops working after destroy, which is a stronger check than absence.
	"appwrite_project_key": "covered by a stronger credential-revocation assertion in its own test",
	// Dedicated database resources provision billable infrastructure and are
	// skipped unless APPWRITE_DEDICATED_DATABASE_TESTS is set, so a check here
	// would never run in CI. Worth adding alongside the first CI run that
	// exercises them.
	"appwrite_postgresql_database":       reasonBillable,
	"appwrite_mysql_database":            reasonBillable,
	"appwrite_mongo_database":            reasonBillable,
	"appwrite_postgresql_backup_policy":  reasonBillable,
	"appwrite_mysql_backup_policy":       reasonBillable,
	"appwrite_mongo_backup_policy":       reasonBillable,
	"appwrite_postgresql_backup_storage": reasonBillable,
	"appwrite_mysql_backup_storage":      reasonBillable,
	"appwrite_mongo_backup_storage":      reasonBillable,
	"appwrite_postgresql_branch":         reasonBillable,
	"appwrite_mysql_branch":              reasonBillable,
	"appwrite_mongo_branch":              reasonBillable,
	"appwrite_postgresql_pooler":         reasonBillable,
	"appwrite_mysql_pooler":              reasonBillable,
	"appwrite_postgresql_extension":      reasonBillable,
	// DocumentsDB and VectorsDB are Cloud-only and skipped unless
	// APPWRITE_CLOUD_TESTS is set.
	"appwrite_documentsdb":            reasonCloudOnly,
	"appwrite_documentsdb_collection": reasonCloudOnly,
	"appwrite_documentsdb_index":      reasonCloudOnly,
	"appwrite_documentsdb_document":   reasonCloudOnly,
	"appwrite_vectorsdb":              reasonCloudOnly,
	"appwrite_vectorsdb_collection":   reasonCloudOnly,
	"appwrite_vectorsdb_index":        reasonCloudOnly,
	"appwrite_vectorsdb_document":     reasonCloudOnly,
}

// CheckDestroy returns a destroy check covering every resource left in state.
//
// Types with no registered check are reported rather than passed over. A test
// that silently checks nothing is worse than no test, because it reads as
// coverage.
func CheckDestroy(t *testing.T) func(*terraform.State) error {
	t.Helper()
	return func(state *terraform.State) error {
		if !acceptanceEnabled() {
			return nil
		}

		var problems []string
		var unchecked []string

		for _, ms := range state.Modules {
			for name, rs := range ms.Resources {
				if rs.Primary == nil || rs.Primary.ID == "" {
					continue
				}

				// Only this provider's resources are ours to verify. The echo
				// provider the ephemeral tests use to surface a value into
				// state also leaves a resource behind, and it has no server
				// side to ask about.
				if !strings.HasPrefix(rs.Type, "appwrite_") {
					continue
				}

				fetch, ok := destroyChecks[rs.Type]
				if !ok {
					if _, expected := destroyCheckGaps[rs.Type]; !expected {
						unchecked = append(unchecked, fmt.Sprintf("%s (%s)", name, rs.Type))
					}
					continue
				}

				err := fetch(t, rs.Primary.Attributes)
				switch {
				case err == nil:
					problems = append(problems, fmt.Sprintf("%s still exists on the server after destroy", name))
				case common.IsNotFoundError(err):
					// Gone, as intended.
				default:
					// Neither present nor provably absent. Reported as a
					// failure: an inconclusive destroy check has to be loud,
					// or a broken credential would read as every resource
					// being cleaned up correctly.
					problems = append(problems, fmt.Sprintf("%s could not be verified as destroyed: %s", name, common.FormatError(err)))
				}
			}
		}

		sort.Strings(problems)
		sort.Strings(unchecked)

		if len(unchecked) > 0 {
			problems = append(problems, fmt.Sprintf(
				"no destroy check is registered for %v; add one to internal/acceptance/destroy.go, or record why not in destroyCheckGaps",
				unchecked,
			))
		}
		if len(problems) > 0 {
			return fmt.Errorf("destroy verification failed:\n  - %s", joinLines(problems))
		}
		return nil
	}
}

func joinLines(lines []string) string {
	out := ""
	for i, l := range lines {
		if i > 0 {
			out += "\n  - "
		}
		out += l
	}
	return out
}

// projectScoped builds a client for the project the resource instance belongs
// to. project_id is Optional+Computed on every project-scoped resource, so it
// is always in state; the environment is the fallback for the few that predate
// that attribute.
func projectScoped(t *testing.T, attrs map[string]string) sdkclient.Client {
	t.Helper()
	if projectID := attrs["project_id"]; projectID != "" {
		return ClientForProjectID(t, projectID)
	}
	return ProjectClient(t)
}
