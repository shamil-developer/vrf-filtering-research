package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type MoveInterface struct{}

func NewMoveInterface() *MoveInterface {
	return &MoveInterface{}
}

func (h *MoveInterface) Handle(
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

	target, err := requiredString(request, "target_namespace")
	if err != nil {
		return err
	}

	log.Info(
		"Переношу интерфейс в namespace",
		"из",
		displayNamespace(namespace),
		"интерфейс",
		iface,
		"в",
		target,
	)

	return net.MoveInterface(namespace, iface, target)
}
