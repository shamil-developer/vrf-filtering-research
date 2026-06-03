package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
	"github.com/shamil-developer/vrf-filtering-research/internal/metrics"
)

type CompareReports struct{}

func NewCompareReports() *CompareReports {
	return &CompareReports{}
}

func (h *CompareReports) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {
	_ = ctx

	log := logger(tools)

	leftPath, err := requiredString(request, "left")
	if err != nil {
		return err
	}
	rightPath, err := requiredString(request, "right")
	if err != nil {
		return err
	}

	left, err := metrics.ReadJSON(leftPath)
	if err != nil {
		return err
	}
	right, err := metrics.ReadJSON(rightPath)
	if err != nil {
		return err
	}

	rows := metrics.CompareResults(left, right)
	log.Print(metrics.CompareTable(rows))

	return nil
}
