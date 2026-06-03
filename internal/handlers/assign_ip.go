package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type AssignIP struct{}

func NewAssignIP() *AssignIP {
	return &AssignIP{}
}

func (h *AssignIP) Handle(
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

	address, err := requiredString(request, "address")
	if err != nil {
		return err
	}

	log.Info(
		"Назначаю IP-адрес",
		"пространство",
		displayNamespace(namespace),
		"интерфейс",
		iface,
		"адрес",
		address,
	)

	return net.AssignIP(namespace, iface, address)
}
