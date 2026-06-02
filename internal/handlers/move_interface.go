package handlers

import (
	"context"
	"fmt"

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

	namespace := request["namespace"].(string)
	iface := request["interface"].(string)
	target := request["target_namespace"].(string)

	fmt.Printf(
		"move_interface namespace=%s interface=%s target_namespace=%s\n",
		namespace,
		iface,
		target,
	)

	return nil
}
