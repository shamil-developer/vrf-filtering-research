package filterprofile

import (
	"strings"
	"testing"
)

func TestProfilesGenerateDiverseRules(t *testing.T) {
	config, err := LoadFile("../../filters.yaml")
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}

	for name := range config.Profiles {
		profile, err := config.Get(name)
		if err != nil {
			t.Fatalf("get profile %s: %v", name, err)
		}

		rules := profile.GenerateRules(100)
		commands := make(map[string]struct{})
		types := make(map[string]struct{})
		actions := make(map[string]struct{})
		for _, rule := range rules {
			if rule.IsFinal {
				continue
			}

			commands[rule.Command] = struct{}{}
			types[rule.Type] = struct{}{}
			actions[rule.Action] = struct{}{}
		}

		if len(commands) < 80 {
			t.Fatalf("profile %s generated only %d unique commands", name, len(commands))
		}
		if len(types) < 5 {
			t.Fatalf("profile %s generated only %d rule types", name, len(types))
		}
		if len(actions) < 6 {
			t.Fatalf("profile %s generated only %d action types", name, len(actions))
		}
	}
}

func TestProfilesDistributeRulesAcrossTopology(t *testing.T) {
	config, err := LoadFile("../../filters.yaml")
	if err != nil {
		t.Fatalf("load profiles: %v", err)
	}

	for name := range config.Profiles {
		profile, err := config.Get(name)
		if err != nil {
			t.Fatalf("get profile %s: %v", name, err)
		}

		rules := profile.GenerateRules(60)
		directions := make(map[string]struct{})
		segments := make(map[string]struct{})
		for _, rule := range rules {
			if rule.IsFinal {
				continue
			}

			if rule.Direction == "" {
				t.Fatalf("profile %s rule %d has empty direction", name, rule.Index)
			}
			if rule.Segment == "" {
				t.Fatalf("profile %s rule %d has empty segment", name, rule.Index)
			}
			if !strings.Contains(rule.Rule, "iifname") || !strings.Contains(rule.Rule, "oifname") {
				t.Fatalf("profile %s rule %d is not bound to map interfaces: %s", name, rule.Index, rule.Rule)
			}
			if hasUnresolvedPlaceholder(rule.Command) {
				t.Fatalf("profile %s rule %d has unresolved placeholder: %s", name, rule.Index, rule.Command)
			}

			directions[rule.Direction] = struct{}{}
			segments[rule.Segment] = struct{}{}
		}

		if len(directions) != len(profile.Topology.Directions) {
			t.Fatalf("profile %s covered %d directions, want %d", name, len(directions), len(profile.Topology.Directions))
		}
		if len(segments) < 3 {
			t.Fatalf("profile %s covered only %d topology segments", name, len(segments))
		}
	}
}

func hasUnresolvedPlaceholder(command string) bool {
	placeholders := []string{
		"{action}",
		"{index}",
		"{bridge}",
		"{vrf}",
		"{segment}",
		"{direction}",
		"{iif}",
		"{oif}",
		"{path}",
		"{a}",
		"{b}",
		"{port}",
		"{length}",
		"{ttl}",
		"{mark}",
		"{zone}",
		"{spi}",
		"{reqid}",
	}
	for _, placeholder := range placeholders {
		if strings.Contains(command, placeholder) {
			return true
		}
	}

	return false
}
