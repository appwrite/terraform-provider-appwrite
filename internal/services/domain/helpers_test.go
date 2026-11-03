package domain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Each case is a name and value as written, and the form Appwrite stores them
// in. The provider has to agree with the server on every one of them, or a
// configuration that spells a name differently from how the server stores it
// plans a change on every run.
func TestNormalizeMatchesTheServer(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		recordType, name, value string
		wantName, wantValue     string
	}{
		"apex as @":             {"A", "@", "203.0.113.10", "@", "203.0.113.10"},
		"apex as an empty name": {"A", "", "203.0.113.10", "@", "203.0.113.10"},
		"apex as the domain":    {"CAA", "Example.COM.", `0 issue "certainly.com"`, "@", `0 issue "certainly.com"`},
		"relative name":         {"A", "WWW", "203.0.113.10", "www", "203.0.113.10"},
		"absolute name":         {"A", "www.example.com.", "203.0.113.10", "www", "203.0.113.10"},
		"wildcard":              {"CNAME", "*.example.com", "Appwrite.Network.", "*", "appwrite.network"},
		"name outside the zone": {"A", "www.example.org", "203.0.113.10", "www.example.org", "203.0.113.10"},
		"zone written twice":    {"CNAME", "www.example.com.example.com", "target.example.org", "www.example.com.example.com", "target.example.org"},
		"apex written twice":    {"A", "example.com.example.com.", "203.0.113.10", "example.com.example.com", "203.0.113.10"},
		"MX target":             {"MX", "@", "Mail.Example.com.", "@", "mail.example.com"},
		"null MX":               {"MX", "@", ".", "@", "."},
		"TXT kept as written":   {"TXT", "_dmarc", `"v=DMARC1; p=none"`, "_dmarc", `"v=DMARC1; p=none"`},
		// Not in the server's cases, but follows from the same rule: only host
		// name values are rewritten.
		"AAAA kept as written": {"AAAA", "@", "2001:0DB8::1", "@", "2001:0DB8::1"},
		"SRV target":           {"SRV", "_sip._tcp.example.com.", "SIP.example.com.", "_sip._tcp", "sip.example.com"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotName := normalizeRecordName(tc.name, "example.com")
			gotValue := normalizeRecordValue(tc.recordType, tc.value)
			if gotName != tc.wantName {
				t.Errorf("name: got %q, want %q", gotName, tc.wantName)
			}
			if gotValue != tc.wantValue {
				t.Errorf("value: got %q, want %q", gotValue, tc.wantValue)
			}

			// A stored record is already in its stored form.
			if again := normalizeRecordName(gotName, "example.com"); again != gotName {
				t.Errorf("name is not stable: %q normalized again to %q", gotName, again)
			}
			if again := normalizeRecordValue(tc.recordType, gotValue); again != gotValue {
				t.Errorf("value is not stable: %q normalized again to %q", gotValue, again)
			}
		})
	}
}

func TestMapRecordToStateKeepsTheConfiguredSpelling(t *testing.T) {
	t.Parallel()

	model := recordResourceModel{
		Domain:  types.StringValue("example.com"),
		Name:    types.StringValue("WWW.example.com."),
		Value:   types.StringValue("Target.Example.org."),
		Comment: types.StringNull(),
	}
	mapRecordToState(&models.DnsRecord{
		Id: "record", Type: "CNAME", Name: "www", Value: "target.example.org", Ttl: 3600,
	}, &model)

	if got := model.Name.ValueString(); got != "WWW.example.com." {
		t.Errorf("name: got %q, want the configured spelling kept", got)
	}
	if got := model.Value.ValueString(); got != "Target.Example.org." {
		t.Errorf("value: got %q, want the configured spelling kept", got)
	}
	if !model.Comment.IsNull() {
		t.Errorf("comment: got %q, want an unset comment to stay null", model.Comment.ValueString())
	}
	if !model.Priority.IsNull() || !model.Weight.IsNull() || !model.Port.IsNull() {
		t.Error("priority, weight and port must be null for a type that does not use them")
	}
}

func TestMapRecordToStateReportsARealChange(t *testing.T) {
	t.Parallel()

	model := recordResourceModel{
		Domain: types.StringValue("example.com"),
		Name:   types.StringValue("www"),
		Value:  types.StringValue("203.0.113.10"),
	}
	mapRecordToState(&models.DnsRecord{Type: "A", Name: "api", Value: "203.0.113.11"}, &model)

	if got := model.Name.ValueString(); got != "api" {
		t.Errorf("name: got %q, want the server's value when it differs", got)
	}
	if got := model.Value.ValueString(); got != "203.0.113.11" {
		t.Errorf("value: got %q, want the server's value when it differs", got)
	}
}

