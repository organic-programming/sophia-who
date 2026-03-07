package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteHolonYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestFileName)

	id := New()
	id.GivenName = "WriteTest"
	id.FamilyName = "Holon"
	id.Motto = "Writing is believing."
	id.Composer = "Test Suite"
	id.Clade = "deterministic/pure"
	id.Lang = "go"

	if err := WriteHolonYAML(id, path); err != nil {
		t.Fatalf("WriteHolonYAML failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("file is empty")
	}
}

func TestWriteHolonYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ManifestFileName)

	original := New()
	original.GivenName = "RoundTrip"
	original.FamilyName = "Tester"
	original.Motto = "What goes out must come back."
	original.Composer = "Test Suite"
	original.Clade = "probabilistic/generative"
	original.Reproduction = "assisted"
	original.Lang = "go"
	original.Aliases = []string{"rt", "round"}

	if err := WriteHolonYAML(original, path); err != nil {
		t.Fatalf("WriteHolonYAML failed: %v", err)
	}

	parsed, raw, err := ReadHolonYAML(path)
	if err != nil {
		t.Fatalf("ReadHolonYAML failed on written file: %v", err)
	}

	if parsed.UUID != original.UUID {
		t.Errorf("UUID: got %q, want %q", parsed.UUID, original.UUID)
	}
	if parsed.GivenName != original.GivenName {
		t.Errorf("GivenName: got %q, want %q", parsed.GivenName, original.GivenName)
	}
	if parsed.FamilyName != original.FamilyName {
		t.Errorf("FamilyName: got %q, want %q", parsed.FamilyName, original.FamilyName)
	}
	if parsed.Motto != original.Motto {
		t.Errorf("Motto: got %q, want %q", parsed.Motto, original.Motto)
	}
	if parsed.Composer != original.Composer {
		t.Errorf("Composer: got %q, want %q", parsed.Composer, original.Composer)
	}
	if parsed.Clade != original.Clade {
		t.Errorf("Clade: got %q, want %q", parsed.Clade, original.Clade)
	}
	if parsed.Reproduction != original.Reproduction {
		t.Errorf("Reproduction: got %q, want %q", parsed.Reproduction, original.Reproduction)
	}
	if parsed.Lang != original.Lang {
		t.Errorf("Lang: got %q, want %q", parsed.Lang, original.Lang)
	}
	if len(parsed.Aliases) != len(original.Aliases) {
		t.Errorf("Aliases count: got %d, want %d", len(parsed.Aliases), len(original.Aliases))
	}
	if len(raw) == 0 {
		t.Error("raw file content must not be empty")
	}
}

func TestWriteHolonYAMLInvalidPath(t *testing.T) {
	id := New()
	id.GivenName = "Bad"
	id.FamilyName = "Path"

	err := WriteHolonYAML(id, "/nonexistent/dir/holon.yaml")
	if err == nil {
		t.Fatal("expected error writing to invalid path")
	}
}
