// Package facts provides functionality to promote FACT messages from
// task-level message buses into a project-level PROJECT-FACTS.md file.
package facts

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

const (
	projectFactsFileName = "PROJECT-FACTS.md"
	factFileMode         = 0o644
)

// PromoteConfig holds configuration for the PromoteFacts operation.
type PromoteConfig struct {
	// RootDir is the root directory containing project subdirectories.
	RootDir string
	// ProjectID is the project to scan.
	ProjectID string
	// DryRun when true does not write the PROJECT-FACTS.md file.
	DryRun bool
	// FilterType is the message type to promote (default: "FACT").
	FilterType string
	// Since is an optional time filter; only messages after this time are promoted.
	Since time.Time
}

// PromoteResult holds a successfully promoted fact entry.
type PromoteResult struct {
	TaskID    string
	Body      string
	Timestamp time.Time
}

// PromoteFacts scans all task message buses in a project and promotes
// messages of the configured type into a project-level PROJECT-FACTS.md.
// Returns the number of newly promoted facts and the number already present.
func PromoteFacts(cfg PromoteConfig) (promoted int, already int, err error) {
	if strings.TrimSpace(cfg.RootDir) == "" {
		return 0, 0, fmt.Errorf("root dir is required")
	}
	if strings.TrimSpace(cfg.ProjectID) == "" {
		return 0, 0, fmt.Errorf("project id is required")
	}

	filterType := strings.TrimSpace(cfg.FilterType)
	if filterType == "" {
		filterType = "FACT"
	}

	projectDir := filepath.Join(cfg.RootDir, cfg.ProjectID)

	// Load existing promoted hashes from PROJECT-FACTS.md.
	factsPath := filepath.Join(projectDir, projectFactsFileName)
	existingHashes, err := loadExistingHashes(factsPath)
	if err != nil {
		return 0, 0, fmt.Errorf("load existing facts: %w", err)
	}

	// Scan all task directories under the project.
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, nil
		}
		return 0, 0, fmt.Errorf("read project dir %q: %w", projectDir, err)
	}

	// Collect new facts, deduplicating across tasks.
	newFacts := make([]PromoteResult, 0)
	seenInSession := make(map[string]bool)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskID := entry.Name()
		busPath := filepath.Join(projectDir, taskID, "TASK-MESSAGE-BUS.md")
		if _, statErr := os.Stat(busPath); os.IsNotExist(statErr) {
			continue
		}

		bus, busErr := messagebus.NewMessageBus(busPath)
		if busErr != nil {
			continue
		}

		messages, readErr := bus.ReadMessages("")
		if readErr != nil {
			continue
		}

		for _, msg := range messages {
			if !strings.EqualFold(msg.Type, filterType) {
				continue
			}
			if !cfg.Since.IsZero() && msg.Timestamp.Before(cfg.Since) {
				continue
			}

			body := strings.TrimSpace(msg.Body)
			if body == "" {
				continue
			}

			hash := hashBody(body)

			if existingHashes[hash] {
				already++
				continue
			}
			if seenInSession[hash] {
				continue
			}
			seenInSession[hash] = true

			newFacts = append(newFacts, PromoteResult{
				TaskID:    taskID,
				Body:      body,
				Timestamp: msg.Timestamp,
			})
		}
	}

	if cfg.DryRun || len(newFacts) == 0 {
		promoted = len(newFacts)
		return promoted, already, nil
	}

	// Read existing content to append.
	existingContent, readErr := os.ReadFile(factsPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return 0, 0, fmt.Errorf("read existing facts file: %w", readErr)
	}

	now := time.Now().UTC()
	var sb strings.Builder

	if len(existingContent) == 0 {
		// Write header for new file.
		sb.WriteString(fmt.Sprintf("# Project Facts: %s\n", cfg.ProjectID))
		sb.WriteString(fmt.Sprintf("Last updated: %s\n", now.Format(time.RFC3339)))
		sb.WriteString("\n")
	} else {
		// Update the "Last updated" line.
		updated := updateLastUpdated(string(existingContent), now)
		sb.WriteString(updated)
	}

	for _, fact := range newFacts {
		title := fact.Body
		if len(title) > 80 {
			title = title[:80]
		}
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("## Fact: %s\n", title))
		sb.WriteString(fmt.Sprintf("- **Body**: %s\n", fact.Body))
		sb.WriteString(fmt.Sprintf("- **Source task**: %s\n", fact.TaskID))
		sb.WriteString(fmt.Sprintf("- **Original timestamp**: %s\n", fact.Timestamp.UTC().Format(time.RFC3339)))
		sb.WriteString(fmt.Sprintf("- **Promoted at**: %s\n", now.Format(time.RFC3339)))
		sb.WriteString("---\n")
	}

	if err := writeFileAtomic(factsPath, []byte(sb.String())); err != nil {
		return 0, 0, fmt.Errorf("write facts file: %w", err)
	}

	promoted = len(newFacts)
	return promoted, already, nil
}

// hashBody returns a stable hex hash for deduplication of fact bodies.
func hashBody(body string) string {
	h := sha256.Sum256([]byte(body))
	return fmt.Sprintf("%x", h)
}

// loadExistingHashes parses an existing PROJECT-FACTS.md and returns a set of
// body hashes for facts already present in the file.
func loadExistingHashes(path string) (map[string]bool, error) {
	hashes := make(map[string]bool)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hashes, nil
		}
		return nil, err
	}

	// Scan for "- **Body**: <content>" lines.
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	const bodyPrefix = "- **Body**: "
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, bodyPrefix) {
			body := strings.TrimSpace(strings.TrimPrefix(line, bodyPrefix))
			if body != "" {
				hashes[hashBody(body)] = true
			}
		}
	}
	return hashes, scanner.Err()
}

// updateLastUpdated replaces the "Last updated: ..." line in content with a fresh timestamp.
func updateLastUpdated(content string, t time.Time) string {
	const prefix = "Last updated: "
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			lines[i] = prefix + t.Format(time.RFC3339)
			return strings.Join(lines, "\n")
		}
	}
	return content
}

// writeFileAtomic writes data to path using a temp-file + rename pattern.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, "facts.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	success := false
	defer func() {
		if !success {
			_ = tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("fsync temp file: %w", err)
	}
	if err := tmpFile.Chmod(factFileMode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	success = true
	return nil
}
