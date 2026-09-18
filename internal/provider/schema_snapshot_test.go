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
	Type      string `json:"type"`
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
			Required:  attr.Required,
			Optional:  attr.Optional,
			Computed:  attr.Computed,
			Sensitive: attr.Sensitive,
			WriteOnly: attr.WriteOnly,
		}
		if attr.NestedType != nil {
			for _, nested := range attr.NestedType.Attributes {
				collectBlock(into, path, &tfprotov6.SchemaBlock{Attributes: []*tfprotov6.SchemaAttribute{nested}})
			}
		}
	}
	for _, nb := range block.BlockTypes {
		path := nb.TypeName
		if prefix != "" {
			path = prefix + "." + nb.TypeName
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

// breakingChanges compares two snapshots and returns the changes that would
// break an existing configuration or state.
//
// The taxonomy is AWS's, from contributing/breaking-changes.md. The asymmetry
// is the whole point: loosening a constraint is safe and tightening one is not,
// so Required becoming Optional is fine while the reverse breaks every
// configuration that omitted it.
func breakingChanges(old, current providerSnapshot) []string {
	out := breakingSchemaChanges("provider", old.Provider, current.Provider)
	out = append(out, breakingSetChanges("resource", old.Resources, current.Resources)...)
	out = append(out, breakingSetChanges("data source", old.DataSources, current.DataSources)...)
	out = append(out, breakingSetChanges("ephemeral resource", old.Ephemeral, current.Ephemeral)...)

	sort.Strings(out)
	return out
}

func breakingSetChanges(kind string, old, current map[string]snapshotSchema) []string {
	var out []string
	for name, oldSchema := range old {
		currentSchema, stillPresent := current[name]
		if !stillPresent {
			out = append(out, fmt.Sprintf("%s %s was removed", kind, name))
			continue
		}
		// A schema version going backwards would make Terraform refuse to
		// upgrade state it has already written.
		if currentSchema.Version < oldSchema.Version {
			out = append(out, fmt.Sprintf("%s %s schema version went backwards, %d to %d",
				kind, name, oldSchema.Version, currentSchema.Version))
		}
		out = append(out, breakingSchemaChanges(kind+" "+name, oldSchema, currentSchema)...)
	}
	return out
}

func breakingSchemaChanges(what string, old, current snapshotSchema) []string {
	var out []string
	for path, oldAttr := range old.Attributes {
		currentAttr, stillPresent := current.Attributes[path]
		if !stillPresent {
			out = append(out, fmt.Sprintf("%s: attribute %q was removed", what, path))
			continue
		}
		if oldAttr.Type != currentAttr.Type {
			out = append(out, fmt.Sprintf("%s: attribute %q changed type from %s to %s",
				what, path, oldAttr.Type, currentAttr.Type))
		}
		if !oldAttr.Required && currentAttr.Required {
			out = append(out, fmt.Sprintf("%s: attribute %q became required", what, path))
		}
		if oldAttr.Computed && !currentAttr.Computed {
			out = append(out, fmt.Sprintf("%s: attribute %q is no longer computed, so state written by an "+
				"older version has a value the provider will now plan to change", what, path))
		}
		if oldAttr.Optional && !currentAttr.Optional && !currentAttr.Required {
			out = append(out, fmt.Sprintf("%s: attribute %q is no longer settable", what, path))
		}
		// Dropping sensitivity publishes a value that used to be masked in plan
		// output and CI logs.
		if oldAttr.Sensitive && !currentAttr.Sensitive {
			out = append(out, fmt.Sprintf("%s: attribute %q is no longer sensitive", what, path))
		}
		if !oldAttr.WriteOnly && currentAttr.WriteOnly {
			out = append(out, fmt.Sprintf("%s: attribute %q became write-only, so it disappears from state", what, path))
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

func summarize(changes []string) string {
	return "  - " + strings.Join(changes, "\n  - ")
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
