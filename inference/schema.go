package inference

import (
	"encoding/json"
	"fmt"
	"math"
)

// JSONSchemaType represents an OpenAPI-compatible JSON Schema
type JSONSchemaType struct {
	Type                 string                     `json:"type,omitempty"`
	Format               string                     `json:"format,omitempty"`
	Properties           map[string]*JSONSchemaType `json:"properties,omitempty"`
	Items                *JSONSchemaType            `json:"items,omitempty"`
	Required             []string                   `json:"required,omitempty"`
	OneOf                []*JSONSchemaType           `json:"oneOf,omitempty"`
	AnyOf                []*JSONSchemaType           `json:"anyOf,omitempty"`
	Nullable             bool                       `json:"nullable,omitempty"`
	AdditionalProperties *JSONSchemaType            `json:"additionalProperties,omitempty"`
	Example              interface{}                `json:"example,omitempty"`
}

// InferSchema infers a JSONSchemaType from an arbitrary Go value
func InferSchema(v interface{}) *JSONSchemaType {
	if v == nil {
		return &JSONSchemaType{Nullable: true}
	}

	switch val := v.(type) {
	case bool:
		return &JSONSchemaType{Type: "boolean", Example: val}

	case float64:
		if val == math.Trunc(val) {
			return &JSONSchemaType{Type: "integer", Format: "int64", Example: val}
		}
		return &JSONSchemaType{Type: "number", Format: "double", Example: val}

	case string:
		s := &JSONSchemaType{Type: "string", Example: val}
		s.Format = inferStringFormat(val)
		return s

	case map[string]interface{}:
		schema := &JSONSchemaType{
			Type:       "object",
			Properties: make(map[string]*JSONSchemaType),
		}
		for k, child := range val {
			schema.Properties[k] = InferSchema(child)
			schema.Required = append(schema.Required, k)
		}
		return schema

	case []interface{}:
		if len(val) == 0 {
			return &JSONSchemaType{Type: "array", Items: &JSONSchemaType{}}
		}
		// Infer from first element; merge all
		itemSchema := InferSchema(val[0])
		for _, item := range val[1:] {
			itemSchema = MergeSchemas(itemSchema, InferSchema(item))
		}
		return &JSONSchemaType{Type: "array", Items: itemSchema}
	}

	return &JSONSchemaType{Type: "string"}
}

// InferSchemaFromBytes parses JSON bytes and infers a schema
func InferSchemaFromBytes(body []byte) (*JSONSchemaType, error) {
	if len(body) == 0 {
		return nil, nil
	}
	var v interface{}
	if err := json.Unmarshal(body, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	return InferSchema(v), nil
}

// MergeSchemas merges two schemas into the most permissive common schema
func MergeSchemas(a, b *JSONSchemaType) *JSONSchemaType {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}

	// Same type: deep merge
	if a.Type == b.Type {
		return mergeCompatible(a, b)
	}

	// Nullable handling
	if a.Nullable || b.Nullable {
		merged := mergeSchemaTypes(a, b)
		merged.Nullable = true
		return merged
	}

	// Widening: int → number
	if isNumeric(a.Type) && isNumeric(b.Type) {
		return &JSONSchemaType{Type: "number", Format: "double"}
	}

	// Different types → oneOf
	return &JSONSchemaType{OneOf: deduplicateSchemas([]*JSONSchemaType{a, b})}
}

func mergeCompatible(a, b *JSONSchemaType) *JSONSchemaType {
	result := &JSONSchemaType{
		Type:    a.Type,
		Format:  a.Format,
		Nullable: a.Nullable || b.Nullable,
	}

	if a.Type == "object" {
		result.Properties = make(map[string]*JSONSchemaType)
		// Merge all keys from both
		allKeys := unionKeys(a.Properties, b.Properties)
		for _, k := range allKeys {
			ap, aOk := a.Properties[k]
			bp, bOk := b.Properties[k]
			if aOk && bOk {
				result.Properties[k] = MergeSchemas(ap, bp)
			} else if aOk {
				result.Properties[k] = ap
			} else {
				result.Properties[k] = bp
			}
		}
		// Required = intersection (only required if present in both)
		result.Required = intersectStrings(a.Required, b.Required)
	}

	if a.Type == "array" {
		result.Items = MergeSchemas(a.Items, b.Items)
	}

	return result
}

func mergeSchemaTypes(a, b *JSONSchemaType) *JSONSchemaType {
	if a.Type == b.Type {
		return mergeCompatible(a, b)
	}
	return &JSONSchemaType{OneOf: deduplicateSchemas([]*JSONSchemaType{a, b})}
}

func deduplicateSchemas(schemas []*JSONSchemaType) []*JSONSchemaType {
	seen := map[string]bool{}
	result := []*JSONSchemaType{}
	for _, s := range schemas {
		key := s.Type + "|" + s.Format
		if !seen[key] {
			seen[key] = true
			result = append(result, s)
		}
	}
	return result
}

func isNumeric(t string) bool {
	return t == "integer" || t == "number"
}

func unionKeys(a, b map[string]*JSONSchemaType) []string {
	seen := map[string]bool{}
	keys := []string{}
	for k := range a {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	for k := range b {
		if !seen[k] {
			seen[k] = true
			keys = append(keys, k)
		}
	}
	return keys
}

func intersectStrings(a, b []string) []string {
	set := map[string]bool{}
	for _, s := range b {
		set[s] = true
	}
	result := []string{}
	for _, s := range a {
		if set[s] {
			result = append(result, s)
		}
	}
	return result
}
