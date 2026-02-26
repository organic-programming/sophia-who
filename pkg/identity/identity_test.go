package identity

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	id := New()

	if id.UUID == "" {
		t.Fatal("UUID must not be empty")
	}
	// UUID v4 format: 8-4-4-4-12
	parts := strings.Split(id.UUID, "-")
	if len(parts) != 5 {
		t.Fatalf("UUID format invalid: %s", id.UUID)
	}

	if id.Status != "draft" {
		t.Errorf("default status = %q, want %q", id.Status, "draft")
	}
	if id.Born == "" {
		t.Fatal("Born must not be empty")
	}
	if id.GeneratedBy != "sophia-who" {
		t.Errorf("GeneratedBy = %q, want %q", id.GeneratedBy, "sophia-who")
	}
	if id.ProtoStatus != "draft" {
		t.Errorf("ProtoStatus = %q, want %q", id.ProtoStatus, "draft")
	}
	if id.Parents == nil {
		t.Error("Parents must be initialized (not nil)")
	}
}

func TestNewGeneratesUniqueUUIDs(t *testing.T) {
	a := New()
	b := New()
	if a.UUID == b.UUID {
		t.Errorf("two calls to New() produced the same UUID: %s", a.UUID)
	}
}
