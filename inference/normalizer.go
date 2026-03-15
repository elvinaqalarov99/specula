package inference

import (
	"regexp"
	"strings"
	"sync"
)

var (
	reNumeric = regexp.MustCompile(`^\d+$`)
	reUUIDSeg = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	reSlug    = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{8,}[a-z0-9])?$`) // long slug-like IDs
)

// trieNode builds a prefix-trie of path segments to detect parameter positions
type trieNode struct {
	mu       sync.RWMutex
	children map[string]*trieNode
	count    int // how many distinct values appeared at this position
	isParam  bool
}

// PathNormalizer accumulates path observations and produces normalized templates
type PathNormalizer struct {
	mu   sync.Mutex
	root *trieNode
}

func NewPathNormalizer() *PathNormalizer {
	return &PathNormalizer{root: &trieNode{children: map[string]*trieNode{}}}
}

// Observe records a raw URL path and returns the normalized template, e.g. /users/{id}
func (n *PathNormalizer) Observe(rawPath string) string {
	rawPath = strings.Split(rawPath, "?")[0] // strip query string
	segments := splitPath(rawPath)

	n.mu.Lock()
	insertTrie(n.root, segments)
	n.mu.Unlock()

	n.mu.Lock()
	markParams(n.root)
	n.mu.Unlock()

	n.mu.Lock()
	template := buildTemplate(n.root, segments)
	n.mu.Unlock()

	return "/" + strings.Join(template, "/")
}

// NormalizedPaths returns all distinct path templates seen so far
func (n *PathNormalizer) NormalizedPaths() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	markParams(n.root)
	var paths []string
	collectPaths(n.root, nil, &paths)
	return paths
}

// ---- trie helpers ----

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return []string{}
	}
	return strings.Split(p, "/")
}

func insertTrie(node *trieNode, segments []string) {
	if len(segments) == 0 {
		node.count++
		return
	}
	seg := segments[0]
	child, ok := node.children[seg]
	if !ok {
		child = &trieNode{children: map[string]*trieNode{}}
		node.children[seg] = child
	}
	insertTrie(child, segments[1:])
}

// markParams scans siblings: if a node has ≥2 children that look like IDs,
// collapse them into a single "{param}" node.
func markParams(node *trieNode) {
	if len(node.children) == 0 {
		return
	}

	// First recurse
	for _, child := range node.children {
		markParams(child)
	}

	// Collect candidate siblings (exclude already-parameterised ones)
	paramCandidates := []string{}
	for seg := range node.children {
		if seg != "{id}" && looksLikeID(seg) {
			paramCandidates = append(paramCandidates, seg)
		}
	}

	if len(paramCandidates) >= 2 {
		// Merge all candidates into a single {id} node
		merged := &trieNode{children: map[string]*trieNode{}}
		for _, seg := range paramCandidates {
			child := node.children[seg]
			mergeTrieNodes(merged, child)
			delete(node.children, seg)
		}
		// If {id} already exists, merge into it; otherwise create
		if existing, ok := node.children["{id}"]; ok {
			mergeTrieNodes(existing, merged)
		} else {
			node.children["{id}"] = merged
		}
	}
}

func mergeTrieNodes(dst, src *trieNode) {
	dst.count += src.count
	for k, v := range src.children {
		if existing, ok := dst.children[k]; ok {
			mergeTrieNodes(existing, v)
		} else {
			dst.children[k] = v
		}
	}
}

func looksLikeID(s string) bool {
	return reNumeric.MatchString(s) || reUUIDSeg.MatchString(s) || reSlug.MatchString(s)
}

func buildTemplate(node *trieNode, segments []string) []string {
	if len(segments) == 0 {
		return []string{}
	}
	seg := segments[0]
	rest := segments[1:]

	// Check if this segment was collapsed into {id}
	if _, ok := node.children["{id}"]; ok {
		if looksLikeID(seg) {
			return append([]string{"{id}"}, buildTemplate(node.children["{id}"], rest)...)
		}
	}
	if child, ok := node.children[seg]; ok {
		return append([]string{seg}, buildTemplate(child, rest)...)
	}
	// Fallback: literal
	return append([]string{seg}, rest...)
}

func collectPaths(node *trieNode, prefix []string, out *[]string) {
	if node.count > 0 || len(node.children) == 0 {
		path := "/" + strings.Join(prefix, "/")
		*out = append(*out, path)
	}
	for seg, child := range node.children {
		collectPaths(child, append(prefix, seg), out)
	}
}
