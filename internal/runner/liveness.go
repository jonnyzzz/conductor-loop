package runner

// IsProcessGroupAlive reports whether a process group is alive.
func IsProcessGroupAlive(pgid int) (bool, error) {
	return isProcessGroupAlive(pgid)
}
