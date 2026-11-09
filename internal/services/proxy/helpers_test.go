package proxy

import (
	"sort"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateRuleConfig(t *testing.T) {
	t.Parallel()

	s := types.StringValue
	null := types.StringNull()
	unknown := types.StringUnknown()

	// Every argument null; each case sets the ones it needs.
	base := func(ruleType string) ruleResourceModel {
		return ruleResourceModel{
			Type:                 s(ruleType),
			Domain:               s("rule.example.com"),
			ResourceID:           null,
			Branch:               null,
			RedirectURL:          null,
			RedirectStatusCode:   types.Int64Null(),
			RedirectResourceType: null,
		}
	}
	redirect := func() ruleResourceModel {
		c := base("redirect")
		c.ResourceID = s("site-id")
		c.RedirectURL = s("https://example.org")
		c.RedirectStatusCode = types.Int64Value(301)
		c.RedirectResourceType = s("site")
		return c
	}

	for _, tc := range []struct {
		name   string
		config ruleResourceModel
		// errors lists the attributes that must be reported, in any order.
		errors []string
	}{
		{name: "site", config: func() ruleResourceModel { c := base("site"); c.ResourceID = s("site-id"); return c }()},
		{name: "site with branch", config: func() ruleResourceModel {
			c := base("site")
			c.ResourceID = s("site-id")
			c.Branch = s("main")
			return c
		}()},
		{name: "function", config: func() ruleResourceModel { c := base("function"); c.ResourceID = s("fn-id"); return c }()},
		{name: "site without resource_id", config: base("site"), errors: []string{"resource_id"}},
		{name: "function without resource_id", config: base("function"), errors: []string{"resource_id"}},
		{name: "site with redirect arguments", config: func() ruleResourceModel {
			c := redirect()
			c.Type = s("site")
			return c
		}(), errors: []string{"redirect_url", "redirect_status_code", "redirect_resource_type"}},

		{name: "api", config: base("api")},
		{name: "api with resource_id", config: func() ruleResourceModel { c := base("api"); c.ResourceID = s("site-id"); return c }(), errors: []string{"resource_id"}},
		{name: "api with branch", config: func() ruleResourceModel { c := base("api"); c.Branch = s("main"); return c }(), errors: []string{"branch"}},
		{name: "api with redirect_url", config: func() ruleResourceModel {
			c := base("api")
			c.RedirectURL = s("https://example.org")
			return c
		}(), errors: []string{"redirect_url"}},

		{name: "redirect", config: redirect()},
		{name: "redirect without arguments", config: base("redirect"),
			errors: []string{"resource_id", "redirect_url", "redirect_status_code", "redirect_resource_type"}},
		{name: "redirect with branch", config: func() ruleResourceModel { c := redirect(); c.Branch = s("main"); return c }(), errors: []string{"branch"}},
		{name: "redirect to a relative URL", config: func() ruleResourceModel {
			c := redirect()
			c.RedirectURL = s("example.org/landing")
			return c
		}(), errors: []string{"redirect_url"}},

		// Values from other resources are judged at apply, not plan.
		{name: "redirect with unknown arguments", config: func() ruleResourceModel {
			c := base("redirect")
			c.ResourceID = unknown
			c.RedirectURL = unknown
			c.RedirectStatusCode = types.Int64Unknown()
			c.RedirectResourceType = unknown
			return c
		}()},
		{name: "api with unknown resource_id", config: func() ruleResourceModel { c := base("api"); c.ResourceID = unknown; return c }()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			diags := validateRuleConfig(tc.config)

			var got []string
			for _, d := range diags {
				withPath, ok := d.(interface{ Path() path.Path })
				if !ok {
					t.Fatalf("diagnostic without a path: %s: %s", d.Summary(), d.Detail())
				}
				got = append(got, withPath.Path().String())
			}
			want := append([]string(nil), tc.errors...)
			sort.Strings(got)
			sort.Strings(want)
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Errorf("errors on %v, want %v (%v)", got, want, diags)
			}
		})
	}
}

func TestMapRuleToState(t *testing.T) {
	t.Parallel()

	ptr := func(v string) *string { return &v }

	t.Run("site rule maps as before", func(t *testing.T) {
		t.Parallel()
		var model ruleResourceModel
		mapRuleToState(&models.ProxyRule{
			Id:                     "rule-id",
			Domain:                 "www.example.com",
			Type:                   "deployment",
			DeploymentResourceType: ptr("site"),
			DeploymentResourceId:   "site-id",
			// Appwrite reports 301 on rules that are not redirects.
			RedirectStatusCode: 301,
			Status:             "verified",
		}, &model)

		if model.Type.ValueString() != "site" || model.ResourceID.ValueString() != "site-id" {
			t.Errorf("type, resource_id = %v, %v", model.Type, model.ResourceID)
		}
		if !model.RedirectURL.IsNull() || !model.RedirectStatusCode.IsNull() || !model.RedirectResourceType.IsNull() {
			t.Errorf("redirect arguments set on a site rule: %v %v %v", model.RedirectURL, model.RedirectStatusCode, model.RedirectResourceType)
		}
		if !model.Branch.IsNull() {
			t.Errorf("branch = %v, want null", model.Branch)
		}
	})

	t.Run("api rule", func(t *testing.T) {
		t.Parallel()
		var model ruleResourceModel
		mapRuleToState(&models.ProxyRule{
			Id:                     "rule-id",
			Domain:                 "api.example.com",
			Type:                   "api",
			DeploymentResourceType: ptr(""),
			RedirectStatusCode:     301,
		}, &model)

		if model.Type.ValueString() != "api" {
			t.Errorf("type = %v, want api", model.Type)
		}
		if !model.ResourceID.IsNull() {
			t.Errorf("resource_id = %v, want null", model.ResourceID)
		}
		if !model.RedirectStatusCode.IsNull() {
			t.Errorf("redirect_status_code = %v, want null", model.RedirectStatusCode)
		}
	})

	t.Run("redirect rule", func(t *testing.T) {
		t.Parallel()
		var model ruleResourceModel
		mapRuleToState(&models.ProxyRule{
			Id:                     "rule-id",
			Domain:                 "old.example.com",
			Type:                   "redirect",
			RedirectUrl:            "https://Example.org/Landing?x=1",
			RedirectStatusCode:     307,
			DeploymentResourceType: ptr("function"),
			DeploymentResourceId:   "function-id",
		}, &model)

		// The parent resource's type must not leak into type, or every plan
		// would replace the rule with a function rule.
		if model.Type.ValueString() != "redirect" {
			t.Errorf("type = %v, want redirect", model.Type)
		}
		if model.ResourceID.ValueString() != "function-id" || model.RedirectResourceType.ValueString() != "function" {
			t.Errorf("resource_id, redirect_resource_type = %v, %v", model.ResourceID, model.RedirectResourceType)
		}
		if model.RedirectURL.ValueString() != "https://Example.org/Landing?x=1" || model.RedirectStatusCode.ValueInt64() != 307 {
			t.Errorf("redirect_url, redirect_status_code = %v, %v", model.RedirectURL, model.RedirectStatusCode)
		}
	})
}
