package project

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type capturedRequest struct {
	method  string
	path    string
	query   string
	project string
	body    map[string]any
}

// settingsServer answers every request with the given JSON and keeps the
// requests it received.
func settingsServer(t *testing.T, response any) (*project.Project, *[]capturedRequest) {
	t.Helper()

	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		requests = append(requests, capturedRequest{method: r.Method, path: r.URL.Path, query: r.URL.RawQuery, body: body})

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)

	client := appwrite.NewClient(appwrite.WithEndpoint(server.URL+"/v1"), appwrite.WithProject("project-id"))
	return appwrite.NewProject(client), &requests
}

func onlyRequest(t *testing.T, requests *[]capturedRequest, method, path string) capturedRequest {
	t.Helper()
	if len(*requests) != 1 {
		t.Fatalf("got %d requests, want 1", len(*requests))
	}
	got := (*requests)[0]
	if got.method != method || got.path != path {
		t.Fatalf("got %s %s, want %s %s", got.method, got.path, method, path)
	}
	return got
}

func assertBody(t *testing.T, got, want map[string]any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("body:\n got %v\nwant %v", got, want)
	}
}

// Every field is sent on every update, so a value removed from the
// configuration is cleared rather than left as it was. The model's null
// optional attributes have to arrive as empty strings to do that.
func TestUpdateSMTPRequest(t *testing.T) {
	t.Parallel()

	model := smtpResourceModel{
		Enabled:      types.BoolValue(true),
		Host:         types.StringValue("smtp.example.com"),
		Port:         types.Int64Value(587),
		Username:     types.StringNull(),
		SenderEmail:  types.StringValue("noreply@example.com"),
		SenderName:   types.StringNull(),
		ReplyToEmail: types.StringValue("support@example.com"),
		ReplyToName:  types.StringNull(),
		Secure:       types.StringNull(),
	}

	client, requests := settingsServer(t, map[string]any{"$id": "project-id"})
	if _, err := updateSMTP(client, model.settings("")); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := onlyRequest(t, requests, http.MethodPatch, "/v1/project/smtp")
	// No secure: the API has no value meaning "none" and rejects "".
	assertBody(t, got.body, map[string]any{
		"enabled":      true,
		"host":         "smtp.example.com",
		"port":         float64(587),
		"username":     "",
		"password":     "",
		"senderEmail":  "noreply@example.com",
		"senderName":   "",
		"replyToEmail": "support@example.com",
		"replyToName":  "",
	})

	model.Enabled = types.BoolValue(false)
	model.Username = types.StringValue("user")
	model.SenderName = types.StringValue("Sender")
	model.ReplyToName = types.StringValue("Support")
	model.Secure = types.StringValue("tls")

	client, requests = settingsServer(t, map[string]any{"$id": "project-id"})
	if _, err := updateSMTP(client, model.settings("secret")); err != nil {
		t.Fatalf("update: %v", err)
	}
	got = onlyRequest(t, requests, http.MethodPatch, "/v1/project/smtp")
	assertBody(t, got.body, map[string]any{
		"enabled":      false,
		"host":         "smtp.example.com",
		"port":         float64(587),
		"username":     "user",
		"password":     "secret",
		"senderEmail":  "noreply@example.com",
		"senderName":   "Sender",
		"replyToEmail": "support@example.com",
		"replyToName":  "Support",
		"secure":       "tls",
	})
}

// Destroy disables custom SMTP explicitly, which also stops Appwrite from
// connecting to a server that may be gone, and clears what can be cleared.
// Host, port and secure are left out: the API rejects an empty host and has no
// empty value for secure.
func TestResetSMTPRequest(t *testing.T) {
	t.Parallel()

	client, requests := settingsServer(t, map[string]any{"$id": "project-id"})
	if _, err := resetSMTP(client); err != nil {
		t.Fatalf("reset: %v", err)
	}
	got := onlyRequest(t, requests, http.MethodPatch, "/v1/project/smtp")
	assertBody(t, got.body, map[string]any{
		"enabled":      false,
		"username":     "",
		"password":     "",
		"senderEmail":  "",
		"senderName":   "",
		"replyToEmail": "",
		"replyToName":  "",
	})
}

