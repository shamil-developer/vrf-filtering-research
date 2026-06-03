package engine

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
)

type ProgressFunc func(done int, total int)

type RuleEngine interface {
	Name() string
	ApplyRulesWithProgress(
		ctx context.Context,
		profile filterprofile.Profile,
		count int,
		progress ProgressFunc,
	) error
}
