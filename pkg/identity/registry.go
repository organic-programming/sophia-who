package identity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocatedIdentity pairs a parsed Identity with the HOLON.md path it came from.
type LocatedIdentity struct {
	Identity Identity
	Path     string
}

// FindAll scans the directory tree from root for HOLON.md files
// and returns the parsed identities.
func FindAll(root string) ([]Identity, error) {
	located, err := FindAllWithPaths(root)
	if err != nil {
		return nil, err
	}

	holons := make([]Identity, 0, len(located))
	for _, h := range located {
		holons = append(holons, h.Identity)
	}

	return holons, nil
}

// FindAllWithPaths scans the directory tree from root for HOLON.md files
// and returns parsed identities with source file paths.
func FindAllWithPaths(root string) ([]LocatedIdentity, error) {
	var holons []LocatedIdentity

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name != "." && name != ".holon" && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "HOLON.md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		id, _, err := ParseFrontmatter(data)
		if err != nil {
			return nil
		}

		holons = append(holons, LocatedIdentity{
			Identity: id,
			Path:     path,
		})
		return nil
	})

	return holons, err
}

// FindByUUID locates a HOLON.md file by full UUID or prefix.
func FindByUUID(root, target string) (string, error) {
	var found string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "HOLON.md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		id, _, err := ParseFrontmatter(data)
		if err != nil {
			return nil
		}

		if id.UUID == target || strings.HasPrefix(id.UUID, target) {
			found = path
			return filepath.SkipAll
		}

		return nil
	})

	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("holon not found: %s", target)
	}
	return found, nil
}

// ParseFrontmatter extracts the YAML frontmatter and the remaining
// markdown body from a HOLON.md file.
func ParseFrontmatter(data []byte) (Identity, string, error) {
	content := string(data)

	if !strings.HasPrefix(content, "---") {
		return Identity{}, "", fmt.Errorf("no YAML frontmatter found")
	}

	rest := content[3:]
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	}

	end := strings.Index(rest, "\n---")
	if end < 0 {
		return Identity{}, "", fmt.Errorf("unclosed YAML frontmatter")
	}

	yamlBlock := rest[:end]
	body := rest[end+4:]

	var id Identity
	if err := yaml.Unmarshal([]byte(yamlBlock), &id); err != nil {
		return Identity{}, "", fmt.Errorf("YAML parse error: %w", err)
	}

	return id, body, nil
}
