package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/client"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
)

// The project's auth policies, as plain values.
//
// GET /project/policies reports each policy under its ID with the fields its
// update route takes, so these structs mirror the routes rather than the
// project document.
type passwordStrengthPolicy struct {
	Min       int64 `json:"min"`
	Uppercase bool  `json:"uppercase"`
	Lowercase bool  `json:"lowercase"`
	Number    bool  `json:"number"`
	Symbols   bool  `json:"symbols"`
}

type passwordPwnedPolicy struct {
	Enabled  bool `json:"enabled"`
	Sessions bool `json:"sessions"`
	Users    bool `json:"users"`
}

type membershipPrivacyPolicy struct {
	UserID         bool `json:"userId"`
	UserEmail      bool `json:"userEmail"`
	UserPhone      bool `json:"userPhone"`
	UserName       bool `json:"userName"`
	UserMFA        bool `json:"userMFA"`
	UserAccessedAt bool `json:"userAccessedAt"`
}

type mfaFactorsPolicy struct {
	TOTP   bool `json:"totp"`
	Email  bool `json:"email"`
	Phone  bool `json:"phone"`
	Custom bool `json:"custom"`
}

type authPolicies struct {
	PasswordDictionary   bool
	PasswordHistory      int64
	PasswordPersonalData bool
	PasswordStrength     passwordStrengthPolicy
	PasswordPwned        passwordPwnedPolicy
	SessionAlert         bool
	SessionDuration      int64
	SessionInvalidation  bool
	SessionLimit         int64
	UserLimit            int64
	MembershipPrivacy    membershipPrivacyPolicy
	MFAFactors           mfaFactorsPolicy
	DenyAliasedEmail     bool
	DenyDisposableEmail  bool
	DenyFreeEmail        bool
	DenyCorporateEmail   bool
}

// Defaults of a new project, from the values Appwrite 2.2.0 writes when it
// creates one and, for settings it does not write, the fallback its policy
// routes report. password-pwned is from 2.3.0, which added it.
const (
	defaultPasswordStrengthMin = 8
	defaultSessionDuration     = 31536000 // one year, in seconds
	defaultSessionLimit        = 10
)

var defaultAuthPolicies = authPolicies{
	PasswordStrength:    passwordStrengthPolicy{Min: defaultPasswordStrengthMin},
	PasswordPwned:       passwordPwnedPolicy{Enabled: true},
	SessionDuration:     defaultSessionDuration,
	SessionInvalidation: true,
	SessionLimit:        defaultSessionLimit,
	MFAFactors:          mfaFactorsPolicy{TOTP: true, Email: true, Phone: true},
}

// Bounds the policy routes accept.
const (
	maxPasswordHistory    = 20
	minPasswordStrength   = 8
	maxPasswordStrength   = 256
	minSessionDuration    = 60
	maxSessionDuration    = 31536000
	minSessionLimit       = 1
	maxSessionLimit       = 100
	maxUserLimit          = 10000
	passwordPwnedPolicyID = "password-pwned"
)

// reportedPolicies is what the server returned: the policies, and which of
// them it reported at all. Servers older than a policy do not report it.
type reportedPolicies struct {
	policies authPolicies
	reported map[string]bool
}

type authPoliciesClient struct {
	project *sdkproject.Project
	raw     client.Client
}

func newAuthPoliciesClient(c client.Client) *authPoliciesClient {
	return &authPoliciesClient{project: sdkproject.New(c), raw: c}
}

