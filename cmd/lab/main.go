package main

import (
	"context"
	"os"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
	"github.com/shamil-developer/vrf-filtering-research/internal/handlers"
	"github.com/shamil-developer/vrf-filtering-research/internal/network"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Steps []chain.Step `yaml:"steps"`
}

func main() {
	logger := log.NewWithOptions(
		os.Stdout,
		log.Options{
			ReportTimestamp: true,
			Prefix:          "lab",
		},
	)

	configPath := os.Getenv("LAB_CONFIG")
	if configPath == "" {
		configPath = "lab.yaml"
	}

	logger.Info("Запускаю сетевой стенд", "конфиг", configPath)

	data, err := os.ReadFile(
		configPath,
	)
	if err != nil {
		logger.Fatal("Не удалось прочитать конфигурацию", "ошибка", err)
	}
	logger.Info("Конфигурация прочитана", "байт", len(data))

	var cfg Config

	if err := yaml.Unmarshal(
		data,
		&cfg,
	); err != nil {
		logger.Fatal("Не удалось разобрать YAML", "ошибка", err)
	}
	logger.Info("YAML разобран", "шагов", len(cfg.Steps))

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tools := &chain.Tools{
		Data: map[string]any{
			"network": network.NewTool(),
			"logger":  logger,
		},
	}

	c := chain.New(
		tools,
		map[string]chain.Handler{
			"print":                  handlers.NewPrint(),
			"create_namespace":       handlers.NewCreateNamespace(),
			"create_interface":       handlers.NewCreateInterface(),
			"move_interface":         handlers.NewMoveInterface(),
			"attach_interface":       handlers.NewAttachInterface(),
			"assign_ip":              handlers.NewAssignIP(),
			"add_route":              handlers.NewAddRoute(),
			"set_sysctl":             handlers.NewSetSysctl(),
			"interface_up":           handlers.NewInterfaceUp(),
			"benchmark_iperf_series": handlers.NewBenchmarkIperfSeries(),
			"compare_reports":        handlers.NewCompareReports(),
		},
	)

	if err := c.Run(
		context.Background(),
		cfg.Steps,
	); err != nil {
		logger.Fatal("Стенд не был создан", "ошибка", err)
	}

	logger.Info("Стенд создан успешно")
}
