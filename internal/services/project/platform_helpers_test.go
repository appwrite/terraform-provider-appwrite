package project

import (
	"context"
	"strings"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidatePlatformConfig(t *testing.T) {
	t.Parallel()

	s := types.StringValue
	null := types.StringNull()
	unknown := types.StringUnknown()

	for name, tc := range map[string]struct {
		config platformResourceModel
		// errors lists the attributes that must be reported, in any order.
		errors []string
	}{
		"web":                     {config: platformResourceModel{Type: s("web"), Hostname: s("app.example.com")}},
		"android":                 {config: platformResourceModel{Type: s("android"), ApplicationID: s("com.example.app")}},
		"apple":                   {config: platformResourceModel{Type: s("apple"), BundleIdentifier: s("com.example.app")}},
		"linux":                   {config: platformResourceModel{Type: s("linux"), PackageName: s("example-app")}},
		"windows":                 {config: platformResourceModel{Type: s("windows"), PackageIdentifierName: s("Example.App")}},
		"web without hostname":    {config: platformResourceModel{Type: s("web")}, errors: []string{"hostname"}},
		"apple without bundle id": {config: platformResourceModel{Type: s("apple"), Hostname: null}, errors: []string{"bundle_identifier"}},
		"android given hostname too": {
			config: platformResourceModel{Type: s("android"), ApplicationID: s("com.example.app"), Hostname: s("app.example.com")},
			errors: []string{"hostname"},
		},
		"wrong type's argument": {
			config: platformResourceModel{Type: s("linux"), PackageIdentifierName: s("Example.App")},
			errors: []string{"package_name", "package_identifier_name"},
		},
		// Known only at apply: neither missing nor unsupported yet.
		"unknown identifier":       {config: platformResourceModel{Type: s("windows"), PackageIdentifierName: unknown}},
		"unknown other identifier": {config: platformResourceModel{Type: s("web"), Hostname: s("a.example.com"), ApplicationID: unknown}},
		// Reported by the type's own validator.
		"unsupported type": {config: platformResourceModel{Type: s("ios"), Hostname: s("app.example.com")}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diags := validatePlatformConfig(tc.config)
			if got, want := diags.ErrorsCount(), len(tc.errors); got != want {
				t.Fatalf("got %d errors, want %d: %v", got, want, diags)
			}
			for _, attribute := range tc.errors {
				found := false
				for _, d := range diags.Errors() {
					if strings.Contains(d.Detail(), attribute) {
						found = true
					}
				}
				if !found {
					t.Errorf("no error mentions %s: %v", attribute, diags)
				}
			}
		})
	}
}

func TestTextValidator(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		value   types.String
		max     int
		wantErr bool
	}{
		"plain":                    {value: types.StringValue("My App"), max: 128},
		"padded is kept":           {value: types.StringValue("  My App  "), max: 128},
		"empty":                    {value: types.StringValue(""), max: 128, wantErr: true},
		"spaces only":              {value: types.StringValue("   "), max: 128, wantErr: true},
		"tab and newline":          {value: types.StringValue("\t\n"), max: 128, wantErr: true},
		"no-break space":           {value: types.StringValue("\u00a0"), max: 128, wantErr: true},
		"zero-width space":         {value: types.StringValue("\u200b"), max: 128, wantErr: true},
		"at the limit":             {value: types.StringValue(strings.Repeat("a", 128)), max: 128},
		"over the limit":           {value: types.StringValue(strings.Repeat("a", 129)), max: 128, wantErr: true},
		"multibyte at the limit":   {value: types.StringValue(strings.Repeat("é", 128)), max: 128},
		"multibyte over the limit": {value: types.StringValue(strings.Repeat("é", 129)), max: 128, wantErr: true},
		"null":                     {value: types.StringNull(), max: 128},
		"unknown":                  {value: types.StringUnknown(), max: 128},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			textValidator{maxLength: tc.max}.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tc.value,
			}, resp)
			if got := resp.Diagnostics.HasError(); got != tc.wantErr {
				t.Errorf("got error %v, want %v: %v", got, tc.wantErr, resp.Diagnostics)
			}
		})
	}
}