func (c *authPoliciesClient) list() (*reportedPolicies, error) {
	list, err := c.project.ListPolicies()
	if err != nil {
		return nil, err
	}
	var decoded struct {
		Policies []json.RawMessage `json:"policies"`
	}
	if err := list.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("decoding policies: %w", err)
	}

	result := &reportedPolicies{policies: defaultAuthPolicies, reported: map[string]bool{}}
	for _, raw := range decoded.Policies {
		var header struct {
			ID string `json:"$id"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, fmt.Errorf("decoding policy: %w", err)
		}
		policy, known := policyByID[header.ID]
		if !known {
			// A policy added in a later Appwrite version than this provider
			// knows about.
			continue
		}
		if err := policy.decode(raw, &result.policies); err != nil {
			return nil, fmt.Errorf("decoding policy %s: %w", header.ID, err)
		}
		result.reported[header.ID] = true
	}
	return result, nil
}

// patch sends a policy update directly. Only password-history needs it: its
// route disables the history with null and rejects 0, and the SDK takes the
// length as an int it always sends.
func (c *authPoliciesClient) patch(policyID string, body map[string]any) error {
	_, err := c.raw.Call("PATCH", "/project/policies/"+client.EncodePath(policyID), map[string]any{
		// client.Call does not add the project header; see common.GetColumnRawJSON.
		"X-Appwrite-Project": c.raw.Config["project"],
		"content-type":       contentTypeJSON,
		"accept":             contentTypeJSON,
	}, body)
	return err
}

// policy describes one policy route: how to read its value out of the list
// response, whether two settings differ in it, and how to send it.
type policy struct {
	id     string
	decode func(raw json.RawMessage, into *authPolicies) error
	equal  func(a, b *authPolicies) bool
	update func(c *authPoliciesClient, want *authPolicies) error
}

func enabledPolicy(id string, field func(*authPolicies) *bool, update func(*sdkproject.Project, bool) error) policy {
	return policy{
		id: id,
		decode: func(raw json.RawMessage, into *authPolicies) error {
			var v struct {
				Enabled bool `json:"enabled"`
			}
			err := json.Unmarshal(raw, &v)
			*field(into) = v.Enabled
			return err
		},
		equal: func(a, b *authPolicies) bool { return *field(a) == *field(b) },
		update: func(c *authPoliciesClient, want *authPolicies) error {
			return update(c.project, *field(want))
		},
	}
}

func numberPolicy(id, key string, field func(*authPolicies) *int64, update func(*authPoliciesClient, int64) error) policy {
	return policy{
		id: id,
		decode: func(raw json.RawMessage, into *authPolicies) error {
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				return err
			}
			value, ok := fields[key]
			if !ok {
				return fmt.Errorf("missing %q", key)
			}
			return json.Unmarshal(value, field(into))
		},
		equal: func(a, b *authPolicies) bool { return *field(a) == *field(b) },
		update: func(c *authPoliciesClient, want *authPolicies) error {
			return update(c, *field(want))
		},
	}
}

func objectPolicy[T comparable](id string, field func(*authPolicies) *T, update func(*sdkproject.Project, T) error) policy {
	return policy{
		id: id,
		decode: func(raw json.RawMessage, into *authPolicies) error {
			// Start from what is already there, so a field missing from the
			// response keeps its default.
			return json.Unmarshal(raw, field(into))
		},
		equal: func(a, b *authPolicies) bool { return *field(a) == *field(b) },
		update: func(c *authPoliciesClient, want *authPolicies) error {
			return update(c.project, *field(want))
		},
	}
}

func discard[T any](_ T, err error) error { return err }

// policies lists every policy route, in the order GET /project/policies
// reports them.
var policies = []policy{
	enabledPolicy("password-dictionary", func(p *authPolicies) *bool { return &p.PasswordDictionary },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdatePasswordDictionaryPolicy(v)) }),
	numberPolicy("password-history", "total", func(p *authPolicies) *int64 { return &p.PasswordHistory },
		func(c *authPoliciesClient, v int64) error {
			if v == 0 {
				return c.patch("password-history", map[string]any{"total": nil})
			}
			return discard(c.project.UpdatePasswordHistoryPolicy(int(v)))
		}),
	objectPolicy("password-strength", func(p *authPolicies) *passwordStrengthPolicy { return &p.PasswordStrength },
		func(c *sdkproject.Project, v passwordStrengthPolicy) error {
			return discard(c.UpdatePasswordStrengthPolicy(
				c.WithUpdatePasswordStrengthPolicyMin(int(v.Min)),
				c.WithUpdatePasswordStrengthPolicyUppercase(v.Uppercase),
				c.WithUpdatePasswordStrengthPolicyLowercase(v.Lowercase),
				c.WithUpdatePasswordStrengthPolicyNumber(v.Number),
				c.WithUpdatePasswordStrengthPolicySymbols(v.Symbols),
			))
		}),
	enabledPolicy("password-personal-data", func(p *authPolicies) *bool { return &p.PasswordPersonalData },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdatePasswordPersonalDataPolicy(v)) }),
	objectPolicy(passwordPwnedPolicyID, func(p *authPolicies) *passwordPwnedPolicy { return &p.PasswordPwned },
		func(c *sdkproject.Project, v passwordPwnedPolicy) error {
			return discard(c.UpdatePasswordPwnedPolicy(
				c.WithUpdatePasswordPwnedPolicyEnabled(v.Enabled),
				c.WithUpdatePasswordPwnedPolicySessions(v.Sessions),
				c.WithUpdatePasswordPwnedPolicyUsers(v.Users),
			))
		}),
	enabledPolicy("session-alert", func(p *authPolicies) *bool { return &p.SessionAlert },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateSessionAlertPolicy(v)) }),
	numberPolicy("session-duration", "duration", func(p *authPolicies) *int64 { return &p.SessionDuration },
		func(c *authPoliciesClient, v int64) error {
			return discard(c.project.UpdateSessionDurationPolicy(int(v)))
		}),
	enabledPolicy("session-invalidation", func(p *authPolicies) *bool { return &p.SessionInvalidation },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateSessionInvalidationPolicy(v)) }),
	numberPolicy("session-limit", "total", func(p *authPolicies) *int64 { return &p.SessionLimit },
		func(c *authPoliciesClient, v int64) error { return discard(c.project.UpdateSessionLimitPolicy(int(v))) }),
	numberPolicy("user-limit", "total", func(p *authPolicies) *int64 { return &p.UserLimit },
		func(c *authPoliciesClient, v int64) error { return discard(c.project.UpdateUserLimitPolicy(int(v))) }),
	objectPolicy("membership-privacy", func(p *authPolicies) *membershipPrivacyPolicy { return &p.MembershipPrivacy },
		func(c *sdkproject.Project, v membershipPrivacyPolicy) error {
			return discard(c.UpdateMembershipPrivacyPolicy(
				c.WithUpdateMembershipPrivacyPolicyUserId(v.UserID),
				c.WithUpdateMembershipPrivacyPolicyUserEmail(v.UserEmail),
				c.WithUpdateMembershipPrivacyPolicyUserPhone(v.UserPhone),
				c.WithUpdateMembershipPrivacyPolicyUserName(v.UserName),
				c.WithUpdateMembershipPrivacyPolicyUserMFA(v.UserMFA),
				c.WithUpdateMembershipPrivacyPolicyUserAccessedAt(v.UserAccessedAt),
			))
		}),
	objectPolicy("mfa-factors", func(p *authPolicies) *mfaFactorsPolicy { return &p.MFAFactors },
		func(c *sdkproject.Project, v mfaFactorsPolicy) error {
			return discard(c.UpdateMFAFactorsPolicy(
				c.WithUpdateMFAFactorsPolicyTotp(v.TOTP),
				c.WithUpdateMFAFactorsPolicyEmail(v.Email),
				c.WithUpdateMFAFactorsPolicyPhone(v.Phone),
				c.WithUpdateMFAFactorsPolicyCustom(v.Custom),
			))
		}),
	enabledPolicy("deny-aliased-email", func(p *authPolicies) *bool { return &p.DenyAliasedEmail },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateDenyAliasedEmailPolicy(v)) }),
	enabledPolicy("deny-disposable-email", func(p *authPolicies) *bool { return &p.DenyDisposableEmail },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateDenyDisposableEmailPolicy(v)) }),
	enabledPolicy("deny-free-email", func(p *authPolicies) *bool { return &p.DenyFreeEmail },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateDenyFreeEmailPolicy(v)) }),
	enabledPolicy("deny-corporate-email", func(p *authPolicies) *bool { return &p.DenyCorporateEmail },
		func(c *sdkproject.Project, v bool) error { return discard(c.UpdateDenyCorporateEmailPolicy(v)) }),
}

var policyByID = func() map[string]policy {
	byID := make(map[string]policy, len(policies))
	for _, p := range policies {
		byID[p.id] = p
	}
	return byID
}()

// errPolicyUnsupported reports a policy the server does not have, set to
// something other than its default.
var errPolicyUnsupported = errors.New("not supported by this Appwrite server")

// applyAuthPolicies updates the policies whose current value differs from the
// wanted one, and returns the policies as the server reports them afterwards.
//
// Only differing policies are sent. That keeps an apply that changes one
// setting to one write, and it is what lets a configuration that leaves a
// policy at its default work against a server too old to have the policy: an
// unreported policy at its default needs no request, and one set to anything
// else fails with errPolicyUnsupported rather than a 404 from a route that
// does not exist.
func applyAuthPolicies(c *authPoliciesClient, want authPolicies) (*reportedPolicies, error) {
	current, err := c.list()
	if err != nil {
		return nil, err
	}

	// Refused before anything is written, so an unsupported setting does not
	// leave the others half applied.
	var unsupported []string
	for _, p := range policies {
		if !current.reported[p.id] && !p.equal(&want, &defaultAuthPolicies) {
			unsupported = append(unsupported, p.id)
		}
	}
	if len(unsupported) > 0 {
		return nil, fmt.Errorf("policy %s: %w; leave it at its default or upgrade Appwrite", strings.Join(unsupported, ", "), errPolicyUnsupported)
	}

	var changed bool
	for _, p := range policies {
		if !current.reported[p.id] || p.equal(&current.policies, &want) {
			continue
		}
		if err := p.update(c, &want); err != nil {
			return nil, fmt.Errorf("policy %s: %w", p.id, err)
		}
		changed = true
	}
	if !changed {
		return current, nil
	}
	return c.list()
}
