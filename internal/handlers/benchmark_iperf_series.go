package handlers

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/shamil-developer/vrf-filtering-research/internal/benchmark"
	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
	"github.com/shamil-developer/vrf-filtering-research/internal/command"
	"github.com/shamil-developer/vrf-filtering-research/internal/ebpf"
	"github.com/shamil-developer/vrf-filtering-research/internal/engine"
	"github.com/shamil-developer/vrf-filtering-research/internal/nft"
	"github.com/shamil-developer/vrf-filtering-research/internal/traffic"
)

type BenchmarkIperfSeries struct{}

func NewBenchmarkIperfSeries() *BenchmarkIperfSeries {
	return &BenchmarkIperfSeries{}
}

func optionalEnvInt(
	name string,
) (int, bool, error) {
	value := os.Getenv(name)
	if value == "" {
		return 0, false, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("parse %s=%q: %w", name, value, err)
	}

	return parsed, true, nil
}

func (h *BenchmarkIperfSeries) Handle(
	ctx context.Context,
	tools *chain.Tools,
	request map[string]any,
) error {
	log := logger(tools)

	name, err := requiredString(request, "name")
	if err != nil {
		return err
	}
	if value := os.Getenv("LAB_NAME"); value != "" {
		name = value
	}

	filterProfile, err := requiredString(request, "filter_profile")
	if err != nil {
		return err
	}
	engineName := optionalString(request, "engine")
	if engineName == "" {
		engineName = "nft"
	}
	if value := os.Getenv("LAB_ENGINE"); value != "" {
		engineName = value
	}

	clientNamespace, err := requiredString(request, "client_namespace")
	if err != nil {
		return err
	}

	serverNamespace, err := requiredString(request, "server_namespace")
	if err != nil {
		return err
	}

	serverIP, err := requiredString(request, "server_ip")
	if err != nil {
		return err
	}

	port, err := optionalInt(request, "port")
	if err != nil {
		return err
	}

	duration, err := optionalInt(request, "duration")
	if err != nil {
		return err
	}
	if value, ok, err := optionalEnvInt("LAB_DURATION"); err != nil {
		return err
	} else if ok {
		duration = value
	}

	rulesStart, err := optionalInt(request, "rules_start")
	if err != nil {
		return err
	}
	if value, ok, err := optionalEnvInt("LAB_RULES_START"); err != nil {
		return err
	} else if ok {
		rulesStart = value
	}

	rulesStep, err := optionalInt(request, "rules_step")
	if err != nil {
		return err
	}
	if value, ok, err := optionalEnvInt("LAB_RULES_STEP"); err != nil {
		return err
	} else if ok {
		rulesStep = value
	}

	rulesMax, err := optionalInt(request, "rules_max")
	if err != nil {
		return err
	}
	if value, ok, err := optionalEnvInt("LAB_RULES_MAX"); err != nil {
		return err
	} else if ok {
		rulesMax = value
	}

	filterProfilesPath := optionalString(request, "filter_profiles_path")
	if filterProfilesPath == "" {
		filterProfilesPath = "filters.yaml"
	}

	reportDir := optionalString(request, "report_dir")
	if reportDir == "" {
		reportDir = "reports"
	}

	commandRunner := command.NewRunner()
	ruleEngine, err := newRuleEngine(engineName, commandRunner)
	if err != nil {
		return err
	}
	seriesRunner := benchmark.NewSeriesRunner(
		log,
		ruleEngine,
		traffic.NewIperfRunner(commandRunner),
	)

	_, err = seriesRunner.Run(ctx, benchmark.SeriesConfig{
		Name:               name,
		FilterProfile:      filterProfile,
		FilterProfilesPath: filterProfilesPath,
		ClientNamespace:    clientNamespace,
		ServerNamespace:    serverNamespace,
		ServerIP:           serverIP,
		Port:               port,
		DurationSeconds:    duration,
		RulesStart:         rulesStart,
		RulesStep:          rulesStep,
		RulesMax:           rulesMax,
		ReportDir:          reportDir,
	})

	return err
}

func newRuleEngine(
	name string,
	commandRunner *command.Runner,
) (engine.RuleEngine, error) {
	switch name {
	case "nft":
		return nft.NewManager(commandRunner), nil
	case "tc+ebpf", "tc_ebpf", "ebpf":
		return ebpf.NewManager(commandRunner), nil
	default:
		return nil, fmt.Errorf("unknown engine %q", name)
	}
}
