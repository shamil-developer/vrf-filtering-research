package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type SetSysctl struct{}

func NewSetSysctl() *SetSysctl {
	return &SetSysctl{}
}

func (h *SetSysctl) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {
	log := logger(tools)

	net, err := networkTool(tools)
	if err != nil {
		return err
	}

	name, err := requiredString(request, "name")
	if err != nil {
		return err
	}

	value, err := requiredString(request, "value")
	if err != nil {
		return err
	}

	log.Info(
		"Настраиваю sysctl",
		"параметр",
		name,
		"значение",
		value,
	)

	return net.SetSysctl(name, value)
}
