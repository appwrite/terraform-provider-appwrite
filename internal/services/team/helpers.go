package team

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// prefsDescription opens the description of prefs on the resource and the data
// source.
const prefsDescription = "The team's preferences, shared by every member, as a JSON-encoded object. Use jsonencode " +
	"to write it and jsondecode to read it."

// emptyPrefs is how a team with no preferences reads back, and the value every
// new team starts with.
const emptyPrefs = "{}"

// canonicalPrefs re-encodes a JSON object so that two spellings of the same
// preferences compare equal.
//
// Appwrite keeps keys in the order they were sent, while jsonencode sorts them,
// and the server's PHP decoder turns 1.0 into 1 and an integer beyond int64
// into a float. Comparing raw strings would report each of those as drift.
// Re-encoding sorts keys, and numbers are reduced to the form the server
// stores: an integer when the value is integral and fits in int64, a float
// otherwise. An integer that fits keeps its exact digits rather than passing
// through float64, which would round anything beyond 2^53.
func canonicalPrefs(raw string) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	var value interface{}
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return "", errors.New("unexpected data after the JSON object")
	}
	if _, ok := value.(map[string]interface{}); !ok {
		return "", fmt.Errorf("got %s", jsonKind(value))
	}
	encoded, err := json.Marshal(canonicalNumbers(value))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func jsonKind(value interface{}) string {
	switch value.(type) {
	case []interface{}:
		return "an array"
	case string:
		return "a string"
	case json.Number:
		return "a number"
	case bool:
		return "a boolean"
	default:
		return "null"
	}
}

func canonicalNumbers(value interface{}) interface{} {
	switch v := value.(type) {
	case json.Number:
		if _, err := v.Int64(); err == nil {
			return v
		}
		f, err := v.Float64()
		if err != nil {
			return v
		}
		if f == math.Trunc(f) && f >= math.MinInt64 && f < math.MaxInt64 {
			return json.Number(strconv.FormatInt(int64(f), 10))
		}
		return f
	case map[string]interface{}:
		for key, element := range v {
			v[key] = canonicalNumbers(element)
		}
	case []interface{}:
		for i, element := range v {
			v[i] = canonicalNumbers(element)
		}
	}
	return value
}

// prefsEqual reports whether two JSON objects hold the same preferences. A
// value that does not parse is equal to nothing.
func prefsEqual(a, b string) bool {
	ca, err := canonicalPrefs(a)
	if err != nil {
		return false
	}
	cb, err := canonicalPrefs(b)
	if err != nil {
		return false
	}
	return ca == cb
}

// prefsForState chooses what to store for the preferences the server returned.
//
// The prior value is kept when it holds the same preferences, so the spelling
// the configuration used -- jsonencode's key order, 1.0 for 1 -- is not
// reported as a change. Anything else is real drift and is stored in canonical
// form.
func prefsForState(prior types.String, actual string) (types.String, error) {
	canonical, err := canonicalPrefs(actual)
	if err != nil {
		return types.StringNull(), fmt.Errorf("decoding team preferences: %w", err)
	}
	if !prior.IsNull() && !prior.IsUnknown() && prefsEqual(prior.ValueString(), canonical) {
		return prior, nil
	}
	return types.StringValue(canonical), nil
}

// rawPrefs extracts the prefs object from a team response. The SDK models it
// as a nested struct whose Decode refuses to run, so it has to be read out of
// the whole response instead.
func rawPrefs(decode func(interface{}) error) (string, error) {
	var body struct {
		Prefs json.RawMessage `json:"prefs"`
	}
	if err := decode(&body); err != nil {
		return "", fmt.Errorf("decoding team: %w", err)
	}
	// Older servers may omit the field; a team with no preferences reads the
	// same either way.
	if len(body.Prefs) == 0 || string(body.Prefs) == "null" || string(body.Prefs) == "[]" {
		return emptyPrefs, nil
	}
	return string(body.Prefs), nil
}

// prefsValidator rejects a prefs value that is not a JSON object at plan time,
// which the server would otherwise refuse only once the team exists.
type prefsValidator struct{}

func (prefsValidator) Description(context.Context) string {
	return "value must be a JSON object"
}

func (v prefsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (prefsValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := canonicalPrefs(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid team preferences",
			fmt.Sprintf("prefs must be a JSON object, such as the output of jsonencode({...}), but %s.", err))
	}
}

// roleKeyPattern is Appwrite's rule for a membership role: letters, digits,
// period, hyphen and underscore, not starting with one of the last three, at
// most 81 characters.
var roleKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,80}$`)

// maxRoles is the server's limit on the roles one membership can hold.
const maxRoles = 100

func roleValidators() []validator.Set {
	return []validator.Set{
		setvalidator.SizeAtMost(maxRoles),
		setvalidator.ValueStringsAre(stringvalidator.RegexMatches(roleKeyPattern,
			"must be 1 to 81 letters, digits, periods, hyphens or underscores, starting with a letter or digit")),
	}
}

// importedKey marks, in private state, a membership that was imported rather
// than created by this resource.
const importedKey = "imported"

// requiresReplaceUnlessAdopting forces replacement when a creation argument
// changes, except for an imported membership's first plan.
//
// Import has no way to recover whether a membership was created from a user
// ID, an email address or a phone number, so it leaves email, phone and name
// null. Without this, a configuration giving any of them would plan to
// replace the membership it was just imported for. Update checks that an
// adopted email or phone actually belongs to the membership's user.
func requiresReplaceUnlessAdopting() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = true
			if !req.StateValue.IsNull() || req.Private == nil {
				return
			}
			imported, diags := req.Private.GetKey(ctx, importedKey)
			resp.Diagnostics.Append(diags...)
			if len(imported) > 0 {
				resp.RequiresReplace = false
			}
		},
		"Changing this forces a new resource to be created, except when it is first set on an imported membership.",
		"Changing this forces a new resource to be created, except when it is first set on an imported membership.",
	)
}
