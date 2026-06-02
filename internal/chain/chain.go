package chain

import (
	"context"
	"fmt"
)

type Tools struct {
	Data map[string]any
}

type Step struct {
	Type    string         `yaml:"type"`
	Request map[string]any `yaml:"request"`
}

type Handler interface {
	Handle(
		ctx context.Context,
		tools *Tools,
		request map[string]any,
	) error
}

type Chain struct {
	tools    *Tools
	registry map[string]Handler
}

func New(
	tools *Tools,
	registry map[string]Handler,
) *Chain {
	return &Chain{
		tools:    tools,
		registry: registry,
	}
}

func (c *Chain) Run(
	ctx context.Context,
	steps []Step,
) error {

	for _, step := range steps {

		handler, ok :=
			c.registry[step.Type]

		if !ok {
			return fmt.Errorf(
				"handler '%s' not found",
				step.Type,
			)
		}

		if err := handler.Handle(
			ctx,
			c.tools,
			step.Request,
		); err != nil {
			return err
		}
	}

	return nil
}
