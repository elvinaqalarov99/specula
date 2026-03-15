package inference

import (
	"testing"
)

func TestNormalizerCollapseNumericIDs(t *testing.T) {
	n := NewPathNormalizer()
	n.Observe("/users/1")
	n.Observe("/users/42")
	n.Observe("/users/99")
	got := n.Observe("/users/7")
	if got != "/users/{id}" {
		t.Errorf("expected /users/{id}, got %s", got)
	}
}

func TestNormalizerCollapseUUIDs(t *testing.T) {
	n := NewPathNormalizer()
	n.Observe("/orders/550e8400-e29b-41d4-a716-446655440000")
	n.Observe("/orders/6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	got := n.Observe("/orders/6ba7b814-9dad-11d1-80b4-00c04fd430c8")
	if got != "/orders/{id}" {
		t.Errorf("expected /orders/{id}, got %s", got)
	}
}

func TestNormalizerLiteralPreserved(t *testing.T) {
	n := NewPathNormalizer()
	n.Observe("/users/me")
	got := n.Observe("/users/me")
	if got != "/users/me" {
		t.Errorf("expected /users/me, got %s", got)
	}
}

func TestNormalizerNestedParam(t *testing.T) {
	n := NewPathNormalizer()
	n.Observe("/users/1/posts/10")
	n.Observe("/users/2/posts/20")
	n.Observe("/users/3/posts/30")
	got := n.Observe("/users/4/posts/40")
	if got != "/users/{id}/posts/{id}" {
		t.Errorf("expected /users/{id}/posts/{id}, got %s", got)
	}
}
