package inference

import (
	"encoding/json"
	"testing"
)

func TestInferBasicTypes(t *testing.T) {
	cases := []struct {
		input    interface{}
		wantType string
	}{
		{true, "boolean"},
		{float64(42), "integer"},
		{float64(3.14), "number"},
		{"hello", "string"},
	}
	for _, c := range cases {
		got := InferSchema(c.input)
		if got.Type != c.wantType {
			t.Errorf("InferSchema(%v): want type=%s, got %s", c.input, c.wantType, got.Type)
		}
	}
}

func TestInferObject(t *testing.T) {
	body := []byte(`{"id": 1, "name": "Alice", "email": "alice@example.com"}`)
	schema, err := InferSchemaFromBytes(body)
	if err != nil {
		t.Fatal(err)
	}
	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}
	if schema.Properties["email"].Format != "email" {
		t.Errorf("expected email format, got %s", schema.Properties["email"].Format)
	}
}

func TestMergeSchemasWiden(t *testing.T) {
	a := &JSONSchemaType{Type: "integer"}
	b := &JSONSchemaType{Type: "string"}
	merged := MergeSchemas(a, b)
	if merged.OneOf == nil || len(merged.OneOf) != 2 {
		data, _ := json.Marshal(merged)
		t.Errorf("expected oneOf with 2 types, got: %s", data)
	}
}

func TestMergeSchemasRequiredIntersection(t *testing.T) {
	a := &JSONSchemaType{
		Type: "object",
		Properties: map[string]*JSONSchemaType{
			"id":   {Type: "integer"},
			"name": {Type: "string"},
		},
		Required: []string{"id", "name"},
	}
	b := &JSONSchemaType{
		Type: "object",
		Properties: map[string]*JSONSchemaType{
			"id":    {Type: "integer"},
			"email": {Type: "string"},
		},
		Required: []string{"id", "email"},
	}
	merged := MergeSchemas(a, b)
	if len(merged.Required) != 1 || merged.Required[0] != "id" {
		t.Errorf("expected required=[id], got %v", merged.Required)
	}
}