// A record written before the server normalized names is still stored with
// its full name. Comparing only one side would show it as a change against a
// configuration that writes the relative name.
func TestMapRecordToStateToleratesALegacyStoredName(t *testing.T) {
	t.Parallel()

	model := recordResourceModel{
		Domain: types.StringValue("example.com"),
		Name:   types.StringValue("www"),
		Value:  types.StringValue("203.0.113.10"),
	}
	mapRecordToState(&models.DnsRecord{Type: "A", Name: "www.example.com", Value: "203.0.113.10"}, &model)

	if got := model.Name.ValueString(); got != "www" {
		t.Errorf("name: got %q, want %q", got, "www")
	}
}

func TestMapRecordToStateNumericFields(t *testing.T) {
	t.Parallel()

	srv := recordResourceModel{Domain: types.StringValue("example.com")}
	mapRecordToState(&models.DnsRecord{Type: "SRV", Name: "_sip._tcp", Value: "sip.example.com", Priority: 10, Weight: 5, Port: 5060}, &srv)
	if srv.Priority.ValueInt64() != 10 || srv.Weight.ValueInt64() != 5 || srv.Port.ValueInt64() != 5060 {
		t.Errorf("SRV: got priority=%v weight=%v port=%v", srv.Priority, srv.Weight, srv.Port)
	}

	// MX priority 0 is a legitimate value, not an absent one.
	mx := recordResourceModel{Domain: types.StringValue("example.com")}
	mapRecordToState(&models.DnsRecord{Type: "MX", Name: "@", Value: "mail.example.com", Priority: 0}, &mx)
	if mx.Priority.IsNull() || mx.Priority.ValueInt64() != 0 {
		t.Errorf("MX: got priority %v, want 0", mx.Priority)
	}
	if !mx.Weight.IsNull() || !mx.Port.IsNull() {
		t.Error("MX: weight and port must be null")
	}
}

func TestValidateRecordConfig(t *testing.T) {
	t.Parallel()

	i := types.Int64Value
	null := types.Int64Null()

	for name, tc := range map[string]struct {
		model     recordResourceModel
		wantError string
	}{
		"A": {model: recordResourceModel{Type: types.StringValue("A"), Value: types.StringValue("203.0.113.10"), Priority: null, Weight: null, Port: null}},
		"A with an IPv6 address": {
			model:     recordResourceModel{Type: types.StringValue("A"), Value: types.StringValue("2001:db8::1"), Priority: null, Weight: null, Port: null},
			wantError: "IPv4",
		},
		"AAAA": {model: recordResourceModel{Type: types.StringValue("AAAA"), Value: types.StringValue("2001:db8::1"), Priority: null, Weight: null, Port: null}},
		"AAAA with an IPv4 address": {
			model:     recordResourceModel{Type: types.StringValue("AAAA"), Value: types.StringValue("203.0.113.10"), Priority: null, Weight: null, Port: null},
			wantError: "IPv6",
		},
		"MX": {model: recordResourceModel{Type: types.StringValue("MX"), Value: types.StringValue("mail.example.com"), Priority: i(10), Weight: null, Port: null}},
		"MX without a priority": {
			model:     recordResourceModel{Type: types.StringValue("MX"), Value: types.StringValue("mail.example.com"), Priority: null, Weight: null, Port: null},
			wantError: "require priority",
		},
		"MX with a port": {
			model:     recordResourceModel{Type: types.StringValue("MX"), Value: types.StringValue("mail.example.com"), Priority: i(10), Weight: null, Port: i(25)},
			wantError: "port is not used",
		},
		"SRV": {model: recordResourceModel{Type: types.StringValue("SRV"), Value: types.StringValue("sip.example.com"), Priority: i(10), Weight: i(5), Port: i(5060)}},
		"SRV without a weight": {
			model:     recordResourceModel{Type: types.StringValue("SRV"), Value: types.StringValue("sip.example.com"), Priority: i(10), Weight: null, Port: i(5060)},
			wantError: "require weight",
		},
		"CNAME with a priority": {
			model:     recordResourceModel{Type: types.StringValue("CNAME"), Value: types.StringValue("target.example.org"), Priority: i(1), Weight: null, Port: null},
			wantError: "priority is not used",
		},
		"long TXT": {model: recordResourceModel{Type: types.StringValue("TXT"), Value: types.StringValue(strings.Repeat("a", 1000)), Priority: null, Weight: null, Port: null}},
		"long CNAME": {
			model:     recordResourceModel{Type: types.StringValue("CNAME"), Value: types.StringValue(strings.Repeat("a", 256)), Priority: null, Weight: null, Port: null},
			wantError: "at most 255",
		},
		"unknown priority is not judged until it is known": {
			model: recordResourceModel{Type: types.StringValue("CNAME"), Value: types.StringValue("target.example.org"), Priority: types.Int64Unknown(), Weight: null, Port: null},
		},
		"unknown value is not checked": {model: recordResourceModel{Type: types.StringValue("A"), Value: types.StringUnknown(), Priority: null, Weight: null, Port: null}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			diags := validateRecordConfig(tc.model)
			if tc.wantError == "" {
				if diags.HasError() {
					t.Fatalf("unexpected error: %v", diags.Errors())
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("expected an error containing %q", tc.wantError)
			}
			if detail := diags.Errors()[0].Detail(); !strings.Contains(detail, tc.wantError) {
				t.Errorf("error %q does not contain %q", detail, tc.wantError)
			}
		})
	}
}

