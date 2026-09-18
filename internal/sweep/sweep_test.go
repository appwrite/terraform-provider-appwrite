package sweep_test

import (
	"testing"

	"github.com/appwrite/terraform-provider-appwrite/internal/sweep"
)

func TestSweepable(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		identifiers []string
		want        bool
	}{
		"prefixed id":              {identifiers: []string{"tf-acc-test-webhook"}, want: true},
		"prefix anywhere in name":  {identifiers: []string{"Order tf-acc-test hook"}, want: true},
		"one of several matches":   {identifiers: []string{"6812ab", "tf-acc-test-bucket"}, want: true},
		"unprefixed":               {identifiers: []string{"production-webhook"}, want: false},
		"empty":                    {identifiers: []string{""}, want: false},
		"no identifiers":           {identifiers: nil, want: false},
		"near miss must not match": {identifiers: []string{"tf-acc"}, want: false},
		"plausible real name":      {identifiers: []string{"terraform-managed-bucket"}, want: false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := sweep.Sweepable(tc.identifiers...); got != tc.want {
				t.Errorf("Sweepable(%q) = %v, want %v", tc.identifiers, got, tc.want)
			}
		})
	}
}

// TestLoadConfigRefusesInheritedProject is the safety property that matters
// most: a sweep must not silently target whichever project the environment
// happens to name, because that is how a sweeper deletes production.
func TestLoadConfigRefusesInheritedProject(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "https://cloud.appwrite.io/v1")
	t.Setenv("APPWRITE_API_KEY", "standard_notarealkey")
	t.Setenv("APPWRITE_PROJECT_ID", "someones-production-project")
	t.Setenv("APPWRITE_SWEEP_PROJECT_ID", "")

	if _, err := sweep.LoadConfig(); err == nil {
		t.Fatal("LoadConfig accepted a run with no APPWRITE_SWEEP_PROJECT_ID; it must not fall back to APPWRITE_PROJECT_ID")
	}
}

func TestLoadConfigUsesSweepProject(t *testing.T) {
	t.Setenv("APPWRITE_ENDPOINT", "https://cloud.appwrite.io/v1")
	t.Setenv("APPWRITE_API_KEY", "standard_notarealkey")
	t.Setenv("APPWRITE_PROJECT_ID", "someones-production-project")
	t.Setenv("APPWRITE_SWEEP_PROJECT_ID", "the-disposable-one")

	cfg, err := sweep.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.ProjectID != "the-disposable-one" {
		t.Errorf("ProjectID = %q, want the project named by APPWRITE_SWEEP_PROJECT_ID", cfg.ProjectID)
	}
}
