package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	shellSetupBlockStart = "# >>> conductor-loop shell aliases (managed by run-agent shell-setup) >>>"
	shellSetupBlockEnd   = "# <<< conductor-loop shell aliases (managed by run-agent shell-setup) <<<"
)

var shellSetupAgents = []string{"claude", "codex", "gemini"}

type shellSetupOptions struct {
	Shell       string
	RCFile      string
	RunAgentBin string
}

func newShellSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shell-setup",
		Short: "Install or remove shell aliases for wrapped agent commands",
		Long: "Install or remove a managed shell init block that aliases claude/codex/gemini\n" +
			"to run-agent wrap for tracked task/run metadata.",
	}

	cmd.AddCommand(newShellSetupInstallCmd())
	cmd.AddCommand(newShellSetupUninstallCmd())

	return cmd
}

func newShellSetupInstallCmd() *cobra.Command {
	var opts shellSetupOptions

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install managed aliases in shell init file",
		RunE: func(cmd *cobra.Command, args []string) error {
			rcPath, shellName, err := resolveShellSetupPath(opts.Shell, opts.RCFile)
			if err != nil {
				return err
			}
			runAgentBin, err := resolveRunAgentBin(opts.RunAgentBin)
			if err != nil {
				return err
			}

			current, err := readOptionalFile(rcPath)
			if err != nil {
				return err
			}
			updated, changed, err := installShellSetupBlock(current, buildShellSetupBlock(runAgentBin))
			if err != nil {
				return err
			}
			if !changed {
				fmt.Fprintf(cmd.OutOrStdout(), "shell aliases already installed in %s\n", rcPath)
				return nil
			}
			if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
				return fmt.Errorf("create shell init parent directory: %w", err)
			}
			if err := os.WriteFile(rcPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("write shell init file: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed shell aliases in %s (%s)\n", rcPath, shellName)
			fmt.Fprintf(cmd.OutOrStdout(), "run `source %s` or open a new shell session to apply changes\n", rcPath)
			return nil
		},
	}

	addShellSetupCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&opts.RunAgentBin, "run-agent-bin", "run-agent", "run-agent executable token used in aliases")

	return cmd
}

func newShellSetupUninstallCmd() *cobra.Command {
	var opts shellSetupOptions

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove managed aliases from shell init file",
		RunE: func(cmd *cobra.Command, args []string) error {
			rcPath, shellName, err := resolveShellSetupPath(opts.Shell, opts.RCFile)
			if err != nil {
				return err
			}

			current, err := readOptionalFile(rcPath)
			if err != nil {
				return err
			}
			updated, changed, err := uninstallShellSetupBlock(current)
			if err != nil {
				return err
			}
			if !changed {
				fmt.Fprintf(cmd.OutOrStdout(), "shell aliases are already absent in %s\n", rcPath)
				return nil
			}
			if err := os.WriteFile(rcPath, []byte(updated), 0o644); err != nil {
				return fmt.Errorf("write shell init file: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed shell aliases from %s (%s)\n", rcPath, shellName)
			return nil
		},
	}

	addShellSetupCommonFlags(cmd, &opts)

	return cmd
}

func addShellSetupCommonFlags(cmd *cobra.Command, opts *shellSetupOptions) {
	cmd.Flags().StringVar(&opts.Shell, "shell", "", "shell type (zsh or bash); inferred from $SHELL when omitted")
	cmd.Flags().StringVar(&opts.RCFile, "rc-file", "", "shell init file path to edit (overrides --shell)")
}

func readOptionalFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

func resolveShellSetupPath(shellName, rcFile string) (string, string, error) {
	customPath := strings.TrimSpace(rcFile)
	if customPath != "" {
		resolved, err := expandTildePath(customPath)
		if err != nil {
			return "", "", err
		}
		return resolved, "custom", nil
	}

	requested := strings.TrimSpace(shellName)
	if requested != "" {
		normalized := normalizeShellName(requested)
		if normalized == "" {
			return "", "", fmt.Errorf("unsupported shell %q (expected zsh or bash)", requested)
		}
		path, err := shellRCPath(normalized)
		return path, normalized, err
	}

	inferred := normalizeShellName(os.Getenv("SHELL"))
	if inferred == "" {
		return "", "", fmt.Errorf("could not infer shell from SHELL; use --shell or --rc-file")
	}
	path, err := shellRCPath(inferred)
	return path, inferred, err
}

func expandTildePath(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if path == "~" {
			return home, nil
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func normalizeShellName(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.TrimPrefix(clean, "-")
	if clean == "" {
		return ""
	}
	if strings.Contains(clean, "/") {
		clean = filepath.Base(clean)
	}
	switch clean {
	case "zsh", "bash":
		return clean
	default:
		return ""
	}
}

func shellRCPath(shellName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	switch shellName {
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func resolveRunAgentBin(value string) (string, error) {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return "run-agent", nil
	}
	if strings.ContainsAny(clean, "'\" \t\r\n") {
		return "", fmt.Errorf("run-agent-bin must be a single shell token without whitespace or quotes")
	}
	return clean, nil
}

func buildShellSetupBlock(runAgentBin string) string {
	lines := []string{
		shellSetupBlockStart,
		"# Safe opt-in: aliases are installed only when this command is run explicitly.",
		"# Remove with: run-agent shell-setup uninstall",
	}
	for _, agentName := range shellSetupAgents {
		lines = append(lines, fmt.Sprintf("alias %s='%s wrap --agent %s --'", agentName, runAgentBin, agentName))
	}
	lines = append(lines, shellSetupBlockEnd, "")
	return strings.Join(lines, "\n")
}

func installShellSetupBlock(content, block string) (string, bool, error) {
	if !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	start, end, found, err := locateShellSetupBlock(content)
	if err != nil {
		return "", false, err
	}
	if found {
		updated := content[:start] + block + content[end:]
		return updated, updated != content, nil
	}

	base := content
	if base != "" && !strings.HasSuffix(base, "\n") {
		base += "\n"
	}
	if base != "" && !strings.HasSuffix(base, "\n\n") {
		base += "\n"
	}
	updated := base + block
	return updated, updated != content, nil
}

func uninstallShellSetupBlock(content string) (string, bool, error) {
	start, end, found, err := locateShellSetupBlock(content)
	if err != nil {
		return "", false, err
	}
	if !found {
		return content, false, nil
	}

	removeStart := start
	if removeStart > 0 && content[removeStart-1] == '\n' {
		removeStart--
	}

	updated := content[:removeStart] + content[end:]
	return updated, true, nil
}

func locateShellSetupBlock(content string) (int, int, bool, error) {
	start := strings.Index(content, shellSetupBlockStart)
	if start < 0 {
		return 0, 0, false, nil
	}
	endMarkerRel := strings.Index(content[start:], shellSetupBlockEnd)
	if endMarkerRel < 0 {
		return 0, 0, false, fmt.Errorf("found shell-setup start marker without end marker")
	}
	end := start + endMarkerRel + len(shellSetupBlockEnd)
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return start, end, true, nil
}