func TestAppwriteNameservers(t *testing.T) {
	t.Parallel()

	domain := &models.Domain{
		Domain: "example.com",
		DnsRecords: []models.DnsRecord{
			{Type: "NS", Name: "@", Value: "ns2.appwrite.zone", Lock: true},
			// Stored before names were normalized.
			{Type: "NS", Name: "example.com", Value: "NS1.appwrite.zone.", Lock: true},
			// A delegation the user added is not one of Appwrite's.
			{Type: "NS", Name: "sub", Value: "ns1.elsewhere.net", Lock: false},
			{Type: "NS", Name: "@", Value: "ns3.elsewhere.net", Lock: false},
			{Type: "A", Name: "@", Value: "203.0.113.10", Lock: true},
		},
	}

	got := appwriteNameservers(domain)
	want := []string{"ns1.appwrite.zone", "ns2.appwrite.zone"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}

	if got := appwriteNameservers(&models.Domain{Domain: "example.com"}); got == nil || len(got) != 0 {
		t.Errorf("a domain without records must give an empty list, not null, got %#v", got)
	}
}

func TestDomainNameValidation(t *testing.T) {
	t.Parallel()

	for name, valid := range map[string]bool{
		"example.com":           true,
		"sub.example.co.uk":     true,
		"xn--bcher-kva.example": true,
		"Example.com":           false,
		"example.com.":          false,
		"https://example.com":   false,
		"example":               false,
		"-example.com":          false,
		"example.com/path":      false,
	} {
		if got := domainNamePattern.MatchString(name); got != valid {
			t.Errorf("%q: got valid=%v, want %v", name, got, valid)
		}
	}
}

// scriptedVerification serves the nameserver verification route, answering
// each call with the next status in the script: 200 reports the domain as
// delegated, 202 as not yet delegated (sent as a 200), and anything else as an
// Appwrite error with that status. The last entry repeats.
func scriptedVerification(t *testing.T, script ...int) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/domains/domain-id/nameservers/verification" {
			http.Error(w, "unexpected request "+r.Method+" "+r.URL.Path, http.StatusNotFound)
			return
		}
		call := int(calls.Add(1)) - 1
		status := script[min(call, len(script)-1)]

		w.Header().Set("Content-Type", "application/json")
		if status != http.StatusOK && status != http.StatusAccepted {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "scripted", "code": status, "type": "scripted"})
			return
		}
		nameservers := ""
		if status == http.StatusOK {
			nameservers = "Appwrite"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"$id":         "domain-id",
			"domain":      "example.com",
			"nameservers": nameservers,
			"dnsRecords": []map[string]any{
				{"type": "NS", "name": "@", "value": "ns1.appwrite.zone", "lock": true},
			},
		})
	}))
	t.Cleanup(server.Close)
	return server, &calls
}

func testDomainsClient(url string) *domains.Domains {
	// No retries: the SDK client here is bare, so each scripted response is
	// seen by the code under test exactly once.
	return appwrite.NewDomains(appwrite.NewClient(appwrite.WithEndpoint(url+"/v1"), appwrite.WithProject("console")))
}

