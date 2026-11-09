package proxy

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/proxy"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	ruleTypeSite     = "site"
	ruleTypeFunction = "function"
	ruleTypeAPI      = "api"
	ruleTypeRedirect = "redirect"

	// The rule types Appwrite reports, as opposed to the ones configured: site
	// and function rules are both "deployment" rules, told apart by
	// deploymentResourceType.
	apiRuleType      = "api"
	redirectRuleType = "redirect"
)

var (
	ruleTypes             = []string{ruleTypeSite, ruleTypeFunction, ruleTypeAPI, ruleTypeRedirect}
	redirectStatusCodes   = []int64{301, 302, 307, 308}
	redirectResourceTypes = []string{ruleTypeSite, ruleTypeFunction}
)

// createRule sends the create request for the rule's type.
func createRule(client *proxy.Proxy, plan ruleResourceModel) (*models.ProxyRule, error) {
	domain := plan.Domain.ValueString()
	switch plan.Type.ValueString() {
	case ruleTypeSite:
		var opts []proxy.CreateSiteRuleOption
		if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
			opts = append(opts, client.WithCreateSiteRuleBranch(plan.Branch.ValueString()))
		}
		return client.CreateSiteRule(domain, plan.ResourceID.ValueString(), opts...)
	case ruleTypeFunction:
		var opts []proxy.CreateFunctionRuleOption
		if !plan.Branch.IsNull() && !plan.Branch.IsUnknown() {
			opts = append(opts, client.WithCreateFunctionRuleBranch(plan.Branch.ValueString()))
		}
		return client.CreateFunctionRule(domain, plan.ResourceID.ValueString(), opts...)
	case ruleTypeAPI:
		return client.CreateAPIRule(domain)
	case ruleTypeRedirect:
		return client.CreateRedirectRule(
			domain,
			plan.RedirectURL.ValueString(),
			strconv.FormatInt(plan.RedirectStatusCode.ValueInt64(), 10),
			plan.ResourceID.ValueString(),
			plan.RedirectResourceType.ValueString(),
		)
	default:
		return nil, fmt.Errorf("unsupported proxy rule type %q", plan.Type.ValueString())
	}
}

// validateRuleConfig checks the arguments each rule type takes, so a missing
// or misplaced one fails the plan rather than the apply.
func validateRuleConfig(config ruleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	ruleType := config.Type.ValueString()

	deployment := ruleType == ruleTypeSite || ruleType == ruleTypeFunction
	redirect := ruleType == ruleTypeRedirect

	fields := []struct {
		name     string
		value    attr.Value
		required bool
		allowed  bool
	}{
		{"resource_id", config.ResourceID, deployment || redirect, deployment || redirect},
		{"branch", config.Branch, false, deployment},
		{"redirect_url", config.RedirectURL, redirect, redirect},
		{"redirect_status_code", config.RedirectStatusCode, redirect, redirect},
		{"redirect_resource_type", config.RedirectResourceType, redirect, redirect},
	}
	for _, field := range fields {
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if field.value.IsUnknown() {
			continue
		}
		switch {
		case field.required && field.value.IsNull():
			diags.AddAttributeError(path.Root(field.name), "Missing required argument",
				fmt.Sprintf("%s rules require %s.", ruleType, field.name))
		case !field.allowed && !field.value.IsNull():
			diags.AddAttributeError(path.Root(field.name), "Unsupported argument",
				fmt.Sprintf("%s is not used by %s rules. Remove it from the configuration.", field.name, ruleType))
		}
	}

	if redirect && !config.RedirectURL.IsNull() && !config.RedirectURL.IsUnknown() {
		if !isAbsoluteURL(config.RedirectURL.ValueString()) {
			diags.AddAttributeError(path.Root("redirect_url"), "Invalid redirect URL",
				fmt.Sprintf("redirect_url must be an absolute URL with a scheme and host, such as https://example.com, got %q.", config.RedirectURL.ValueString()))
		}
	}
	return diags
}

func isAbsoluteURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

// mapRuleToState copies a rule into the model.
//
// Appwrite reports site and function rules as type "deployment" with the
// resource type in deploymentResourceType, and that mapping is kept exactly as
// it was before API and redirect rules were supported. A redirect rule also
// carries a site or function in deploymentResourceType and
// deploymentResourceId: the resource whose domains it is listed under, not the
// rule's own type.
func mapRuleToState(rule *models.ProxyRule, model *ruleResourceModel) {
	model.ID = types.StringValue(rule.Id)
	model.Domain = types.StringValue(rule.Domain)
	model.Status = types.StringValue(rule.Status)
	model.Logs = types.StringValue(rule.Logs)
	model.RenewAt = types.StringValue(rule.RenewAt)
	model.CreatedAt = types.StringValue(rule.CreatedAt)
	model.UpdatedAt = types.StringValue(rule.UpdatedAt)

	// Appwrite fills redirectStatusCode with 301 on every rule, so it is only
	// meaningful on a redirect rule.
	model.RedirectURL = types.StringNull()
	model.RedirectStatusCode = types.Int64Null()
	model.RedirectResourceType = types.StringNull()

	switch rule.Type {
	case apiRuleType:
		model.Type = types.StringValue(ruleTypeAPI)
		model.ResourceID = types.StringNull()
	case redirectRuleType:
		model.Type = types.StringValue(ruleTypeRedirect)
		model.ResourceID = types.StringValue(rule.DeploymentResourceId)
		model.RedirectURL = types.StringValue(rule.RedirectUrl)
		model.RedirectStatusCode = types.Int64Value(int64(rule.RedirectStatusCode))
		model.RedirectResourceType = types.StringValue(common.Deref(rule.DeploymentResourceType))
	default:
		model.Type = types.StringValue(common.Deref(rule.DeploymentResourceType))
		model.ResourceID = types.StringValue(rule.DeploymentResourceId)
	}

	if rule.DeploymentVcsProviderBranch != "" {
		model.Branch = types.StringValue(rule.DeploymentVcsProviderBranch)
	} else if !model.Branch.IsNull() {
		model.Branch = types.StringValue("")
	}
}
