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

	namespace := request["namespace"].(string)
	kind := request["kind"].(string)

	switch kind {

	case "bridge":

		name := request["name"].(string)

		fmt.Printf(
			"create_interface namespace=%s kind=bridge name=%s\n",
			namespace,
			name,
		)

	case "vrf":

		name := request["name"].(string)
		table := request["table"]

		fmt.Printf(
			"create_interface namespace=%s kind=vrf name=%s table=%v\n",
			namespace,
			name,
			table,
		)

	case "veth":

		left := request["left"].(string)
		right := request["right"].(string)

		fmt.Printf(
			"create_interface namespace=%s kind=veth left=%s right=%s\n",
			namespace,
			left,
			right,
		)

	default:

		return fmt.Errorf(
			"unknown interface kind: %s",
			kind,
		)
	}

	return nil
}
