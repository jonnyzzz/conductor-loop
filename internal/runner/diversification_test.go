package runner

import (
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/config"
)

func makeTestConfig(agentNames ...string) *config.Config {
	agents := make(map[string]config.AgentConfig, len(agentNames))
	for _, name := range agentNames {
		agents[name] = config.AgentConfig{Type: name}
	}
	return &config.Config{
		Agents: agents,
		Defaults: config.DefaultConfig{
			Timeout: 300,
		},
	}
}

// --- roundRobinSelector ---

func TestRoundRobinSelector_Cycles(t *testing.T) {
	sel := newRoundRobinSelector([]string{"a", "b", "c"})
	want := []string{"a", "b", "c", "a", "b", "c"}
	for i, w := range want {
		got := sel.Next()
		if got != w {
			t.Fatalf("call %d: got %q, want %q", i, got, w)
		}
	}
}

func TestRoundRobinSelector_Single(t *testing.T) {
	sel := newRoundRobinSelector([]string{"only"})
	for i := 0; i < 5; i++ {
		if got := sel.Next(); got != "only" {
			t.Fatalf("call %d: got %q, want %q", i, got, "only")
		}
	}
}

func TestRoundRobinSelector_Empty(t *testing.T) {
	sel := newRoundRobinSelector([]string{})
	if got := sel.Next(); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestRoundRobinSelector_Fallback(t *testing.T) {
	sel := newRoundRobinSelector([]string{"a", "b", "c"})
	tests := []struct {
		failed string
		want   string
	}{
		{"a", "b"},
		{"b", "c"},
		{"c", "a"},
	}
	for _, tt := range tests {
		got := sel.Fallback(tt.failed)
		if got != tt.want {
			t.Errorf("Fallback(%q) = %q, want %q", tt.failed, got, tt.want)
		}
	}
}

func TestRoundRobinSelector_FallbackSingle(t *testing.T) {
	sel := newRoundRobinSelector([]string{"only"})
	if got := sel.Fallback("only"); got != "" {
		t.Fatalf("single-agent fallback should return empty, got %q", got)
	}
}

func TestRoundRobinSelector_FallbackUnknown(t *testing.T) {
	sel := newRoundRobinSelector([]string{"a", "b"})
	got := sel.Fallback("unknown")
	if got == "" {
		t.Fatal("unknown failed agent should return first non-failed agent")
	}
}

// --- weightedSelector ---

func TestWeightedSelector_Next_ReturnsAgents(t *testing.T) {
	sel := newWeightedSelector([]string{"a", "b"}, []int{1, 1})
	seen := map[string]int{}
	for i := 0; i < 100; i++ {
		seen[sel.Next()]++
	}
	if seen["a"] == 0 || seen["b"] == 0 {
		t.Fatalf("expected both agents selected over 100 calls, got %v", seen)
	}
}

func TestWeightedSelector_Fallback(t *testing.T) {
	sel := newWeightedSelector([]string{"a", "b", "c"}, []int{1, 10, 1})
	// highest weight when a fails should be b
	got := sel.Fallback("a")
	if got != "b" {
		t.Errorf("Fallback(a) = %q, want %q", got, "b")
	}
}

func TestWeightedSelector_FallbackNoOption(t *testing.T) {
	sel := newWeightedSelector([]string{"only"}, []int{5})
	if got := sel.Fallback("only"); got != "" {
		t.Fatalf("single-agent fallback should return empty, got %q", got)
	}
}

// --- NewDiversificationPolicy ---

func TestNewDiversificationPolicy_Nil_ReturnsNil(t *testing.T) {
	p, err := NewDiversificationPolicy(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Fatal("expected nil policy for nil config")
	}
}

func TestNewDiversificationPolicy_Disabled_ReturnsNil(t *testing.T) {
	d := &config.DiversificationConfig{Enabled: false}
	p, err := NewDiversificationPolicy(d, makeTestConfig("claude"))
	if err != nil {
		t.Fatal(err)
	}
	if p != nil {
		t.Fatal("expected nil policy when disabled")
	}
}

func TestNewDiversificationPolicy_RoundRobin(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:  true,
		Strategy: "round-robin",
		Agents:   []string{"claude", "gemini"},
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil policy")
	}
}

