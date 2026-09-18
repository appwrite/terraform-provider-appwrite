package sweep

import (
	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Register wires every sweeper into the testing framework. Called from TestMain
// so that `go test -sweep` has them available.
//
// Dependencies matter: a table cannot be deleted after its database is gone, so
// the framework has to run the inner sweeper first. Naming them here rather
// than spreading resource.AddTestSweepers across service packages keeps that
// ordering visible in one place.
func Register() {
	resource.AddTestSweepers("appwrite_webhook", &resource.Sweeper{
		Name: "appwrite_webhook",
		F:    sweepWebhooks,
	})
	resource.AddTestSweepers("appwrite_auth_user", &resource.Sweeper{
		Name: "appwrite_auth_user",
		F:    sweepUsers,
	})
	resource.AddTestSweepers("appwrite_auth_team", &resource.Sweeper{
		Name: "appwrite_auth_team",
		F:    sweepTeams,
	})
	resource.AddTestSweepers("appwrite_messaging_topic", &resource.Sweeper{
		Name: "appwrite_messaging_topic",
		F:    sweepTopics,
	})
	resource.AddTestSweepers("appwrite_messaging_provider", &resource.Sweeper{
		Name: "appwrite_messaging_provider",
		F:    sweepProviders,
	})
	resource.AddTestSweepers("appwrite_storage_bucket", &resource.Sweeper{
		Name: "appwrite_storage_bucket",
		F:    sweepBuckets,
	})
	resource.AddTestSweepers("appwrite_function", &resource.Sweeper{
		Name: "appwrite_function",
		F:    sweepFunctions,
	})
	resource.AddTestSweepers("appwrite_site", &resource.Sweeper{
		Name: "appwrite_site",
		F:    sweepSites,
	})
	resource.AddTestSweepers("appwrite_proxy_rule", &resource.Sweeper{
		Name: "appwrite_proxy_rule",
		F:    sweepProxyRules,
	})
	resource.AddTestSweepers("appwrite_backup_policy", &resource.Sweeper{
		Name: "appwrite_backup_policy",
		F:    sweepBackupPolicies,
	})
	// Tables, columns, indexes and rows live inside a database, and deleting the
	// database takes them with it. One sweeper at the database level is both
	// sufficient and considerably faster than four.
	resource.AddTestSweepers("appwrite_tablesdb", &resource.Sweeper{
		Name: "appwrite_tablesdb",
		F:    sweepDatabases,
	})
}

// The region argument is the framework's, and Appwrite has no equivalent: a key
// is already scoped to one endpoint and project. Ignored throughout.

func sweepWebhooks(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewWebhooks(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, w := range list.Webhooks {
		if !Sweepable(w.Id, w.Name) {
			continue
		}
		if _, err := svc.Delete(w.Id); err != nil {
			e.fail("webhook", w.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("webhooks")
}

func sweepUsers(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewUsers(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, u := range list.Users {
		if !Sweepable(u.Id, u.Name, u.Email) {
			continue
		}
		if _, err := svc.Delete(u.Id); err != nil {
			e.fail("user", u.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("users")
}

func sweepTeams(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewTeams(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, tm := range list.Teams {
		if !Sweepable(tm.Id, tm.Name) {
			continue
		}
		if _, err := svc.Delete(tm.Id); err != nil {
			e.fail("team", tm.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("teams")
}

func sweepTopics(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewMessaging(cfg.Client())
	list, err := svc.ListTopics()
	if err != nil {
		return err
	}
	var e errs
	for _, tp := range list.Topics {
		if !Sweepable(tp.Id, tp.Name) {
			continue
		}
		if _, err := svc.DeleteTopic(tp.Id); err != nil {
			e.fail("topic", tp.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("messaging topics")
}

func sweepProviders(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewMessaging(cfg.Client())
	list, err := svc.ListProviders()
	if err != nil {
		return err
	}
	var e errs
	for _, p := range list.Providers {
		if !Sweepable(p.Id, p.Name) {
			continue
		}
		if _, err := svc.DeleteProvider(p.Id); err != nil {
			e.fail("provider", p.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("messaging providers")
}

func sweepBuckets(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewStorage(cfg.Client())
	list, err := svc.ListBuckets()
	if err != nil {
		return err
	}
	var e errs
	for _, b := range list.Buckets {
		if !Sweepable(b.Id, b.Name) {
			continue
		}
		// Files go with the bucket, so they need no sweeper of their own.
		if _, err := svc.DeleteBucket(b.Id); err != nil {
			e.fail("bucket", b.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("storage buckets")
}

func sweepFunctions(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewFunctions(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, f := range list.Functions {
		if !Sweepable(f.Id, f.Name) {
			continue
		}
		// Deployments and variables belong to the function and go with it.
		if _, err := svc.Delete(f.Id); err != nil {
			e.fail("function", f.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("functions")
}

func sweepSites(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewSites(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, s := range list.Sites {
		if !Sweepable(s.Id, s.Name) {
			continue
		}
		if _, err := svc.Delete(s.Id); err != nil {
			e.fail("site", s.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("sites")
}

func sweepProxyRules(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewProxy(cfg.Client())
	list, err := svc.ListRules()
	if err != nil {
		return err
	}
	var e errs
	for _, r := range list.Rules {
		// A proxy rule has no name; its domain is the only thing a test controls.
		if !Sweepable(r.Id, r.Domain) {
			continue
		}
		if _, err := svc.DeleteRule(r.Id); err != nil {
			e.fail("proxy rule", r.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("proxy rules")
}

func sweepBackupPolicies(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewBackups(cfg.Client())
	list, err := svc.ListPolicies()
	if err != nil {
		return err
	}
	var e errs
	for _, p := range list.Policies {
		if !Sweepable(p.Id, p.Name) {
			continue
		}
		if _, err := svc.DeletePolicy(p.Id); err != nil {
			e.fail("backup policy", p.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("backup policies")
}

func sweepDatabases(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewTablesDB(cfg.Client())
	list, err := svc.List()
	if err != nil {
		return err
	}
	var e errs
	for _, d := range list.Databases {
		if !Sweepable(d.Id, d.Name) {
			continue
		}
		if _, err := svc.Delete(d.Id); err != nil {
			e.fail("database", d.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("databases")
}
