package handlers

import (
	"context"
	"fmt"

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

	namespace := request["namespace"].(string)
	iface := request["interface"].(string)
	address := request["address"].(string)

	fmt.Printf(
		"assign_ip namespace=%s interface=%s address=%s\n",
		namespace,
		iface,
		address,
	)

	return nil
}
