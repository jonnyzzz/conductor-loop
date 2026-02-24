package storage

import "testing"

// TestStatusConstants verifies the new task-level status constants are defined
// and have the correct canonical string values required by the API contract.
func TestStatusConstants(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{"StatusRunning", StatusRunning, "running"},
		{"StatusCompleted", StatusCompleted, "completed"},
		{"StatusFailed", StatusFailed, "failed"},
		{"StatusUnknown", StatusUnknown, "unknown"},
		{"StatusAllFinished", StatusAllFinished, "all_finished"},
		{"StatusQueued", StatusQueued, "queued"},
		{"StatusBlocked", StatusBlocked, "blocked"},
		{"StatusPartialFail", StatusPartialFail, "partial_failure"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.value != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.value, tc.want)
			}
		})
	}
}
