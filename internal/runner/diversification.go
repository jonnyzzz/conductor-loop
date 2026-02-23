package runner

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
)

// DiversificationStrategy controls how agents are selected across runs.
type DiversificationStrategy string

const (
	// StrategyRoundRobin cycles through agents in order, guaranteeing even
	// distribution over time.
	StrategyRoundRobin DiversificationStrategy = "round-robin"

	// StrategyWeighted selects agents proportionally by the configured weights.
	StrategyWeighted DiversificationStrategy = "weighted"
)

// diversificationSelector is the interface implemented by each strategy.
type diversificationSelector interface {
	// Next returns the name of the agent to use for the next run.
	Next() string

	// Fallback returns the next agent to try after the given agent failed.
	// Returns "" when no further fallback is available.
	Fallback(failed string) string
}

// roundRobinSelector cycles through agents in a fixed order.
type roundRobinSelector struct {
	mu     sync.Mutex
	agents []string // ordered list of agent names
	idx    int
}

func newRoundRobinSelector(agents []string) *roundRobinSelector {
	cp := make([]string, len(agents))
	copy(cp, agents)
	return &roundRobinSelector{agents: cp}
}

func (s *roundRobinSelector) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.agents) == 0 {
		return ""
	}
	name := s.agents[s.idx%len(s.agents)]
	s.idx++
	return name
}

func (s *roundRobinSelector) Fallback(failed string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.agents) == 0 {
		return ""
	}
	// Find the position of the failed agent and return the next one.
	for i, name := range s.agents {
		if name == failed {
			next := (i + 1) % len(s.agents)
			if s.agents[next] == failed {
				// Only one agent in the list — no fallback.
				return ""
			}
			return s.agents[next]
		}
	}
	// Not found — return the first agent that isn't the failed one.
	for _, name := range s.agents {
		if name != failed {
			return name
		}
	}
	return ""
}

// weightedSelector picks agents proportionally by weight using a simple
// cumulative distribution function approach.
type weightedSelector struct {
	mu      sync.Mutex
	agents  []string
	weights []int
	total   int
	rng     *rand.Rand
}

func newWeightedSelector(agents []string, weights []int) *weightedSelector {
	total := 0
	for _, w := range weights {
		total += w
	}
	return &weightedSelector{
		agents:  agents,
		weights: weights,
		total:   total,
		rng:     rand.New(rand.NewSource(42)), //nolint:gosec — not a security use
	}
}

func (s *weightedSelector) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.agents) == 0 || s.total == 0 {
		return ""
	}
	pick := s.rng.Intn(s.total) //nolint:gosec
	cumulative := 0
	for i, w := range s.weights {
		cumulative += w
		if pick < cumulative {
			return s.agents[i]
		}
	}
	return s.agents[len(s.agents)-1]
}

func (s *weightedSelector) Fallback(failed string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Return the highest-weight agent that is not the failed one.
	best := ""
	bestWeight := -1
	for i, name := range s.agents {
		if name == failed {
			continue
		}
		if s.weights[i] > bestWeight {
			bestWeight = s.weights[i]
			best = name
		}
	}
	return best
}

// DiversificationPolicy wraps config and a selector to provide agent selection
// with optional fallback-on-failure behaviour.
type DiversificationPolicy struct {
	cfg      *config.DiversificationConfig
	allCfg   *config.Config
	selector diversificationSelector
}

// NewDiversificationPolicy constructs a DiversificationPolicy from configuration.
// Returns nil when cfg is nil or disabled — callers treat nil as "no policy".
func NewDiversificationPolicy(d *config.DiversificationConfig, allCfg *config.Config) (*DiversificationPolicy, error) {
	if d == nil || !d.Enabled {
		return nil, nil
	}

	agents, err := resolveAgentList(d, allCfg)
	if err != nil {
		return nil, err
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("diversification: no agents available after resolution")
	}

	strategy := strings.ToLower(strings.TrimSpace(d.Strategy))
	if strategy == "" {
		strategy = string(StrategyRoundRobin)
	}

	var sel diversificationSelector
	switch strategy {
	case string(StrategyRoundRobin):
		sel = newRoundRobinSelector(agents)
	case string(StrategyWeighted):
		weights := d.Weights
		if len(weights) == 0 {
			weights = make([]int, len(agents))
			for i := range weights {
				weights[i] = 1
			}
		}
		sel = newWeightedSelector(agents, weights)
	default:
		return nil, fmt.Errorf("diversification: unknown strategy %q", strategy)
	}

	obslog.Log(log.Default(), "INFO", "runner", "diversification_policy_init",
		obslog.F("strategy", strategy),
		obslog.F("agents", strings.Join(agents, ",")),
		obslog.F("fallback_on_failure", d.FallbackOnFailure),
	)

	return &DiversificationPolicy{
		cfg:      d,
		allCfg:   allCfg,
		selector: sel,
	}, nil
}

// SelectAgent returns the next agent selection according to the diversification
// policy. When a non-empty preferred agent is given it takes precedence and the
// policy is not consulted for initial selection (but fallback still applies).
func (p *DiversificationPolicy) SelectAgent(preferred string) (agentSelection, error) {
	if preferred != "" {
		return selectAgent(p.allCfg, preferred)
	}
	name := p.selector.Next()
	if name == "" {
		return agentSelection{}, fmt.Errorf("diversification: selector returned empty agent name")
	}
	sel, err := selectAgent(p.allCfg, name)
	if err != nil {
		return agentSelection{}, fmt.Errorf("diversification: resolve agent %q: %w", name, err)
	}
	obslog.Log(log.Default(), "INFO", "runner", "diversification_agent_selected",
		obslog.F("agent_name", name),
		obslog.F("agent_type", sel.Type),
	)
	return sel, nil
}

// FallbackAgent returns the next agent to try after the given agent name failed.
// Returns an error when FallbackOnFailure is false or no fallback is available.
func (p *DiversificationPolicy) FallbackAgent(failedName string) (agentSelection, error) {
	if !p.cfg.FallbackOnFailure {
		return agentSelection{}, fmt.Errorf("diversification: fallback_on_failure is disabled")
	}
	name := p.selector.Fallback(failedName)
	if name == "" {
		return agentSelection{}, fmt.Errorf("diversification: no fallback available after %q", failedName)
	}
	sel, err := selectAgent(p.allCfg, name)
	if err != nil {
		return agentSelection{}, fmt.Errorf("diversification: resolve fallback agent %q: %w", name, err)
	}
	obslog.Log(log.Default(), "WARN", "runner", "diversification_fallback_selected",
		obslog.F("failed_agent", failedName),
		obslog.F("fallback_agent", name),
		obslog.F("fallback_type", sel.Type),
	)
	return sel, nil
}

// resolveAgentList builds the ordered agent name list from the diversification
// config, falling back to all configured agents in alphabetical order.
func resolveAgentList(d *config.DiversificationConfig, allCfg *config.Config) ([]string, error) {
	if len(d.Agents) > 0 {
		// Validate each name exists in the top-level agents map.
		for _, name := range d.Agents {
			if _, ok := allCfg.Agents[name]; !ok {
				return nil, fmt.Errorf("diversification: agent %q not found in agents config", name)
			}
		}
		return d.Agents, nil
	}
	// No explicit list — use all configured agents, sorted for determinism.
	keys := make([]string, 0, len(allCfg.Agents))
	for k := range allCfg.Agents {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys, nil
}