// The polling tests share the package-level interval, so they do not run in
// parallel.
func TestWaitForDelegation(t *testing.T) {
	original := verificationInterval
	verificationInterval = time.Millisecond
	t.Cleanup(func() { verificationInterval = original })

	t.Run("checks until delegated", func(t *testing.T) {
		server, calls := scriptedVerification(t, http.StatusAccepted, http.StatusAccepted, http.StatusOK)

		domain, err := waitForDelegation(t.Context(), testDomainsClient(server.URL), "domain-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if domain.Id != "domain-id" {
			t.Errorf("got domain %q", domain.Id)
		}
		if got := calls.Load(); got != 3 {
			t.Errorf("got %d checks, want 3", got)
		}
	})

	t.Run("waits out rate limits and server errors", func(t *testing.T) {
		server, calls := scriptedVerification(t, http.StatusTooManyRequests, http.StatusBadGateway, http.StatusOK)

		if _, err := waitForDelegation(t.Context(), testDomainsClient(server.URL), "domain-id"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := calls.Load(); got != 3 {
			t.Errorf("got %d checks, want 3", got)
		}
	})

	t.Run("fails at once on an error asking again cannot fix", func(t *testing.T) {
		server, calls := scriptedVerification(t, http.StatusUnauthorized, http.StatusOK)

		if _, err := waitForDelegation(t.Context(), testDomainsClient(server.URL), "domain-id"); err == nil {
			t.Fatal("expected an error")
		}
		if got := calls.Load(); got != 1 {
			t.Errorf("got %d checks, want 1", got)
		}
	})

	t.Run("names Appwrite's nameservers when it times out", func(t *testing.T) {
		server, _ := scriptedVerification(t, http.StatusAccepted)

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		_, err := waitForDelegation(ctx, testDomainsClient(server.URL), "domain-id")
		if err == nil {
			t.Fatal("expected a timeout")
		}
		if !strings.Contains(err.Error(), "ns1.appwrite.zone") {
			t.Errorf("error does not name the nameservers to set: %v", err)
		}
	})
}

func TestMapDomainToStateCase(t *testing.T) {
	t.Parallel()

	stored := &models.Domain{Id: "id", Domain: "Example.com"}

	// After an import there is no spelling to keep. The resource takes the
	// lowercase form its configuration must use; anything else would plan a
	// replacement that deletes the zone.
	imported := domainResourceModel{Domain: types.StringNull()}
	mapDomainToState(t.Context(), stored, &imported, true)
	if got := imported.Domain.ValueString(); got != "example.com" {
		t.Errorf("resource after import: got %q, want %q", got, "example.com")
	}

	configured := domainResourceModel{Domain: types.StringValue("example.com")}
	mapDomainToState(t.Context(), stored, &configured, true)
	if got := configured.Domain.ValueString(); got != "example.com" {
		t.Errorf("resource: got %q, want the configured spelling", got)
	}

	// The data source accepts any case, and keeps what it was given.
	lookup := domainResourceModel{Domain: types.StringValue("EXAMPLE.com")}
	mapDomainToState(t.Context(), stored, &lookup, false)
	if got := lookup.Domain.ValueString(); got != "EXAMPLE.com" {
		t.Errorf("data source: got %q, want the configured spelling", got)
	}

	byID := domainResourceModel{Domain: types.StringNull()}
	mapDomainToState(t.Context(), stored, &byID, false)
	if got := byID.Domain.ValueString(); got != "Example.com" {
		t.Errorf("data source by ID: got %q, want the stored spelling", got)
	}
}

func TestFindDomainByName(t *testing.T) {
	t.Parallel()

	// md5("example.com")
	const hashedID = "5ababd603b22780302dd8d83498e5172"

	newServer := func(t *testing.T, hashed bool) (*httptest.Server, *[]string) {
		var requests []string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests = append(requests, r.Method+" "+r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.URL.Path == "/v1/domains/"+hashedID && hashed:
				_ = json.NewEncoder(w).Encode(map[string]any{"$id": hashedID, "domain": "example.com"})
			case r.URL.Path == "/v1/domains/bought-id":
				_ = json.NewEncoder(w).Encode(map[string]any{"$id": "bought-id", "domain": "Example.com"})
			case r.URL.Path == "/v1/domains" && !hashed:
				_ = json.NewEncoder(w).Encode(map[string]any{"total": 1, "domains": []map[string]any{{"$id": "bought-id", "domain": "Example.com"}}})
			default:
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "not found", "code": 404, "type": "domain_not_found"})
			}
		}))
		t.Cleanup(server.Close)
		return server, &requests
	}

	t.Run("by the ID its name hashes to", func(t *testing.T) {
		t.Parallel()
		server, requests := newServer(t, true)

		domain, err := findDomainByName(testDomainsClient(server.URL), "org", "Example.COM")
		if err != nil || domain == nil || domain.Id != hashedID {
			t.Fatalf("got %+v, %v", domain, err)
		}
		if len(*requests) != 1 {
			t.Errorf("got requests %v, want a single get", *requests)
		}
	})

	t.Run("by query when the ID was hashed from another spelling", func(t *testing.T) {
		t.Parallel()
		server, _ := newServer(t, false)

		domain, err := findDomainByName(testDomainsClient(server.URL), "org", "example.com")
		if err != nil || domain == nil || domain.Id != "bought-id" {
			t.Fatalf("got %+v, %v", domain, err)
		}
	})
}
