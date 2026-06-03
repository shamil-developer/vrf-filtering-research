package handlers

import (
	"context"
	"fmt"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type CreateInterface struct{}

func NewCreateInterface() *CreateInterface {
	return &CreateInterface{}
}

func (h *CreateInterface) Handle(
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

	kind, err := requiredString(request, "kind")
	if err != nil {
		return err
	}

	switch kind {

	case "bridge":

		name, err := requiredString(request, "name")
		if err != nil {
			return err
		}

		log.Info(
			"Создаю Linux bridge",
			"пространство",
			displayNamespace(namespace),
			"имя",
			name,
		)

		return net.CreateBridge(namespace, name)

	case "vrf":

		name, err := requiredString(request, "name")
		if err != nil {
			return err
		}

		table, err := optionalInt(request, "table")
		if err != nil {
			return err
		}
		if table == 0 {
			return fmt.Errorf("table not found")
		}

		log.Info(
			"Создаю VRF",
			"пространство",
			displayNamespace(namespace),
			"имя",
			name,
			"таблица",
			table,
		)

		return net.CreateVRF(namespace, name, uint32(table))

	case "veth":

		left, err := requiredString(request, "left")
		if err != nil {
			return err
		}

		right, err := requiredString(request, "right")
		if err != nil {
			return err
		}

		log.Info(
			"Создаю veth-пару",
			"пространство",
			displayNamespace(namespace),
			"левый",
			left,
			"правый",
			right,
		)

		return net.CreateVeth(namespace, left, right)

	default:

		return fmt.Errorf(
			"unknown interface kind: %s",
			kind,
		)
	}

}
