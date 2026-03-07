package identity

import (
	"os"
	"path/filepath"
	"testing"
)

const validManifest = `schema: "holon/v0"
uuid: "test-uuid-1234"
given_name: "TestHolon"
family_name: "Prober"
motto: "Test all the things."
composer: "B. ALTER"
clade: "deterministic/pure"
status: draft
born: "2026-01-01"
parents: []
reproduction: "manual"
aliases: ["testholon"]
generated_by: "test"
lang: "go"
proto_status: draft
description: |
  Test holon description.
`

func TestParseHolonYAML(t *testing.T) {
	id, err := ParseHolonYAML([]byte(validManifest))
	if err != nil {
		t.Fatalf("ParseHolonYAML failed: %v", err)
	}
	if id.UUID != "test-uuid-1234" {
		t.Errorf("UUID = %q, want %q", id.UUID, "test-uuid-1234")
	}
	if id.GivenName != "TestHolon" {
		t.Errorf("GivenName = %q, want %q", id.GivenName, "TestHolon")
	}
	if id.FamilyName != "Prober" {
		t.Errorf("FamilyName = %q, want %q", id.FamilyName, "Prober")
	}
	if id.Clade != "deterministic/pure" {
		t.Errorf("Clade = %q, want %q", id.Clade, "deterministic/pure")
	}
	if id.Status != "draft" {
		t.Errorf("Status = %q, want %q", id.Status, "draft")
	}
	if id.Description == "" {
		t.Error("description must not be empty")
	}
}

func TestParseHolonYAMLInvalidYAML(t *testing.T) {
	_, err := ParseHolonYAML([]byte(":\n  - broken"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

// setupTestDir creates a temporary directory tree with holon.yaml files
// for testing FindAll and FindByUUID.
func setupTestDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	for _, h := range []struct {
		dir, uuid, name string
	}{
		{"holon-a", "aaaa-1111", "Alpha"},
		{"holon-b", "bbbb-2222", "Beta"},
	} {
		dir := filepath.Join(root, h.dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		content := `schema: "holon/v0"
uuid: "` + h.uuid + `"
given_name: "` + h.name + `"
family_name: "Test"
status: draft
`
		if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	hidden := filepath.Join(root, ".secret")
	if err := os.MkdirAll(hidden, 0755); err != nil {
		t.Fatal(err)
	}
	hiddenContent := `schema: "holon/v0"
uuid: "hidden-uuid"
given_name: "Hidden"
family_name: "Test"
status: draft
`
	if err := os.WriteFile(filepath.Join(hidden, ManifestFileName), []byte(hiddenContent), 0644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestFindAll(t *testing.T) {
	root := setupTestDir(t)

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 2 {
		t.Fatalf("FindAll found %d holons, want 2", len(holons))
	}

	uuids := map[string]bool{}
	for _, h := range holons {
		uuids[h.UUID] = true
	}
	if !uuids["aaaa-1111"] {
		t.Error("FindAll did not find holon-a")
	}
	if !uuids["bbbb-2222"] {
		t.Error("FindAll did not find holon-b")
	}
	if uuids["hidden-uuid"] {
		t.Error("FindAll should skip hidden directories")
	}
}

func TestFindAllEmptyDir(t *testing.T) {
	root := t.TempDir()

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 0 {
		t.Errorf("FindAll found %d holons in empty dir, want 0", len(holons))
	}
}

func TestFindByUUIDExact(t *testing.T) {
	root := setupTestDir(t)

	path, err := FindByUUID(root, "aaaa-1111")
	if err != nil {
		t.Fatalf("FindByUUID failed: %v", err)
	}
	if path == "" {
		t.Fatal("FindByUUID returned empty path")
	}
}

func TestFindByUUIDPrefix(t *testing.T) {
	root := setupTestDir(t)

	path, err := FindByUUID(root, "bbbb")
	if err != nil {
		t.Fatalf("FindByUUID prefix failed: %v", err)
	}
	if path == "" {
		t.Fatal("FindByUUID prefix returned empty path")
	}
}

func TestFindByUUIDNotFound(t *testing.T) {
	root := setupTestDir(t)

	_, err := FindByUUID(root, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent UUID")
	}
}

func TestFindAllSkipsUnparseableFiles(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "bad-holon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(":\n  - broken"), 0644); err != nil {
		t.Fatal(err)
	}

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 0 {
		t.Errorf("FindAll found %d holons, want 0 (unparseable should be skipped)", len(holons))
	}
}

func TestFindAllSkipsUnreadableFiles(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "unreadable")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Skip("cannot change file permissions on this OS")
	}
	defer os.Chmod(path, 0644) //nolint:errcheck

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 0 {
		t.Errorf("FindAll found %d holons, want 0 (unreadable should be skipped)", len(holons))
	}
}

func TestFindByUUIDSkipsUnreadable(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "unreadable")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, ManifestFileName)
	if err := os.WriteFile(path, []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Skip("cannot change file permissions on this OS")
	}
	defer os.Chmod(path, 0644) //nolint:errcheck

	_, err := FindByUUID(root, "test-uuid-1234")
	if err == nil {
		t.Fatal("expected error when file is unreadable")
	}
}

func TestFindByUUIDSkipsUnparseable(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "bad")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(":\n  - broken"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := FindByUUID(root, "anything")
	if err == nil {
		t.Fatal("expected error for UUID not found after skipping unparseable")
	}
}

func TestFindAllNonexistentRoot(t *testing.T) {
	_, err := FindAll("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Logf("FindAll on nonexistent root returned expected error: %v", err)
	}
}

func TestFindAllIgnoresNonHolonFiles(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, "mixed")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 1 {
		t.Errorf("FindAll found %d holons, want 1 (should ignore non-holon.yaml files)", len(holons))
	}
}

func TestFindAllSkipsDotHolonDir(t *testing.T) {
	root := t.TempDir()

	dir := filepath.Join(root, ".holon", "some-tool")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ManifestFileName), []byte(validManifest), 0644); err != nil {
		t.Fatal(err)
	}

	holons, err := FindAll(root)
	if err != nil {
		t.Fatalf("FindAll failed: %v", err)
	}
	if len(holons) != 0 {
		t.Errorf("FindAll found %d holons, want 0 (.holon/ should be skipped)", len(holons))
	}
}

func TestScanAllWithPathsStreamsFoundAndProgress(t *testing.T) {
	root := setupTestDir(t)

	var found []string
	var progress []ScanProgress

	err := ScanAllWithPaths(root, 1, func(h LocatedIdentity) {
		found = append(found, h.Identity.UUID)
	}, func(p ScanProgress) {
		progress = append(progress, p)
	})
	if err != nil {
		t.Fatalf("ScanAllWithPaths failed: %v", err)
	}

	if len(found) != 2 {
		t.Fatalf("ScanAllWithPaths found %d holons, want 2", len(found))
	}

	uuids := map[string]bool{}
	for _, uuid := range found {
		uuids[uuid] = true
	}
	if !uuids["aaaa-1111"] || !uuids["bbbb-2222"] {
		t.Fatalf("ScanAllWithPaths returned unexpected UUIDs: %#v", found)
	}
	if uuids["hidden-uuid"] {
		t.Fatal("ScanAllWithPaths should skip hidden directories")
	}

	if len(progress) == 0 {
		t.Fatal("ScanAllWithPaths should emit progress updates")
	}

	last := progress[len(progress)-1]
	if last.HolonsFound != 2 {
		t.Fatalf("last progress holons found = %d, want 2", last.HolonsFound)
	}
	if last.ScannedFiles == 0 {
		t.Fatal("last progress scanned files should be > 0")
	}
}