func TestUpdateEmailTemplateRequest(t *testing.T) {
	t.Parallel()

	model := emailTemplateResourceModel{
		Type:         types.StringValue("recovery"),
		Locale:       types.StringValue("pt-br"),
		Subject:      types.StringValue("Subject"),
		Message:      types.StringValue("<p>{{user}}</p>"),
		SenderName:   types.StringValue("Sender"),
		SenderEmail:  types.StringNull(),
		ReplyToEmail: types.StringNull(),
		ReplyToName:  types.StringNull(),
	}

	client, requests := settingsServer(t, map[string]any{"templateId": "recovery", "locale": "pt-br"})
	if _, err := updateEmailTemplate(client, model.settings()); err != nil {
		t.Fatalf("update: %v", err)
	}
	got := onlyRequest(t, requests, http.MethodPatch, "/v1/project/templates/email")
	assertBody(t, got.body, map[string]any{
		"templateId":   "recovery",
		"locale":       "pt-br",
		"subject":      "Subject",
		"message":      "<p>{{user}}</p>",
		"senderName":   "Sender",
		"senderEmail":  "",
		"replyToEmail": "",
		"replyToName":  "",
	})
}

// Reading a single template returns Appwrite's default text when there is no
// custom one, so the resource reads the list, which holds custom templates
// only. A template missing from it has been reset.
func TestFindCustomEmailTemplate(t *testing.T) {
	t.Parallel()

	list := map[string]any{
		"total": 2,
		"templates": []map[string]any{
			{"templateId": "verification", "locale": "de", "subject": "de subject", "message": "de message"},
			{"templateId": "verification", "locale": "en", "subject": "en subject", "message": "en message"},
		},
	}

	client, requests := settingsServer(t, list)
	found, err := findCustomEmailTemplate(client, "verification", "en")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	got := onlyRequest(t, requests, http.MethodGet, "/v1/project/templates/email")
	// A limit would page the list, and the resource would miss templates
	// past the first page.
	if got.query != "" {
		t.Errorf("list sent query %q, want none", got.query)
	}
	if found == nil || found.Subject != "en subject" {
		t.Fatalf("found %+v, want the en template", found)
	}

	client, _ = settingsServer(t, list)
	found, err = findCustomEmailTemplate(client, "recovery", "en")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if found != nil {
		t.Fatalf("found %+v for a template that is not customized", found)
	}
}

func TestParseEmailTemplateImportID(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		id, projectID, templateType, locale string
		wantErr                             bool
	}{
		{id: "verification/en", templateType: "verification", locale: "en"},
		{id: "project-id/recovery/pt-br", projectID: "project-id", templateType: "recovery", locale: "pt-br"},
		{id: "verification", wantErr: true},
		{id: "verification/", wantErr: true},
		{id: "/verification/en", wantErr: true},
		{id: "a/b/c/d", wantErr: true},
		{id: "", wantErr: true},
	} {
		projectID, templateType, locale, err := parseEmailTemplateImportID(tc.id)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: no error", tc.id)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tc.id, err)
			continue
		}
		if projectID != tc.projectID || templateType != tc.templateType || locale != tc.locale {
			t.Errorf("%q: got %q %q %q", tc.id, projectID, templateType, locale)
		}
	}
}

func TestCheckSecureRemoval(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		stored  string
		planned types.String
		wantErr bool
	}{
		{stored: "", planned: types.StringNull()},
		{stored: "", planned: types.StringValue("tls")},
		{stored: "tls", planned: types.StringValue("tls")},
		{stored: "tls", planned: types.StringValue("ssl")},
		{stored: "tls", planned: types.StringNull(), wantErr: true},
		{stored: "ssl", planned: types.StringNull(), wantErr: true},
	} {
		err := checkSecureRemoval(tc.stored, tc.planned)
		if (err != nil) != tc.wantErr {
			t.Errorf("stored %q, planned %v: err = %v, want error %v", tc.stored, tc.planned, err, tc.wantErr)
		}
	}
}

