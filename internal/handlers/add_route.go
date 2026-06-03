package handlers

import (
	"context"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type AddRoute struct{}

func NewAddRoute() *AddRoute {
	return &AddRoute{}
}

func (h *AddRoute) Handle(
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
	dst := optionalString(request, "dst")
	gateway := optionalString(request, "gateway")
	iface := optionalString(request, "interface")

	table, err := optionalInt(request, "table")
	if err != nil {
		return err
	}

	log.Info(
		"Добавляю маршрут",
		"пространство",
		displayNamespace(namespace),
		"назначение",
		displayDefault(dst),
		"шлюз",
		displayEmpty(gateway),
		"интерфейс",
		displayEmpty(iface),
		"таблица",
		displayTable(table),
	)

	return net.AddRoute(namespace, dst, gateway, iface, table)
}