func TestNewDiversificationPolicy_UnknownStrategy(t *testing.T) {
	cfg := makeTestConfig("claude")
	d := &config.DiversificationConfig{
		Enabled:  true,
		Strategy: "banana",
	}
	_, err := NewDiversificationPolicy(d, cfg)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestNewDiversificationPolicy_UnknownAgent(t *testing.T) {
	cfg := makeTestConfig("claude")
	d := &config.DiversificationConfig{
		Enabled: true,
		Agents:  []string{"notfound"},
	}
	_, err := NewDiversificationPolicy(d, cfg)
	if err == nil {
		t.Fatal("expected error for unknown agent in policy")
	}
}

func TestNewDiversificationPolicy_UsesAllAgentsWhenEmpty(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:  true,
		Strategy: "round-robin",
		// No Agents specified — should default to all.
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("expected non-nil policy")
	}
	// Should cycle through both agents.
	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		sel, err := p.SelectAgent("")
		if err != nil {
			t.Fatal(err)
		}
		seen[sel.Name] = true
	}
	if !seen["claude"] || !seen["gemini"] {
		t.Fatalf("expected both agents seen, got %v", seen)
	}
}

// --- DiversificationPolicy.SelectAgent ---

func TestDiversificationPolicy_SelectAgent_BypassesWithPreferred(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:  true,
		Strategy: "round-robin",
		Agents:   []string{"claude", "gemini"},
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Explicit preferred agent should bypass policy.
	sel, err := p.SelectAgent("gemini")
	if err != nil {
		t.Fatal(err)
	}
	if sel.Name != "gemini" {
		t.Errorf("SelectAgent(gemini) = %q, want gemini", sel.Name)
	}
}

func TestDiversificationPolicy_SelectAgent_RoundRobin(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:  true,
		Strategy: "round-robin",
		Agents:   []string{"claude", "gemini"},
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}

	got0, _ := p.SelectAgent("")
	got1, _ := p.SelectAgent("")
	got2, _ := p.SelectAgent("")

	if got0.Name == got1.Name {
		t.Errorf("consecutive round-robin calls should alternate: both returned %q", got0.Name)
	}
	if got2.Name != got0.Name {
		t.Errorf("third call should wrap to first: got %q, want %q", got2.Name, got0.Name)
	}
}

// --- DiversificationPolicy.FallbackAgent ---

func TestDiversificationPolicy_FallbackAgent_DisabledReturnsError(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:           true,
		Strategy:          "round-robin",
		Agents:            []string{"claude", "gemini"},
		FallbackOnFailure: false,
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.FallbackAgent("claude")
	if err == nil {
		t.Fatal("expected error when fallback_on_failure is disabled")
	}
}

func TestDiversificationPolicy_FallbackAgent_ReturnsNext(t *testing.T) {
	cfg := makeTestConfig("claude", "gemini")
	d := &config.DiversificationConfig{
		Enabled:           true,
		Strategy:          "round-robin",
		Agents:            []string{"claude", "gemini"},
		FallbackOnFailure: true,
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}

	fb, err := p.FallbackAgent("claude")
	if err != nil {
		t.Fatal(err)
	}
	if fb.Name != "gemini" {
		t.Errorf("FallbackAgent(claude) = %q, want gemini", fb.Name)
	}
}

func TestDiversificationPolicy_FallbackAgent_NoFallbackForSingle(t *testing.T) {
	cfg := makeTestConfig("claude")
	d := &config.DiversificationConfig{
		Enabled:           true,
		Strategy:          "round-robin",
		Agents:            []string{"claude"},
		FallbackOnFailure: true,
	}
	p, err := NewDiversificationPolicy(d, cfg)
	if err != nil {
		t.Fatal(err)
	}

	_, err = p.FallbackAgent("claude")
	if err == nil {
		t.Fatal("expected error: no fallback available for single-agent policy")
	}
}

// --- resolveAgentList ---

func TestResolveAgentList_ExplicitOrder(t *testing.T) {
	cfg := makeTestConfig("claude", "codex", "gemini")
	d := &config.DiversificationConfig{
		Agents: []string{"gemini", "claude"},
	}
	got, err := resolveAgentList(d, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "gemini" || got[1] != "claude" {
		t.Errorf("unexpected list %v", got)
	}
}

func TestResolveAgentList_AllAgentsFallback(t *testing.T) {
	cfg := makeTestConfig("gemini", "claude") // map order intentionally mixed
	d := &config.DiversificationConfig{}
	got, err := resolveAgentList(d, cfg)
	if err != nil {
		t.Fatal(err)
	}
	// Should be sorted alphabetically.
	if len(got) != 2 || got[0] != "claude" || got[1] != "gemini" {
		t.Errorf("expected [claude gemini], got %v", got)
	}
}

func TestResolveAgentList_UnknownAgentError(t *testing.T) {
	cfg := makeTestConfig("claude")
	d := &config.DiversificationConfig{
		Agents: []string{"unknown"},
	}
	_, err := resolveAgentList(d, cfg)
	if err == nil {
		t.Fatal("expected error for unknown agent name")
	}
}
