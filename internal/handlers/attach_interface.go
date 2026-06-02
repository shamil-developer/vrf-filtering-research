package handlers

import (
	"context"
	"fmt"

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

	namespace := request["namespace"].(string)
	parent := request["parent"].(string)
	child := request["child"].(string)

	fmt.Printf(
		"attach_interface namespace=%s parent=%s child=%s\n",
		namespace,
		parent,
		child,
	)

	return nil
}