// The API returns empty strings for unset fields and never returns the
// password. Neither may show up as a diff.
func TestMapSMTPToState(t *testing.T) {
	t.Parallel()

	model := smtpResourceModel{
		Password:     types.StringValue("kept"),
		Username:     types.StringNull(),
		SenderName:   types.StringValue("Old"),
		ReplyToEmail: types.StringNull(),
		ReplyToName:  types.StringValue(""),
		Secure:       types.StringNull(),
	}
	mapSMTPToState(&models.Project{
		Id:              "project-id",
		SmtpEnabled:     true,
		SmtpHost:        "smtp.example.com",
		SmtpPort:        465,
		SmtpSenderEmail: "noreply@example.com",
		SmtpSenderName:  "",
		SmtpSecure:      "ssl",
	}, &model)

	want := smtpResourceModel{
		ID:           types.StringValue("project-id"),
		Enabled:      types.BoolValue(true),
		Host:         types.StringValue("smtp.example.com"),
		Port:         types.Int64Value(465),
		Username:     types.StringNull(),
		Password:     types.StringValue("kept"),
		SenderEmail:  types.StringValue("noreply@example.com"),
		SenderName:   types.StringValue(""),
		ReplyToEmail: types.StringNull(),
		ReplyToName:  types.StringValue(""),
		Secure:       types.StringValue("ssl"),
	}
	if !reflect.DeepEqual(model, want) {
		t.Errorf("got  %+v\nwant %+v", model, want)
	}
}

func TestSMTPConfigured(t *testing.T) {
	t.Parallel()

	if smtpConfigured(&models.Project{}) {
		t.Error("a project with no SMTP host reports a configuration")
	}
	if !smtpConfigured(&models.Project{SmtpHost: "smtp.example.com"}) {
		t.Error("a disabled configuration with a host is not reported")
	}
	if !smtpConfigured(&models.Project{SmtpEnabled: true}) {
		t.Error("an enabled configuration is not reported")
	}
}

// Type and locale are checked at plan time against the lists Appwrite accepts,
// so a typo fails the plan instead of the apply.
func TestEmailTemplateValidation(t *testing.T) {
	t.Parallel()

	attributes := resourceSchema(t, NewEmailTemplateResource()).Attributes
	for _, tc := range []struct {
		attribute, value string
		valid            bool
	}{
		{"type", "verification", true},
		{"type", "otpSession", true},
		{"type", "Verification", false},
		{"type", "welcome", false},
		{"locale", "en", true},
		{"locale", "pt-br", true},
		{"locale", "zh-tw", true},
		{"locale", "pt-BR", false},
		{"locale", "pt_br", false},
		{"locale", "xx", false},
	} {
		if got := stringValid(t, attributes[tc.attribute], tc.attribute, tc.value); got != tc.valid {
			t.Errorf("%s = %q: valid %v, want %v", tc.attribute, tc.value, got, tc.valid)
		}
	}
}

func TestSMTPValidation(t *testing.T) {
	t.Parallel()

	attributes := resourceSchema(t, NewSMTPResource()).Attributes
	for _, tc := range []struct {
		attribute, value string
		valid            bool
	}{
		{"host", "smtp.example.com", true},
		{"host", "10.0.0.1", true},
		{"host", "smtp.example.com:587", false},
		{"host", "smtp://smtp.example.com", false},
		{"host", "", false},
		{"secure", "tls", true},
		{"secure", "ssl", true},
		{"secure", "", false},
		{"secure", "starttls", false},
	} {
		if got := stringValid(t, attributes[tc.attribute], tc.attribute, tc.value); got != tc.valid {
			t.Errorf("%s = %q: valid %v, want %v", tc.attribute, tc.value, got, tc.valid)
		}
	}
}

func resourceSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func stringValid(t *testing.T, attribute schema.Attribute, name, value string) bool {
	t.Helper()
	stringAttribute, ok := attribute.(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a string attribute", name)
	}
	for _, v := range stringAttribute.Validators {
		var resp validator.StringResponse
		v.ValidateString(context.Background(), validator.StringRequest{
			Path:        path.Root(name),
			ConfigValue: types.StringValue(value),
		}, &resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}
