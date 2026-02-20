package storage

import (
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// taskIDPattern matches task-<YYYYMMDD>-<HHMMSS>-<slug> where slug is 3-50
// lowercase alphanumeric characters with internal hyphens allowed.
var taskIDPattern = regexp.MustCompile(`^task-\d{8}-\d{6}-[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

const randomSlugChars = "abcdefghijklmnopqrstuvwxyz0123456789"

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
