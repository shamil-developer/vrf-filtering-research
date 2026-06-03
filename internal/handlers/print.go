package handlers

import (
	"context"
	"fmt"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
)

type Print struct{}

func NewPrint() *Print {
	return &Print{}
}

func (h *Print) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {

	message, ok :=
		request["message"].(string)

	if !ok {
		return fmt.Errorf(
			"message not found",
		)
	}

	logger(tools).Info(message)

	return nil
}
