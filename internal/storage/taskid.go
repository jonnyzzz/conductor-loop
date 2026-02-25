package storage

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

// taskIDPattern matches task-<YYYYMMDD>-<HHMMSS>-<slug> where slug is 3-50
// lowercase alphanumeric characters with internal hyphens allowed.
var taskIDPattern = regexp.MustCompile(`^task-\d{8}-\d{6}-[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

const randomSlugChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// projectIDPattern matches a valid project ID: one or more lowercase
// alphanumeric characters or hyphens, no path separators or dots.
var projectIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// ValidateProjectID returns an error if the project ID is not a safe, flat
// directory name. Project IDs must not contain path separators, preventing a
// project from being nested under another project folder.
func ValidateProjectID(projectID string) error {
	if strings.TrimSpace(projectID) == "" {
		return fmt.Errorf("project ID is empty")
	}
	if strings.ContainsAny(projectID, "/\\") {
		return fmt.Errorf("invalid project ID %q: must not contain path separators (projects cannot be nested)", projectID)
	}
	if projectID == "." || projectID == ".." {
		return fmt.Errorf("invalid project ID %q: must not be a relative path reference", projectID)
	}
	if strings.Contains(projectID, "\x00") {
		return fmt.Errorf("invalid project ID %q: must not contain null bytes", projectID)
	}
	if !projectIDPattern.MatchString(projectID) {
		return fmt.Errorf("invalid project ID %q: must contain only lowercase letters, digits, and hyphens (e.g. my-project)", projectID)
	}
	return nil
}

// ValidateTaskID returns an error if the task ID does not match the required
// format task-<YYYYMMDD>-<HHMMSS>-<slug>.
func ValidateTaskID(taskID string) error {
	if !taskIDPattern.MatchString(taskID) {
		return fmt.Errorf("invalid task ID %q: must match task-<YYYYMMDD>-<HHMMSS>-<slug> (e.g. task-20260220-153045-my-feature)", taskID)
	}
	return nil
}

// GenerateTaskID generates a task ID using the current time and the given slug.
// If slug is empty, a 6-character random alphanumeric slug is generated.
func GenerateTaskID(slug string) string {
	return generateTaskIDAt(time.Now(), slug)
}

func generateTaskIDAt(now time.Time, slug string) string {
	ts := now.UTC().Format("20060102-150405")
	if slug == "" {
		slug = randomSlug(6)
	}
	return fmt.Sprintf("task-%s-%s", ts, slug)
}

func randomSlug(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = randomSlugChars[rand.Intn(len(randomSlugChars))]
	}
	return string(b)
}
