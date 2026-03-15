package inference

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Observation is a single captured request/response pair
type Observation struct {
	Method         string
	PathTemplate   string // normalized, e.g. /users/{id}
	RawPath        string
	QueryParams    map[string]string
	RequestHeaders http.Header
	RequestBody    []byte
	StatusCode     int
	ResponseBody   []byte
	ContentType    string
}

// OpenAPISpec is the live-built specification
type OpenAPISpec struct {
	OpenAPI    string                 `json:"openapi"`
	Info       Info                  `json:"info"`
	Paths      map[string]PathItem   `json:"paths"`
	Components Components            `json:"components,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type PathItem map[string]*Operation // key = lowercase HTTP method

type Operation struct {
	Summary     string              `json:"summary,omitempty"`
	OperationID string              `json:"operationId,omitempty"`
	Parameters  []Parameter         `json:"parameters,omitempty"`
	RequestBody *RequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Tags        []string            `json:"tags,omitempty"`
	// internal counters (not serialized)
	observationCount int `json:"-"`
	flaggedFields    []string `json:"-"`
}

type Parameter struct {
	Name     string          `json:"name"`
	In       string          `json:"in"` // path, query, header
	Required bool            `json:"required"`
	Schema   *JSONSchemaType `json:"schema"`
}

type RequestBody struct {
	Required bool                      `json:"required"`
	Content  map[string]MediaTypeEntry `json:"content"`
}

type Response struct {
	Description string                    `json:"description"`
	Content     map[string]MediaTypeEntry `json:"content,omitempty"`
}

type MediaTypeEntry struct {
	Schema *JSONSchemaType `json:"schema,omitempty"`
}

type Components struct {
	Schemas map[string]*JSONSchemaType `json:"schemas,omitempty"`
}

// SpecMerger maintains and updates the live OpenAPI spec
type SpecMerger struct {
	mu         sync.RWMutex
	spec       *OpenAPISpec
	normalizer *PathNormalizer
}

func NewSpecMerger(title string) *SpecMerger {
	return &SpecMerger{
		spec: &OpenAPISpec{
			OpenAPI: "3.0.3",
			Info:    Info{Title: title, Version: "0.0.0"},
			Paths:   map[string]PathItem{},
		},
		normalizer: NewPathNormalizer(),
	}
}

// Ingest processes a single observation and updates the spec
func (m *SpecMerger) Ingest(obs *Observation) {
	pathTemplate := m.normalizer.Observe(obs.RawPath)
	obs.PathTemplate = pathTemplate

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.spec.Paths[pathTemplate]; !ok {
		m.spec.Paths[pathTemplate] = PathItem{}
	}
	pathItem := m.spec.Paths[pathTemplate]

	method := strings.ToLower(obs.Method)
	op, exists := pathItem[method]
	if !exists {
		op = &Operation{
			Summary:     fmt.Sprintf("%s %s", strings.ToUpper(method), pathTemplate),
			OperationID: buildOperationID(method, pathTemplate),
			Responses:   map[string]Response{},
			Tags:        inferTags(pathTemplate),
		}
		// Extract path parameters
		op.Parameters = extractPathParams(pathTemplate)
		pathItem[method] = op
	}

	op.observationCount++

	// Merge query parameters
	for k, v := range obs.QueryParams {
		mergeQueryParam(op, k, v)
	}

	// Merge request body
	if len(obs.RequestBody) > 0 {
		schema, err := InferSchemaFromBytes(obs.RequestBody)
		if err == nil && schema != nil {
			ct := obs.ContentType
			if ct == "" {
				ct = "application/json"
			}
			ct = strings.Split(ct, ";")[0]
			if op.RequestBody == nil {
				op.RequestBody = &RequestBody{
					Required: true,
					Content:  map[string]MediaTypeEntry{ct: {Schema: schema}},
				}
			} else {
				existing := op.RequestBody.Content[ct]
				merged := MergeSchemas(existing.Schema, schema)
				op.RequestBody.Content[ct] = MediaTypeEntry{Schema: merged}
			}
		}
	}

	// Merge response
	statusKey := fmt.Sprintf("%d", obs.StatusCode)
	existing, hasStatus := op.Responses[statusKey]
	if !hasStatus {
		existing = Response{
			Description: http.StatusText(obs.StatusCode),
			Content:     map[string]MediaTypeEntry{},
		}
	}

	if len(obs.ResponseBody) > 0 {
		schema, err := InferSchemaFromBytes(obs.ResponseBody)
		if err == nil && schema != nil {
			ct := "application/json"
			prev := existing.Content[ct]
			merged := MergeSchemas(prev.Schema, schema)
			existing.Content[ct] = MediaTypeEntry{Schema: merged}
		}
	}
	op.Responses[statusKey] = existing
}

// Spec returns a snapshot of the current OpenAPI spec
func (m *SpecMerger) Spec() *OpenAPISpec {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.spec
}

// ---- helpers ----

func buildOperationID(method, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := method
	for _, p := range parts {
		if strings.HasPrefix(p, "{") {
			out += "By" + strings.Title(strings.Trim(p, "{}"))
		} else {
			out += strings.Title(p)
		}
	}
	return out
}

func extractPathParams(path string) []Parameter {
	params := []Parameter{}
	for _, seg := range strings.Split(path, "/") {
		if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			name := strings.Trim(seg, "{}")
			params = append(params, Parameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   &JSONSchemaType{Type: "string"},
			})
		}
	}
	return params
}

func mergeQueryParam(op *Operation, name, value string) {
	for i, p := range op.Parameters {
		if p.In == "query" && p.Name == name {
			// Already present; widen schema if needed
			inferred := InferSchema(value)
			op.Parameters[i].Schema = MergeSchemas(p.Schema, inferred)
			return
		}
	}
	op.Parameters = append(op.Parameters, Parameter{
		Name:     name,
		In:       "query",
		Required: false,
		Schema:   InferSchema(value),
	})
}

func inferTags(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) > 0 && parts[0] != "" {
		// strip leading "api" or version prefix
		tag := parts[0]
		if tag == "api" && len(parts) > 1 {
			tag = parts[1]
		}
		if strings.HasPrefix(tag, "v") && len(tag) <= 3 {
			if len(parts) > 1 {
				tag = parts[1]
			}
		}
		return []string{tag}
	}
	return []string{"default"}
}

// SortedPaths returns path keys sorted for stable output
func (s *OpenAPISpec) SortedPaths() []string {
	keys := make([]string, 0, len(s.Paths))
	for k := range s.Paths {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
