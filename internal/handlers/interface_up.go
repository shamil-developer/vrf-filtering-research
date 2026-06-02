package handlers

import (
	"context"
	"fmt"

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

	namespace := request["namespace"].(string)
	iface := request["interface"].(string)

	fmt.Printf(
		"interface_up namespace=%s interface=%s\n",
		namespace,
		iface,
	)

	return nil
}
