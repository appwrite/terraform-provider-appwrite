package project

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
)

// Appwrite's auth setting routes read the project's auth settings, change
// their own key and write the whole attribute back, so concurrent writes lose
// all but one of them. Terraform applies independent resources in parallel,
// so the resources here share one lock per project. The fake server loses
// writes the same way.

// TestFakeLosesConcurrentWrites shows the fake reproduces the race, so that
// TestAuthSettingsWritesAreSerialized passing means something.
func TestFakeLosesConcurrentWrites(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	fake.writeDelay = 20 * time.Millisecond
	project := appwrite.NewProject(appwrite.NewClient(testClients(server.URL).BaseOptions...))

	const phones = 5
	var wg sync.WaitGroup
	for i := range phones {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := project.CreateMockPhone(fmt.Sprintf("+1555555010%d", i), "123456"); err != nil {
				t.Errorf("create: %v", err)
			}
		}()
	}
	wg.Wait()

	if stored := len(fake.current().mocks); stored == phones {
		t.Fatalf("all %d concurrent writes survived, so the fake does not reproduce the lost update", phones)
	}
}

// TestAuthSettingsWritesAreSerialized applies auth methods, policies and ten
// mock numbers at once, as Terraform would, and checks every change survived.
func TestAuthSettingsWritesAreSerialized(t *testing.T) {
	t.Parallel()
	fake, server := newFakeAuthServer(t)
	fake.supportPasswordPwned()
	fake.writeDelay = 2 * time.Millisecond
	clients := testClients(server.URL)

	const phones = 10
	var wg sync.WaitGroup
	errs := make(chan string, phones+2)
	for i := range phones {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := newCRUD(t, NewMockPhoneResource(), clients)
			_, errText := c.create(mockPhone(fmt.Sprintf("+1555555020%d", i), "123456"))
			errs <- errText
		}()
	}
	wg.Add(2)
	go func() {
		defer wg.Done()
		c := newCRUD(t, NewAuthMethodsResource(), clients)
		_, errText := c.create(authMethodsPlan(map[string]bool{"email-password": true}))
		errs <- errText
	}()
	go func() {
		defer wg.Done()
		want := defaultAuthPolicies
		want.SessionLimit = 42
		want.SessionAlert = true
		want.MFAFactors.Custom = true
		c := newCRUD(t, NewAuthPoliciesResource(), clients)
		_, errText := c.create(authPoliciesPlan(t, want))
		errs <- errText
	}()
	wg.Wait()
	close(errs)
	for errText := range errs {
		if errText != "" {
			t.Error(errText)
		}
	}

	if fake.overlapped.Load() {
		t.Error("two writes to the project's auth settings were in flight at once")
	}
	got := fake.current()
	if len(got.mocks) != phones {
		t.Errorf("%d mock numbers stored, want %d", len(got.mocks), phones)
	}
	for _, method := range authMethods {
		if want := method.id == "email-password"; got.methods[method.id] != want {
			t.Errorf("auth method %s enabled=%t, want %t", method.id, got.methods[method.id], want)
		}
	}
	for id, want := range map[string]map[string]any{
		"session-limit": {"total": 42.0},
		"session-alert": {"enabled": true},
		"mfa-factors":   {"totp": true, "email": true, "phone": true, "custom": true},
	} {
		for key, value := range want {
			if got.policies[id][key] != value {
				t.Errorf("policy %s: %s = %v, want %v", id, key, got.policies[id][key], value)
			}
		}
	}
}
