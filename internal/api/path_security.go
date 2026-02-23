package api

import (
	"fmt"
	"path/filepath"
	"strings"
)

func joinPathWithinRoot(rootDir string, segments ...string) (string, *apiError) {
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanRoot == "" || cleanRoot == "." {
		return "", apiErrorInternal("configured root is empty", nil)
	}

	all := make([]string, 0, len(segments)+1)
	all = append(all, cleanRoot)
	all = append(all, segments...)
	target := filepath.Clean(filepath.Join(all...))
	if err := requirePathWithinRoot(cleanRoot, target, "target path"); err != nil {
		return "", err
	}

	return target, nil
}

func requirePathWithinRoot(rootDir, targetPath, field string) *apiError {
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	cleanTarget := filepath.Clean(strings.TrimSpace(targetPath))
	if cleanRoot == "" || cleanRoot == "." {
		return apiErrorInternal("configured root is empty", nil)
	}
	if cleanTarget == "" {
		return apiErrorBadRequest(fmt.Sprintf("%s is empty", field))
	}
	if !pathWithinRoot(cleanTarget, cleanRoot) {
		return apiErrorForbidden(fmt.Sprintf("%s escapes configured root", field))
	}
	return nil
}
