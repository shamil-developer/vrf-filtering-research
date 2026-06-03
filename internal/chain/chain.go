package chain

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
)

type Tools struct {
	Data map[string]any
}

func (t *Tools) Get(
	name string,
) any {
	if t == nil || t.Data == nil {
		return nil
	}

	return t.Data[name]
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
	logger, _ := c.tools.Get("logger").(*log.Logger)

	if logger != nil {
		logger.Info("Начинаю выполнять карту", "шагов", len(steps))
	}

	for index, step := range steps {
		if logger != nil {
			logger.Info(
				"Выполняю шаг",
				"шаг",
				index+1,
				"всего",
				len(steps),
				"тип",
				step.Type,
			)
		}

		handler, ok :=
			c.registry[step.Type]

		if !ok {
			if logger != nil {
				logger.Error("Handler для шага не найден", "тип", step.Type)
			}

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
			if logger != nil {
				logger.Error(
					"Шаг завершился ошибкой",
					"шаг",
					index+1,
					"тип",
					step.Type,
					"ошибка",
					err,
				)
			}

			return err
		}

		if logger != nil {
			logger.Info("Шаг выполнен", "шаг", index+1, "тип", step.Type)
		}
	}

	if logger != nil {
		logger.Info("Карта выполнена полностью")
	}

	return nil
}
