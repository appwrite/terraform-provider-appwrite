package provider_test

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

// attributeInfo is one attribute of one resource, flattened out of the schema.
type attributeInfo struct {
	Resource        string
	Path            string
	Description     string
	ForcesNewOnAny  bool
	ForcesNewIfSet  bool
	IsRequired      bool
	IsOptional      bool
	IsComputed      bool
	IsSensitive     bool
	IsWriteOnly     bool
	AttributeGoType string
}

// walkResourceSchemas flattens every managed resource's schema into a list of
// attributes.
//
// Reflection rather than a type switch over the twenty-odd concrete attribute
// types the framework defines. A type switch would have to be extended every
// time a resource starts using a type it does not yet list, and the failure mode
// is silence: the attribute is skipped and whatever the test was asserting stops
// being asserted for it. Reading the fields by name is uglier and cannot miss.
func walkResourceSchemas() ([]attributeInfo, error) {
	ctx := context.Background()
	var out []attributeInfo

	for _, newResource := range provider.New("test")().Resources(ctx) {
		res := newResource()

		var meta resource.MetadataResponse
		res.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "appwrite"}, &meta)

		var schemaResp resource.SchemaResponse
		res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			return nil, fmt.Errorf("%s: schema returned diagnostics: %v", meta.TypeName, schemaResp.Diagnostics.Errors())
		}

		out = append(out, flattenAttributes(meta.TypeName, "", schemaResp.Schema.Attributes)...)
		out = append(out, flattenBlocks(meta.TypeName, "", schemaResp.Schema.Blocks)...)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Resource != out[j].Resource {
			return out[i].Resource < out[j].Resource
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

func flattenAttributes(resourceType, prefix string, attrs map[string]rschema.Attribute) []attributeInfo {
	var out []attributeInfo
	for name, attr := range attrs {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}

		info := attributeInfo{
			Resource:        resourceType,
			Path:            path,
			Description:     attr.GetDescription(),
			IsRequired:      attr.IsRequired(),
			IsOptional:      attr.IsOptional(),
			IsComputed:      attr.IsComputed(),
			IsSensitive:     attr.IsSensitive(),
			IsWriteOnly:     attr.IsWriteOnly(),
			AttributeGoType: reflect.TypeOf(attr).String(),
		}
		info.ForcesNewOnAny, info.ForcesNewIfSet = replacementBehavior(attr)
		out = append(out, info)

		// Nested attributes, whether single, list, set or map.
		if nested := nestedAttributes(attr); nested != nil {
			out = append(out, flattenAttributes(resourceType, path, nested)...)
		}
	}
	return out
}

func flattenBlocks(resourceType, prefix string, blocks map[string]rschema.Block) []attributeInfo {
	var out []attributeInfo
	for name, block := range blocks {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		nestedObj := block.GetNestedObject()
		if nestedObj == nil {
			continue
		}
		// GetAttributes returns the framework's internal map type, whose values
		// are the same Attribute interface; copy it into the exported shape the
		// walker uses everywhere else.
		attrs := make(map[string]rschema.Attribute)
		for name, attr := range nestedObj.GetAttributes() {
			attrs[name] = attr
		}
		out = append(out, flattenAttributes(resourceType, path, attrs)...)
	}
	return out
}

// nestedAttributes returns an attribute's nested attributes, or nil.
func nestedAttributes(attr rschema.Attribute) map[string]rschema.Attribute {
	nested, ok := attr.(interface {
		GetNestedObject() rschema.NestedAttributeObject
	})
	if !ok {
		return nil
	}
	obj := nested.GetNestedObject()
	if len(obj.Attributes) == 0 {
		return nil
	}
	return obj.Attributes
}

// replacementBehavior reports whether any plan modifier on the attribute forces
// replacement, and whether it only does so when the attribute is configured.
//
// Detected from what the modifier says about itself rather than its Go type. The
// framework builds RequiresReplace out of RequiresReplaceIf, so the concrete type
// is the same for both and for any provider-defined conditional modifier; the
// description is the thing that distinguishes them, and it is what ends up in
// documentation anyway.
func replacementBehavior(attr rschema.Attribute) (forcesNew bool, onlyIfConfigured bool) {
	field := reflect.ValueOf(attr)
	if field.Kind() == reflect.Pointer {
		field = field.Elem()
	}
	if field.Kind() != reflect.Struct {
		return false, false
	}
	modifiers := field.FieldByName("PlanModifiers")
	if !modifiers.IsValid() || modifiers.Kind() != reflect.Slice {
		return false, false
	}

	ctx := context.Background()
	for i := range modifiers.Len() {
		describer, ok := modifiers.Index(i).Interface().(interface {
			Description(context.Context) string
		})
		if !ok {
			continue
		}
		switch text := describer.Description(ctx); {
		case contains(text, "is configured and changes, Terraform will destroy and recreate"):
			return true, true
		case contains(text, "Terraform will destroy and recreate"):
			return true, false
		}
	}
	return false, false
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
