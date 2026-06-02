package handlers

import (
	"context"
	"fmt"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type CreateNamespace struct{}

func NewCreateNamespace() *CreateNamespace {
	return &CreateNamespace{}
}

func (h *CreateNamespace) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {

	namespace, _ := request["namespace"].(string)
	name := request["name"].(string)

	fmt.Printf(
		"create_namespace namespace=%s name=%s\n",
		namespace,
		name,
	)

	return nil
}