func TestPatterns(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]bool{
		"web":                  true,
		"6aeb1c279e4fabbe793e": true,
		"tf-acc-test.web_1":    true,
		"-web":                 false,
		"_web":                 false,
		".web":                 false,
		"unique()":             false,
		"web platform":         false,
	} {
		if got := platformIDPattern.MatchString(value); got != want {
			t.Errorf("id %q: got %v, want %v", value, got, want)
		}
	}

	for value, want := range map[string]bool{
		"app.example.com":     true,
		"*.example.com":       true,
		"localhost":           true,
		"https://example.com": false,
		"example.com/app":     false,
		"example.com:3000":    false,
		"[::1]":               false,
	} {
		if got := hostnamePattern.MatchString(value); got != want {
			t.Errorf("hostname %q: got %v, want %v", value, got, want)
		}
	}
}

func TestMapPlatformToState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		platform  models.Model
		attribute string
		get       func(platformResourceModel) types.String
	}{
		{&models.PlatformWeb{Id: "p", Name: "n", Type: "web", Hostname: "*.Example.COM"}, "hostname",
			func(m platformResourceModel) types.String { return m.Hostname }},
		{&models.PlatformAndroid{Id: "p", Name: "n", Type: "android", ApplicationId: "com.example.app"}, "application_id",
			func(m platformResourceModel) types.String { return m.ApplicationID }},
		{&models.PlatformApple{Id: "p", Name: "n", Type: "apple", BundleIdentifier: "com.example.app"}, "bundle_identifier",
			func(m platformResourceModel) types.String { return m.BundleIdentifier }},
		{&models.PlatformLinux{Id: "p", Name: "n", Type: "linux", PackageName: "example-app"}, "package_name",
			func(m platformResourceModel) types.String { return m.PackageName }},
		{&models.PlatformWindows{Id: "p", Name: "n", Type: "windows", PackageIdentifierName: "Example.App"}, "package_identifier_name",
			func(m platformResourceModel) types.String { return m.PackageIdentifierName }},
	} {
		t.Run(tc.attribute, func(t *testing.T) {
			t.Parallel()

			// Start from a model holding every identifier, as a state from
			// before a replacement could: only the platform's own survives.
			model := platformResourceModel{
				Hostname:              types.StringValue("stale"),
				ApplicationID:         types.StringValue("stale"),
				BundleIdentifier:      types.StringValue("stale"),
				PackageName:           types.StringValue("stale"),
				PackageIdentifierName: types.StringValue("stale"),
			}
			if err := mapPlatformToState(tc.platform, &model); err != nil {
				t.Fatal(err)
			}
			fields, _ := flattenPlatform(tc.platform)
			if got := tc.get(model); got.ValueString() != fields.identifier {
				t.Errorf("%s: got %s, want %q", tc.attribute, got, fields.identifier)
			}
			if model.Type.ValueString() != fields.platformType {
				t.Errorf("type: got %s, want %s", model.Type, fields.platformType)
			}
			for _, kind := range platformKinds {
				if kind.attribute == tc.attribute {
					continue
				}
				if value := *model.identifierField(kind.name); !value.IsNull() {
					t.Errorf("%s: got %s, want null", kind.attribute, value)
				}
			}
		})
	}

	if err := mapPlatformToState(&models.PlatformList{}, &platformResourceModel{}); err == nil {
		t.Error("expected an error for a model that is not a platform")
	}
}

func TestPlatformImportID(t *testing.T) {
	t.Parallel()

	for id, want := range map[string][2]string{
		"platform":         {"", "platform"},
		"project/platform": {"project", "platform"},
	} {
		projectID, platformID, err := platformImportID(id)
		if err != nil {
			t.Errorf("%q: %v", id, err)
			continue
		}
		if projectID != want[0] || platformID != want[1] {
			t.Errorf("%q: got %q, %q", id, projectID, platformID)
		}
	}
	for _, id := range []string{"", "/platform", "project/", "a/b/c"} {
		if _, _, err := platformImportID(id); err == nil {
			t.Errorf("%q: expected an error", id)
		}
	}
}
