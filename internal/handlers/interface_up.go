package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type InterfaceUp struct{}

func NewInterfaceUp() *InterfaceUp {
	return &InterfaceUp{}
}

func (h *InterfaceUp) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {
	log := logger(tools)

	net, err := networkTool(tools)
	if err != nil {
		return err
	}

	namespace := optionalString(request, "namespace")

	iface, err := requiredString(request, "interface")
	if err != nil {
		return err
	}

	log.Info(
		"Поднимаю интерфейс",
		"пространство",
		displayNamespace(namespace),
		"интерфейс",
		iface,
	)

	return net.InterfaceUp(namespace, iface)
}
