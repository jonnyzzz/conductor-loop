package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// HomeFolders holds the project directory linkage stored in home-folders.md.
// Format: YAML with project_root, source_folders[], additional_folders[].
type HomeFolders struct {
	ProjectRoot       string        `yaml:"project_root"`
	SourceFolders     []FolderEntry `yaml:"source_folders,omitempty"`
	AdditionalFolders []FolderEntry `yaml:"additional_folders,omitempty"`
}

// FolderEntry is a folder path with an optional human-readable description.
type FolderEntry struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description,omitempty"`
}

const homeFoldersFileName = "home-folders.md"

// HomeFoldersPath returns the canonical path for home-folders.md under the given
// runs root and project ID.
func HomeFoldersPath(runsRoot, projectID string) string {
	return filepath.Join(runsRoot, projectID, homeFoldersFileName)
}

// ReadHomeFolders reads and parses home-folders.md from path.
func ReadHomeFolders(path string) (*HomeFolders, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read home-folders")
	}
	var hf HomeFolders
	if err := yaml.Unmarshal(data, &hf); err != nil {
		return nil, errors.Wrap(err, "parse home-folders")
	}
	return &hf, nil
}

// WriteHomeFolders atomically writes hf to path.
// The parent directory is created with 0755 permissions if it does not exist.
func WriteHomeFolders(path string, hf *HomeFolders) error {
	if hf == nil {
		return errors.New("home-folders is nil")
	}
	data, err := yaml.Marshal(hf)
	if err != nil {
		return errors.Wrap(err, "marshal home-folders")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return errors.Wrap(err, "create project dir")
	}
	return writeFileAtomic(path, data)
}

// FindProjectByDir scans all project directories under runsRoot and returns
// the first project ID whose home-folders.md records projectDir as project_root.
// Returns empty string if no match is found.
func FindProjectByDir(runsRoot, projectDir string) string {
	entries, err := os.ReadDir(runsRoot)
	if err != nil {
		return ""
	}
	target := filepath.Clean(strings.TrimSpace(projectDir))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if ValidateProjectID(e.Name()) != nil {
			continue
		}
		hfPath := filepath.Join(runsRoot, e.Name(), homeFoldersFileName)
		hf, err := ReadHomeFolders(hfPath)
		if err != nil {
			continue
		}
		root := filepath.Clean(strings.TrimSpace(hf.ProjectRoot))
		if root == target {
			return e.Name()
		}
	}
	return ""
}

// InitProject creates the canonical project layout under runsRoot:
//   - <runsRoot>/<projectID>/home-folders.md  (overwritten if present)
//   - <runsRoot>/<projectID>/PROJECT-MESSAGE-BUS.md (created if absent)
//
// Returns the project directory path.
func InitProject(runsRoot, projectID string, hf *HomeFolders) (string, error) {
	if err := ValidateProjectID(projectID); err != nil {
		return "", err
	}
	if hf == nil {
		return "", errors.New("home-folders is nil")
	}
	projectDir := filepath.Join(runsRoot, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", errors.Wrap(err, "create project directory")
	}
	hfPath := filepath.Join(projectDir, homeFoldersFileName)
	if err := WriteHomeFolders(hfPath, hf); err != nil {
		return "", fmt.Errorf("write home-folders: %w", err)
	}
	busMDPath := filepath.Join(projectDir, "PROJECT-MESSAGE-BUS.md")
	if _, err := os.Stat(busMDPath); os.IsNotExist(err) {
		if err := os.WriteFile(busMDPath, nil, 0o644); err != nil {
			return "", errors.Wrap(err, "create PROJECT-MESSAGE-BUS.md")
		}
	}
	return projectDir, nil
}
