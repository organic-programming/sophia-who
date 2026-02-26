// Package cli implements the interactive command-line interface for Sophia Who?.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/organic-programming/sophia-who/pkg/identity"
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
func RunList(root string) error {
	if root == "" {
		root = "."
	}
	root = filepath.Clean(root)

	localSeen := map[string]struct{}{}
	printedHeader := false
	printedEntries := 0
	inlineProgress := isTerminal(os.Stderr)
	progressVisible := false

	clearProgressLine := func() {
		if !inlineProgress || !progressVisible {
			return
		}
		fmt.Fprint(os.Stderr, "\r\033[2K")
		progressVisible = false
	}

	printProgress := func(scanLabel string, scannedFiles int) {
		if !inlineProgress {
			fmt.Fprintf(os.Stderr, "[scan] %s: %d files scanned\n", scanLabel, scannedFiles)
			return
		}
		fmt.Fprintf(os.Stderr, "\r\033[2K[scan] %s: %d files scanned", scanLabel, scannedFiles)
		progressVisible = true
	}

	printEntry := func(id identity.Identity, origin, path string) {
		clearProgressLine()

		if !printedHeader {
			fmt.Printf("%-38s %-33s %-8s %-25s %-8s %s\n", "UUID", "NAME", "ORIGIN", "CLADE", "STATUS", "PATH")
			fmt.Println(strings.Repeat("─", 150))
			printedHeader = true
		}

		name := strings.TrimSpace(id.GivenName + " " + id.FamilyName)
		fmt.Printf("%-38s %-33s %-8s %-25s %-8s %s\n", id.UUID, name, origin, id.Clade, id.Status, path)
		printedEntries++
	}

	scanAndPrint := func(scanRoot, scanLabel, origin string, dedupe map[string]struct{}) {
		lastReported := 0
		err := identity.ScanAllWithPaths(scanRoot, 500, func(h identity.LocatedIdentity) {
			key := h.Identity.UUID
			if key == "" {
				key = h.Path
			}
			if dedupe != nil {
				if _, duplicate := dedupe[key]; duplicate {
					return
				}
				dedupe[key] = struct{}{}
			}

			printEntry(h.Identity, origin, relHolonDir(root, h.Path))
		}, func(progress identity.ScanProgress) {
			if progress.ScannedFiles == 0 || progress.ScannedFiles == lastReported {
				return
			}
			lastReported = progress.ScannedFiles
			printProgress(scanLabel, progress.ScannedFiles)
		})
		if err != nil {
			return
		}
	}

	// Local holons: <root>/holons/
	scanAndPrint(filepath.Join(root, "holons"), "local", "local", localSeen)

	// Also scan root itself for HOLON.md (standalone project)
	scanAndPrint(root, "root", "local", localSeen)

	// Cached holons: ~/.holon/cache/
	cacheDir := holonCacheDir()
	if cacheDir != "" {
		scanAndPrint(cacheDir, "cache", "cached", nil)
	}

	clearProgressLine()

	if printedEntries == 0 {
		fmt.Println("No holons found.")
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

func relHolonDir(root, holonPath string) string {
	dir := filepath.Dir(holonPath)
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return filepath.Clean(dir)
	}
	return filepath.Clean(rel)
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
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
