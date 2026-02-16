// Package cli implements the interactive command-line interface for Sophia Who?.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/organic-programming/sophia-who/pkg/identity"

	"gopkg.in/yaml.v3"
)

// RunNew interactively creates a new holon identity.
func RunNew() error {
	scanner := bufio.NewScanner(os.Stdin)
	id := identity.New()

	fmt.Println("─── Sophia Who? — New Holon Identity ───")
	fmt.Printf("UUID: %s (generated)\n\n", id.UUID)

	id.FamilyName = ask(scanner, "Family name (the function — e.g. Transcriber, Prober)")
	id.GivenName = ask(scanner, "Given name (the character — e.g. Swift, Deep)")
	id.Composer = ask(scanner, "Composer (who is making this decision?)")
	id.Motto = ask(scanner, "Motto (the dessein in one sentence)")

	fmt.Println("\nClade (computational nature):")
	for i, c := range identity.Clades {
		fmt.Printf("  %d. %s\n", i+1, c)
	}
	id.Clade = askChoice(scanner, "Choose clade", identity.Clades)

	fmt.Println("\nReproduction mode:")
	for i, r := range identity.ReproductionModes {
		fmt.Printf("  %d. %s\n", i+1, r)
	}
	id.Reproduction = askChoice(scanner, "Choose reproduction mode", identity.ReproductionModes)

	id.Lang = askDefault(scanner, "Implementation language", "go")

	aliases := askDefault(scanner, "Aliases (comma-separated, or empty)", "")
	if aliases != "" {
		for _, a := range strings.Split(aliases, ",") {
			if trimmed := strings.TrimSpace(a); trimmed != "" {
				id.Aliases = append(id.Aliases, trimmed)
			}
		}
	}

	license := askDefault(scanner, "Wrapped binary license (e.g. MIT, GPL-3.0, or empty)", "")
	if license != "" {
		id.WrappedLicense = license
	}

	dirName := strings.ToLower(id.GivenName + "-" + strings.TrimSuffix(id.FamilyName, "?"))
	dirName = strings.ReplaceAll(dirName, " ", "-")
	outputDir := askDefault(scanner, "Output directory", filepath.Join(".holon", dirName))

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", outputDir, err)
	}

	outputPath := filepath.Join(outputDir, "HOLON.md")

	if err := identity.WriteHolonMD(id, outputPath); err != nil {
		return err
	}

	fmt.Printf("\n✓ Born: %s %s\n", id.GivenName, id.FamilyName)
	fmt.Printf("  UUID: %s\n", id.UUID)
	fmt.Printf("  File: %s\n", outputPath)

	return nil
}

// RunShow reads and displays a holon's identity by UUID.
func RunShow(target string) error {
	path, err := identity.FindByUUID(".", target)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}

	fmt.Println(string(data))
	return nil
}

// RunList scans both local holons and the global cache, labeling the origin
// of each so the actant knows what is local and what is a dependency.
func RunList() error {
	type entry struct {
		id     identity.Identity
		origin string
	}
	var entries []entry

	// Local holons: project/holons/
	localHolons, err := identity.FindAll("holons")
	if err == nil {
		for _, h := range localHolons {
			entries = append(entries, entry{id: h, origin: "local"})
		}
	}

	// Also scan current directory root for HOLON.md (standalone project)
	rootHolons, err := identity.FindAll(".")
	if err == nil {
		for _, h := range rootHolons {
			// Avoid duplicates from the holons/ scan
			duplicate := false
			for _, e := range entries {
				if e.id.UUID == h.UUID {
					duplicate = true
					break
				}
			}
			if !duplicate {
				entries = append(entries, entry{id: h, origin: "local"})
			}
		}
	}

	// Cached holons: ~/.holon/cache/
	cacheDir := holonCacheDir()
	if cacheDir != "" {
		cachedHolons, err := identity.FindAll(cacheDir)
		if err == nil {
			for _, h := range cachedHolons {
				entries = append(entries, entry{id: h, origin: "cached"})
			}
		}
	}

	if len(entries) == 0 {
		fmt.Println("No holons found.")
		return nil
	}

	fmt.Printf("%-38s %-20s %-8s %-25s %s\n", "UUID", "NAME", "ORIGIN", "CLADE", "STATUS")
	fmt.Println(strings.Repeat("─", 105))

	for _, e := range entries {
		name := e.id.GivenName + " " + e.id.FamilyName
		fmt.Printf("%-38s %-20s %-8s %-25s %s\n", e.id.UUID, name, e.origin, e.id.Clade, e.id.Status)
	}

	return nil
}

// holonCacheDir returns the global holon cache directory (~/.holon/cache/).
// Returns an empty string if the home directory cannot be determined.
func holonCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".holon", "cache")
}

// RunPin captures version, OS, and architecture information for a holon's binary.
func RunPin(target string) error {
	path, err := identity.FindByUUID(".", target)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", path, err)
	}

	id, body, err := identity.ParseFrontmatter(data)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("─── Pin version for %s %s ───\n\n", id.GivenName, id.FamilyName)

	id.BinaryPath = askDefault(scanner, "Binary path", id.BinaryPath)
	id.BinaryVersion = askDefault(scanner, "Binary version", id.BinaryVersion)
	id.GitTag = askDefault(scanner, "Git tag (or empty)", id.GitTag)
	id.GitCommit = askDefault(scanner, "Git commit (or empty)", id.GitCommit)
	id.OS = askDefault(scanner, "OS", id.OS)
	id.Arch = askDefault(scanner, "Arch", id.Arch)

	yamlData, err := yaml.Marshal(id)
	if err != nil {
		return fmt.Errorf("yaml marshal error: %w", err)
	}

	output := "---\n# Holon Identity v1\n" + string(yamlData) + "---\n" + body

	if err := os.WriteFile(path, []byte(output), 0644); err != nil {
		return fmt.Errorf("cannot write %s: %w", path, err)
	}

	fmt.Printf("\n✓ Pinned: %s %s\n", id.GivenName, id.FamilyName)
	return nil
}

func ask(scanner *bufio.Scanner, prompt string) string {
	for {
		fmt.Printf("%s: ", prompt)
		scanner.Scan()
		answer := strings.TrimSpace(scanner.Text())
		if answer != "" {
			return answer
		}
		fmt.Println("  (required)")
	}
}

func askDefault(scanner *bufio.Scanner, prompt, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		fmt.Printf("%s: ", prompt)
	}
	scanner.Scan()
	answer := strings.TrimSpace(scanner.Text())
	if answer == "" {
		return defaultVal
	}
	return answer
}

func askChoice(scanner *bufio.Scanner, prompt string, choices []string) string {
	for {
		fmt.Printf("%s (1-%d): ", prompt, len(choices))
		scanner.Scan()
		answer := strings.TrimSpace(scanner.Text())
		for i, c := range choices {
			if answer == fmt.Sprintf("%d", i+1) || answer == c {
				return c
			}
		}
		fmt.Println("  (invalid choice)")
	}
}
