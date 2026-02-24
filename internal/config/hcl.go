// Package config — minimal HCL config parser for ~/.run-agent/conductor-loop.hcl.
// Supports a subset of HCL syntax sufficient for conductor-loop configuration:
//   - Block headers:        name { ... }
//   - String values:        key = "value"
//   - Integer values:       key = 42
//   - Single-line comments: # ... or // ...
//
// Agent type is inferred from the block name when the "type" attribute is absent,
// so "codex { ... }" needs no explicit type = "codex".
//
// Reserved block names: defaults, api, storage.
// All other blocks are treated as agent configurations.
package config

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// parseHCLConfig parses a minimal HCL config file into a Config struct.
func parseHCLConfig(data []byte) (*Config, error) {
	blocks, err := parseHCLBlocks(data)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Agents: make(map[string]AgentConfig),
	}

	for _, b := range blocks {
		switch b.name {
		case "defaults":
			if err := applyHCLDefaultsBlock(cfg, b.values); err != nil {
				return nil, fmt.Errorf("defaults block: %w", err)
			}
		case "api":
			if err := applyHCLAPIBlock(cfg, b.values); err != nil {
				return nil, fmt.Errorf("api block: %w", err)
			}
		case "storage":
			if err := applyHCLStorageBlock(cfg, b.values); err != nil {
				return nil, fmt.Errorf("storage block: %w", err)
			}
		default:
			// Agent block — type inferred from block name if absent
			agent := AgentConfig{}
			if v, ok := b.values["type"]; ok {
				agent.Type = v
			}
			if v, ok := b.values["token"]; ok {
				agent.Token = v
			}
			if v, ok := b.values["token_file"]; ok {
				agent.TokenFile = v
			}
			if v, ok := b.values["base_url"]; ok {
				agent.BaseURL = v
			}
			if v, ok := b.values["model"]; ok {
				agent.Model = v
			}
			cfg.Agents[b.name] = agent
		}
	}

	return cfg, nil
}

// hclBlock holds a parsed HCL block.
type hclBlock struct {
	name   string
	values map[string]string
}

// parseHCLBlocks reads the flat-block HCL subset into a slice of named blocks.
func parseHCLBlocks(data []byte) ([]hclBlock, error) {
	var blocks []hclBlock
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0

	var current *hclBlock
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments.
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		if current == nil {
			// Expect: identifier {
			if !strings.HasSuffix(line, "{") {
				return nil, fmt.Errorf("line %d: expected block open (name {), got: %q", lineNum, line)
			}
			name := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			if name == "" {
				return nil, fmt.Errorf("line %d: empty block name", lineNum)
			}
			if strings.ContainsAny(name, " \t") {
				return nil, fmt.Errorf("line %d: block name %q must not contain spaces", lineNum, name)
			}
			current = &hclBlock{name: name, values: make(map[string]string)}
		} else {
			if line == "}" {
				blocks = append(blocks, *current)
				current = nil
				continue
			}
			// key = value
			idx := strings.Index(line, "=")
			if idx < 0 {
				return nil, fmt.Errorf("line %d: expected key = value, got: %q", lineNum, line)
			}
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			val = hclUnquote(val)
			current.values[key] = val
		}
	}

	if scanner.Err() != nil {
		return nil, fmt.Errorf("scan: %w", scanner.Err())
	}
	if current != nil {
		return nil, fmt.Errorf("unexpected end of file: block %q not closed", current.name)
	}

	return blocks, nil
}

// hclUnquote removes surrounding double-quotes from a string value and unescapes
// basic escape sequences. Returns the raw value unchanged for non-quoted tokens
// (e.g. integer literals, booleans).
func hclUnquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		inner = strings.ReplaceAll(inner, `\n`, "\n")
		return inner
	}
	return s
}

func applyHCLDefaultsBlock(cfg *Config, values map[string]string) error {
	if v, ok := values["agent"]; ok {
		cfg.Defaults.Agent = v
	}
	if v, ok := values["timeout"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("timeout: %w", err)
		}
		cfg.Defaults.Timeout = n
	}
	if v, ok := values["max_concurrent_runs"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("max_concurrent_runs: %w", err)
		}
		cfg.Defaults.MaxConcurrentRuns = n
	}
	if v, ok := values["max_concurrent_root_tasks"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("max_concurrent_root_tasks: %w", err)
		}
		cfg.Defaults.MaxConcurrentRootTasks = n
	}
	return nil
}

func applyHCLAPIBlock(cfg *Config, values map[string]string) error {
	if v, ok := values["host"]; ok {
		cfg.API.Host = v
	}
	if v, ok := values["port"]; ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("port: %w", err)
		}
		cfg.API.Port = n
	}
	return nil
}

// homeHCLTemplate is written to ~/.run-agent/conductor-loop.hcl on first run
// when the file does not already exist.
const homeHCLTemplate = `# conductor-loop — personal configuration
# Documentation: https://github.com/jonnyzzz/conductor-loop/blob/main/docs/user/configuration.md
# GitHub:        https://github.com/jonnyzzz/conductor-loop
#
# Add your agent API tokens below.
# The agent type is inferred from the block name — no "type" field required.
#
# Example:
#
# codex {
#   token_file = "~/.config/tokens/openai"
# }
#
# claude {
#   token_file = "~/.config/tokens/anthropic"
# }
#
# gemini {
#   token_file = "~/.config/tokens/google"
# }
#
# defaults {
#   agent               = "claude"
#   timeout             = 300
#   max_concurrent_runs = 4
# }
`

func applyHCLStorageBlock(cfg *Config, values map[string]string) error {
	if v, ok := values["runs_dir"]; ok {
		cfg.Storage.RunsDir = v
	}
	return nil
}
