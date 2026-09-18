package provider_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

// baselinePath is the committed snapshot the current schema is compared against.
const baselinePath = "../../.release/provider-schema.json"

// snapshotAttribute records only what a breaking-change comparison needs.
//
// Descriptions are deliberately left out. They change constantly and never break
// a configuration, and including them would make the baseline churn on every
// documentation edit until nobody read the diff.
type snapshotAttribute struct {
	Type string `json:"type"`
	// Nesting is the collection shape of a nested attribute or block: single,
	// list, set, map or group. Recorded separately from Type because every
	// nested attribute serializes as "object" regardless of shape, and moving a
	// nested block from list to set changes how a configuration must be written
	// and how state is keyed. Without this the two snapshots compare equal and
	// a breaking change walks straight through the gate.
	Nesting   string `json:"nesting,omitempty"`
	Required  bool   `json:"required,omitempty"`
	Optional  bool   `json:"optional,omitempty"`
	Computed  bool   `json:"computed,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
	WriteOnly bool   `json:"write_only,omitempty"`
}

type snapshotSchema struct {
	Version    int64                        `json:"version"`
	Attributes map[string]snapshotAttribute `json:"attributes"`
}

// providerSnapshot is the whole provider surface, in a form that serializes
// deterministically.
type providerSnapshot struct {
	Provider    snapshotSchema            `json:"provider"`
	Resources   map[string]snapshotSchema `json:"resources"`
	DataSources map[string]snapshotSchema `json:"data_sources"`
	Ephemeral   map[string]snapshotSchema `json:"ephemeral_resources"`
}

func takeProviderSnapshot() (providerSnapshot, error) {
	server, err := providerserver.NewProtocol6WithError(provider.New("test")())()
	if err != nil {
		return providerSnapshot{}, fmt.Errorf("creating provider server: %w", err)
	}
	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		return providerSnapshot{}, fmt.Errorf("GetProviderSchema: %w", err)
	}

	snap := providerSnapshot{
		Resources:   map[string]snapshotSchema{},
		DataSources: map[string]snapshotSchema{},
		Ephemeral:   map[string]snapshotSchema{},
	}
	if resp.Provider != nil {
		snap.Provider = convertSchema(resp.Provider)
	}
	for name, s := range resp.ResourceSchemas {
		snap.Resources[name] = convertSchema(s)
	}
	for name, s := range resp.DataSourceSchemas {
		snap.DataSources[name] = convertSchema(s)
	}
	for name, s := range resp.EphemeralResourceSchemas {
		snap.Ephemeral[name] = convertSchema(s)
	}
	return snap, nil
}

func convertSchema(s *tfprotov6.Schema) snapshotSchema {
	out := snapshotSchema{Version: s.Version, Attributes: map[string]snapshotAttribute{}}
	if s.Block == nil {
		return out
	}
	collectBlock(out.Attributes, "", s.Block)
	return out
}

func collectBlock(into map[string]snapshotAttribute, prefix string, block *tfprotov6.SchemaBlock) {
	for _, attr := range block.Attributes {
		path := attr.Name
		if prefix != "" {
			path = prefix + "." + attr.Name
		}
		into[path] = snapshotAttribute{
			Type:      attributeTypeString(attr),
			Nesting:   objectNestingString(attr),
			Required:  attr.Required,
			Optional:  attr.Optional,
			Computed:  attr.Computed,
			Sensitive: attr.Sensitive,
			WriteOnly: attr.WriteOnly,
		}
		if attr.NestedType != nil {
			collectBlock(into, path, &tfprotov6.SchemaBlock{Attributes: attr.NestedType.Attributes})
		}
	}
	for _, nb := range block.BlockTypes {
		path := nb.TypeName
		if prefix != "" {
			path = prefix + "." + nb.TypeName
		}
		// The block itself gets an entry, not just its contents. Otherwise a
		// block changing from list to single, or disappearing while its
		// attributes move elsewhere, leaves no trace in the snapshot.
		into[path] = snapshotAttribute{
			Type:    "block",
			Nesting: blockNestingString(nb.Nesting),
		}
		if nb.Block != nil {
			collectBlock(into, path, nb.Block)
		}
	}
}

func attributeTypeString(attr *tfprotov6.SchemaAttribute) string {
	if attr.NestedType != nil {
		return "object"
	}
	if attr.Type == nil {
		return "unknown"
	}
	return attr.Type.String()
}

func objectNestingString(attr *tfprotov6.SchemaAttribute) string {
	if attr.NestedType == nil {
		return ""
	}
	switch attr.NestedType.Nesting {
	case tfprotov6.SchemaObjectNestingModeSingle:
		return "single"
	case tfprotov6.SchemaObjectNestingModeList:
		return "list"
	case tfprotov6.SchemaObjectNestingModeSet:
		return "set"
	case tfprotov6.SchemaObjectNestingModeMap:
		return "map"
	case tfprotov6.SchemaObjectNestingModeInvalid:
		return "invalid"
	default:
		// Named rather than dropped: an unrecognized mode from a newer protocol
		// still has to differ from a recognized one, or the comparison silently
		// treats a change as no change.
		return fmt.Sprintf("unknown(%d)", attr.NestedType.Nesting)
	}
}

func blockNestingString(mode tfprotov6.SchemaNestedBlockNestingMode) string {
	switch mode {
	case tfprotov6.SchemaNestedBlockNestingModeSingle:
		return "single"
	case tfprotov6.SchemaNestedBlockNestingModeList:
		return "list"
	case tfprotov6.SchemaNestedBlockNestingModeSet:
		return "set"
	case tfprotov6.SchemaNestedBlockNestingModeMap:
		return "map"
	case tfprotov6.SchemaNestedBlockNestingModeGroup:
		return "group"
	case tfprotov6.SchemaNestedBlockNestingModeInvalid:
		return "invalid"
	default:
		return fmt.Sprintf("unknown(%d)", mode)
	}
}

// breakKind names a category of backwards-incompatible change.
//
// Typed rather than left implicit in the message so the tests can assert the
// classification instead of the prose. The sentence is for a human reading a
// failure; the kind is the contract.
type breakKind string

const (
	breakRemoved          breakKind = "removed"
	breakTypeChanged      breakKind = "type-changed"
	breakNestingChanged   breakKind = "nesting-changed"
	breakBecameRequired   breakKind = "became-required"
	breakComputedDropped  breakKind = "computed-dropped"
	breakNotSettable      breakKind = "not-settable"
	breakSensitiveDropped breakKind = "sensitive-dropped"
	breakBecameWriteOnly  breakKind = "became-write-only"
	breakVersionLowered   breakKind = "schema-version-lowered"
)

// breakingChange is one finding.
type breakingChange struct {
	Kind   breakKind
	Target string
	Detail string
}

func (b breakingChange) String() string {
	return fmt.Sprintf("%s: %s (%s)", b.Target, b.Detail, b.Kind)
}

// breakingChanges compares two snapshots and returns the changes that would
// break an existing configuration or state.
//
// The taxonomy is AWS's, from contributing/breaking-changes.md. The asymmetry
// is the whole point: loosening a constraint is safe and tightening one is not,
// so Required becoming Optional is fine while the reverse breaks every
// configuration that omitted it.
func breakingChanges(old, current providerSnapshot) []breakingChange {
	out := breakingSchemaChanges("provider", old.Provider, current.Provider)
	out = append(out, breakingSetChanges("resource", old.Resources, current.Resources)...)
	out = append(out, breakingSetChanges("data source", old.DataSources, current.DataSources)...)
	out = append(out, breakingSetChanges("ephemeral resource", old.Ephemeral, current.Ephemeral)...)

	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func breakingSetChanges(kind string, old, current map[string]snapshotSchema) []breakingChange {
	var out []breakingChange
	for name, oldSchema := range old {
		currentSchema, stillPresent := current[name]
		if !stillPresent {
			out = append(out, breakingChange{breakRemoved, kind + " " + name, "was removed"})
			continue
		}
		// A schema version going backwards would make Terraform refuse to
		// upgrade state it has already written.
		if currentSchema.Version < oldSchema.Version {
			out = append(out, breakingChange{breakVersionLowered, kind + " " + name,
				fmt.Sprintf("schema version went backwards, %d to %d", oldSchema.Version, currentSchema.Version)})
		}
		out = append(out, breakingSchemaChanges(kind+" "+name, oldSchema, currentSchema)...)
	}
	return out
}

func breakingSchemaChanges(what string, old, current snapshotSchema) []breakingChange {
	var out []breakingChange
	add := func(kind breakKind, path, detail string) {
		out = append(out, breakingChange{kind, fmt.Sprintf("%s attribute %q", what, path), detail})
	}

	for path, oldAttr := range old.Attributes {
		currentAttr, stillPresent := current.Attributes[path]
		if !stillPresent {
			add(breakRemoved, path, "was removed")
			continue
		}
		if oldAttr.Type != currentAttr.Type {
			add(breakTypeChanged, path, fmt.Sprintf("changed type from %s to %s", oldAttr.Type, currentAttr.Type))
		}
		// Changing a nested block or attribute between list, set, map and single
		// changes how the configuration is written and how state is keyed, and
		// the type alone does not show it.
		if oldAttr.Nesting != currentAttr.Nesting {
			add(breakNestingChanged, path, fmt.Sprintf("changed nesting mode from %q to %q", oldAttr.Nesting, currentAttr.Nesting))
		}
		if !oldAttr.Required && currentAttr.Required {
			add(breakBecameRequired, path, "became required")
		}
		if oldAttr.Computed && !currentAttr.Computed {
			add(breakComputedDropped, path, "is no longer computed, so state written by an older version "+
				"holds a value the provider will now plan to change")
		}
		if oldAttr.Optional && !currentAttr.Optional && !currentAttr.Required {
			add(breakNotSettable, path, "is no longer settable")
		}
		// Dropping sensitivity publishes a value that used to be masked in plan
		// output and CI logs.
		if oldAttr.Sensitive && !currentAttr.Sensitive {
			add(breakSensitiveDropped, path, "is no longer sensitive")
		}
		if !oldAttr.WriteOnly && currentAttr.WriteOnly {
			add(breakBecameWriteOnly, path, "became write-only, so it disappears from state")
		}
	}
	return out
}

func marshalSnapshot(s providerSnapshot) ([]byte, error) {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func summarize[T fmt.Stringer](changes []T) string {
	lines := make([]string, 0, len(changes))
	for _, c := range changes {
		lines = append(lines, c.String())
	}
	return "  - " + strings.Join(lines, "\n  - ")
}

// summarizeStrings is the plain-string variant, for the added-surface list which
// has no classification to carry.
func summarizeStrings(items []string) string {
	return "  - " + strings.Join(items, "\n  - ")
}

func unmarshalSnapshot(raw []byte, into *providerSnapshot) error {
	return json.Unmarshal(raw, into)
}

// addedSurface lists everything present now and absent from the baseline. Not
// breaking, but it means the baseline is stale.
func addedSurface(old, current providerSnapshot) []string {
	var out []string

	out = append(out, addedSet("resource", old.Resources, current.Resources)...)
	out = append(out, addedSet("data source", old.DataSources, current.DataSources)...)
	out = append(out, addedSet("ephemeral resource", old.Ephemeral, current.Ephemeral)...)
	for path := range current.Provider.Attributes {
		if _, ok := old.Provider.Attributes[path]; !ok {
			out = append(out, fmt.Sprintf("provider: attribute %q added", path))
		}
	}

	sort.Strings(out)
	return out
}

func addedSet(kind string, old, current map[string]snapshotSchema) []string {
	var out []string
	for name, currentSchema := range current {
		oldSchema, existed := old[name]
		if !existed {
			out = append(out, fmt.Sprintf("%s %s added", kind, name))
			continue
		}
		for path := range currentSchema.Attributes {
			if _, ok := oldSchema.Attributes[path]; !ok {
				out = append(out, fmt.Sprintf("%s %s: attribute %q added", kind, name, path))
			}
		}
	}
	return out
}
