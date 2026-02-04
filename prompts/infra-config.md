# Task: Implement Configuration Management

**Task ID**: infra-config
**Phase**: Core Infrastructure
**Agent Type**: Implementation (Codex preferred)
**Project Root**: ~/Work/conductor-loop

## Objective
Implement YAML-based configuration with secure token handling and validation.

## Specifications
Read: docs/specifications/subsystem-runner-orchestration-config-schema.md

## Required Implementation

### 1. Package Structure
Location: `internal/config/`
Files:
- config.go - Config struct and loading
- validation.go - Config validation
- tokens.go - Token/token_file handling

### 2. Config Struct
```go
type Config struct {
    Agents map[string]AgentConfig `yaml:"agents"`
    Defaults DefaultConfig `yaml:"defaults"`
}

type AgentConfig struct {
    Type      string `yaml:"type"`      // claude, codex, gemini, perplexity, xai
    Token     string `yaml:"token,omitempty"`
    TokenFile string `yaml:"token_file,omitempty"`
    BaseURL   string `yaml:"base_url,omitempty"`
    Model     string `yaml:"model,omitempty"`
}

type DefaultConfig struct {
    Agent   string `yaml:"agent"`
    Timeout int    `yaml:"timeout"`
}
```

### 3. Loading Logic
Implement:
- LoadConfig(path string) (*Config, error)
- Load from YAML file
- Resolve token_file to token value
- Apply environment variable overrides (CONDUCTOR_AGENT_<NAME>_TOKEN)
- Validate configuration

### 4. Token Handling
Per specification:
- If token is set, use it directly
- If token_file is set, read token from file
- Support both relative and absolute paths
- Error if both token and token_file are set
- Support environment variable override

### 5. Validation
Validate:
- At least one agent configured
- Agent type is valid (claude, codex, gemini, perplexity, xai)
- Either token or token_file is set (not both)
- Token files exist and are readable
- Timeouts are positive

## Tests Required
Location: `test/unit/config_test.go`
- TestLoadConfig
- TestTokenFileResolution
- TestTokenFromEnv
- TestConfigValidation (invalid configs)
- TestAgentDefaults

## Success Criteria
- All tests pass
- IntelliJ MCP review: no warnings
- Example config.yaml works

## Example Config
```yaml
agents:
  claude:
    type: claude
    token_file: ~/.config/claude/token

  codex:
    type: codex
    token_file: ~/.config/codex/token

  gemini:
    type: gemini
    token: ${GEMINI_API_KEY}

defaults:
  agent: claude
  timeout: 3600
```

## References
- docs/decisions/CRITICAL-PROBLEMS-RESOLVED.md
- THE_PROMPT_v5.md: Stage 5 (Implement changes and tests)

## Output
Log to MESSAGE-BUS.md:
- FACT: Configuration system implemented
- FACT: N unit tests passing
- FACT: Token handling secure
