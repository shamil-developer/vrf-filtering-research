package nft

import (
	"context"
	"fmt"

	"github.com/shamil-developer/vrf-filtering-research/internal/command"
	"github.com/shamil-developer/vrf-filtering-research/internal/engine"
	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
)

type Manager struct {
	runner *command.Runner
}

func NewManager(
	runner *command.Runner,
) *Manager {
	return &Manager{
		runner: runner,
	}
}

func (m *Manager) Name() string {
	return "nft"
}

func (m *Manager) ApplyRules(
	ctx context.Context,
	profile filterprofile.Profile,
	count int,
) error {
	return m.ApplyRulesWithProgress(ctx, profile, count, nil)
}

func (m *Manager) ApplyRulesWithProgress(
	ctx context.Context,
	profile filterprofile.Profile,
	count int,
	progress engine.ProgressFunc,
) error {
	if err := m.Reset(ctx, profile); err != nil {
		return err
	}

	rules := profile.GenerateRules(count)
	total := len(rules)
	for index, rule := range rules {
		command := rule.Command

		if _, err := m.runner.RunShell(ctx, command); err != nil {
			return fmt.Errorf("add nft rule %q: %w", rule.Rule, err)
		}

		if progress != nil {
			progress(index+1, total)
		}
	}

	return nil
}

func (m *Manager) Reset(
	ctx context.Context,
	profile filterprofile.Profile,
) error {
	_, _ = m.runner.Run(ctx, "nft", "delete", "table", profile.Family, profile.Table)

	if _, err := m.runner.Run(ctx, "nft", "add", "table", profile.Family, profile.Table); err != nil {
		return fmt.Errorf("create nft table %s %s: %w", profile.Family, profile.Table, err)
	}

	chainDefinition := fmt.Sprintf(
		"{ type filter hook %s priority %d; policy %s; }",
		profile.Hook,
		profile.Priority,
		profile.Policy,
	)

	if _, err := m.runner.Run(
		ctx,
		"nft",
		"add",
		"chain",
		profile.Family,
		profile.Table,
		profile.Chain,
		chainDefinition,
	); err != nil {
		return fmt.Errorf("create nft chain %s %s %s: %w", profile.Family, profile.Table, profile.Chain, err)
	}

	return nil
}
