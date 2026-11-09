package sweep

import (
	"encoding/json"
	"errors"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
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
	resource.AddTestSweepers("appwrite_project_variable", &resource.Sweeper{
		Name: "appwrite_project_variable",
		F:    sweepProjectVariables,
	})
	resource.AddTestSweepers("appwrite_proxy_rule", &resource.Sweeper{
		Name: "appwrite_proxy_rule",
		F:    sweepProxyRules,
	})
	resource.AddTestSweepers("appwrite_project_platform", &resource.Sweeper{
		Name: "appwrite_project_platform",
		F:    sweepPlatforms,
	})
	// WAF rules are Cloud-only, but a leftover one keeps filtering the test
	// project's traffic and counts against the plan's limit on rules.
	resource.AddTestSweepers("appwrite_waf_rule", &resource.Sweeper{
		Name: "appwrite_waf_rule",
		F:    sweepWAFRules,
	})
	resource.AddTestSweepers("appwrite_backup_policy", &resource.Sweeper{
		Name: "appwrite_backup_policy",
		F:    sweepBackupPolicies,
	})
	resource.AddTestSweepers("appwrite_project_oauth_provider", &resource.Sweeper{
		Name: "appwrite_project_oauth_provider",
		F:    sweepOAuthProviders,
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

// Project variables have server-assigned IDs and keys that cannot contain
// Prefix, so they are matched on a TF_ACC_TEST_ key prefix instead.
func sweepProjectVariables(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewProject(cfg.Client())
	list, err := svc.ListVariables()
	if err != nil {
		return err
	}
	var e errs
	for _, v := range list.Variables {
		if !Sweepable(v.Id) && !SweepableKey(v.Key) {
			continue
		}
		if _, err := svc.DeleteVariable(v.Id); err != nil {
			e.fail("project variable", v.Id, err)
			continue
		}
		e.ok()
	}
	return e.result("project variables")
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

func sweepPlatforms(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewProject(cfg.Client())
	list, err := svc.ListPlatforms()
	if err != nil {
		return err
	}
	var e errs
	// The list holds a different model per platform type, decoded only as far
	// as a map. Every type has an ID and a name.
	for _, item := range list.Platforms {
		p, _ := item.(map[string]any)
		platformID, _ := p["$id"].(string)
		name, _ := p["name"].(string)
		if platformID == "" || !Sweepable(platformID, name) {
			continue
		}
		if _, err := svc.DeletePlatform(platformID); err != nil {
			e.fail("platform", platformID, err)
			continue
		}
		e.ok()
	}
	return e.result("platforms")
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

func sweepWAFRules(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	svc := appwrite.NewWaf(cfg.Client())
	list, err := svc.ListRules()
	if err != nil {
		// Self-hosted Appwrite has no WAF routes and answers 404, which
		// means there is nothing to sweep rather than that the sweep failed.
		var appErr *client.AppwriteError
		if errors.As(err, &appErr) && appErr.GetStatusCode() == 404 {
			return nil
		}
		return err
	}
	// The list holds a different model per rule type, so it comes back
	// untyped; only the ID and name matter here.
	var rules struct {
		Rules []struct {
			ID   string `json:"$id"`
			Name string `json:"name"`
		} `json:"rules"`
	}
	if err := list.Decode(&rules); err != nil {
		return err
	}
	var e errs
	for _, r := range rules.Rules {
		if !Sweepable(r.ID, r.Name) {
			continue
		}
		if _, err := svc.DeleteRule(r.ID); err != nil {
			e.fail("WAF rule", r.ID, err)
			continue
		}
		e.ok()
	}
	return e.result("WAF rules")
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

// sweepOAuthProviders resets OAuth2 providers a test configured. A provider
// cannot be deleted, so sweeping one means disabling it and clearing what it
// holds, as destroying the resource does. A provider is the test's when one of
// its settings carries the prefix; the tests put it in every client ID.
func sweepOAuthProviders(string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	c := cfg.Client()
	headers := map[string]interface{}{"X-Appwrite-Project": c.Config["project"], "content-type": "application/json"}

	// Read raw: the SDK's GetOAuth2Provider cannot decode every provider.
	resp, err := c.Call("GET", "/project/oauth2", headers, map[string]interface{}{})
	if err != nil {
		return err
	}
	raw, err := client.ResponseBody(resp)
	if err != nil {
		return err
	}
	var list struct {
		Providers []map[string]interface{} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &list); err != nil {
		return err
	}

	var e errs
	for _, provider := range list.Providers {
		id, _ := provider["$id"].(string)
		var identifiers []string
		reset := map[string]interface{}{}
		// The response uses the same keys as the update route, so the reset
		// is built from it: strings emptied, flags off, ID lists emptied.
		// prompt falls back to a default rather than to empty, and maxAge
		// cannot be cleared, so both are left alone.
		for key, value := range provider {
			switch v := value.(type) {
			case string:
				if key != "$id" {
					identifiers = append(identifiers, v)
					reset[key] = ""
				}
			case bool:
				reset[key] = false
			case []interface{}:
				if key != "prompt" {
					for _, item := range v {
						if s, ok := item.(string); ok {
							identifiers = append(identifiers, s)
						}
					}
					reset[key] = []string{}
				}
			}
		}
		if !Sweepable(identifiers...) {
			continue
		}
		if _, err := c.Call("PATCH", "/project/oauth2/"+client.EncodePath(id), headers, reset); err != nil {
			e.fail("OAuth2 provider", id, err)
			continue
		}
		e.ok()
	}
	return e.result("OAuth2 providers")
}
