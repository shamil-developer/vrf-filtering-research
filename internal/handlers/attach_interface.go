package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type AttachInterface struct{}

func NewAttachInterface() *AttachInterface {
	return &AttachInterface{}
}

func (h *AttachInterface) Handle(
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

	parent, err := requiredString(request, "parent")
	if err != nil {
		return err
	}

	child, err := requiredString(request, "child")
	if err != nil {
		return err
	}

	log.Info(
		"Подключаю интерфейс к master-устройству",
		"пространство",
		displayNamespace(namespace),
		"master",
		parent,
		"интерфейс",
		child,
	)

	return net.AttachInterface(namespace, parent, child)
}
