package storage

import "strings"

const (
	// ProcessOwnershipManaged means conductor started and owns the process lifecycle.
	ProcessOwnershipManaged = "managed"
	// ProcessOwnershipExternal means conductor is only observing an externally owned process.
	ProcessOwnershipExternal = "external"
)

// NormalizeProcessOwnership normalizes persisted ownership values.
// Empty values default to managed for backward compatibility.
func NormalizeProcessOwnership(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", ProcessOwnershipManaged:
		return ProcessOwnershipManaged
	case ProcessOwnershipExternal:
		return ProcessOwnershipExternal
	default:
		return ProcessOwnershipManaged
	}
}

// EffectiveProcessOwnership returns the normalized ownership for a run-info record.
func EffectiveProcessOwnership(info *RunInfo) string {
	if info == nil {
		return ProcessOwnershipManaged
	}
	return NormalizeProcessOwnership(info.ProcessOwnership)
}

// CanTerminateProcess reports whether stop commands should signal the recorded process.
func CanTerminateProcess(info *RunInfo) bool {
	return EffectiveProcessOwnership(info) == ProcessOwnershipManaged
}
