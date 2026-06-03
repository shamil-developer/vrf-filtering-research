package benchmark

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/log"
	"github.com/shamil-developer/vrf-filtering-research/internal/engine"
	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
	"github.com/shamil-developer/vrf-filtering-research/internal/metrics"
	"github.com/shamil-developer/vrf-filtering-research/internal/progress"
	"github.com/shamil-developer/vrf-filtering-research/internal/traffic"
)

type SeriesConfig struct {
	Name               string
	FilterProfile      string
	FilterProfilesPath string
	ClientNamespace    string
	ServerNamespace    string
	ServerIP           string
	Port               int
	DurationSeconds    int
	RulesStart         int
	RulesStep          int
	RulesMax           int
	ReportDir          string
}

type SeriesRunner struct {
	logger  *log.Logger
	engine  engine.RuleEngine
	traffic *traffic.IperfRunner
}

func NewSeriesRunner(
	logger *log.Logger,
	ruleEngine engine.RuleEngine,
	trafficRunner *traffic.IperfRunner,
) *SeriesRunner {
	return &SeriesRunner{
		logger:  logger,
		engine:  ruleEngine,
		traffic: trafficRunner,
	}
}

func (r *SeriesRunner) Run(
	ctx context.Context,
	config SeriesConfig,
) ([]metrics.Result, error) {
	config = config.withDefaults()

	if err := config.validate(); err != nil {
		return nil, err
	}

	profiles, err := filterprofile.LoadFile(config.FilterProfilesPath)
	if err != nil {
		return nil, err
	}

	profile, err := profiles.Get(config.FilterProfile)
	if err != nil {
		return nil, err
	}

	store := metrics.NewStore()

	r.logger.Info(
		"Запускаю benchmark-серию",
		"имя",
		config.Name,
		"engine",
		r.engine.Name(),
		"профиль",
		config.FilterProfile,
		"старт_правил",
		config.RulesStart,
		"шаг_правил",
		config.RulesStep,
		"максимум_правил",
		config.RulesMax,
		"секунд",
		config.DurationSeconds,
	)

	previousCount := 0
	for count := config.RulesStart; count <= config.RulesMax; count += config.RulesStep {
		r.logger.Info("Готовлю ruleset", "engine", r.engine.Name(), "правил", count)

		rulesProgress := progress.NewLine()
		lastRulesProgress := 0
		if count > 0 {
			rulesProgress.Update(formatRulesProgress(r.engine.Name(), count, config.RulesMax, 0))
		}

		if err := r.engine.ApplyRulesWithProgress(ctx, profile, count, func(done int, total int) {
			if total <= 1 {
				return
			}

			visibleTotal := total - 1
			visibleDone := done
			if visibleDone > visibleTotal {
				visibleDone = visibleTotal
			}

			if visibleDone == 0 {
				return
			}
			if visibleDone == lastRulesProgress {
				return
			}

			interval := 100
			if visibleTotal < interval {
				interval = visibleTotal
			}

			if visibleDone == visibleTotal || visibleDone-lastRulesProgress >= interval {
				lastRulesProgress = visibleDone
				rulesProgress.Update(formatRulesProgress(r.engine.Name(), count, config.RulesMax, visibleDone))
			}
		}); err != nil {
			rulesProgress.Done("")
			return nil, err
		}
		if count > 0 {
			rulesProgress.Done(formatRulesDone(r.engine.Name(), count, config.RulesMax))
		}

		addedRulesTable := renderAddedRulesTable(profile, r.engine.Name(), previousCount, count, config.RulesMax)
		if addedRulesTable != "" {
			r.logger.Print(addedRulesTable)
		}

		r.logger.Info(
			"Запускаю передачу трафика",
			"правил",
			count,
			"клиент",
			config.ClientNamespace,
			"сервер",
			config.ServerNamespace,
			"адрес_сервера",
			config.ServerIP,
			"секунд",
			config.DurationSeconds,
		)

		trafficProgress := progress.NewLine()
		trafficProgress.Update(formatTrafficProgress(count, config.RulesMax, 0, config.DurationSeconds))

		iperfResult, err := r.traffic.RunTCPWithProgress(
			ctx,
			config.ClientNamespace,
			config.ServerNamespace,
			config.ServerIP,
			config.Port,
			config.DurationSeconds,
			func(doneSeconds int, totalSeconds int) {
				trafficProgress.Update(formatTrafficProgress(count, config.RulesMax, doneSeconds, totalSeconds))
			},
		)
		if err != nil {
			trafficProgress.Done("")
			return nil, err
		}
		trafficProgress.Done(formatTrafficDone(count, config.RulesMax, config.DurationSeconds))

		result := store.Add(metrics.Result{
			SeriesName:            config.Name,
			Engine:                r.engine.Name(),
			Profile:               config.FilterProfile,
			RuleCount:             count,
			BitsPerSecondSent:     iperfResult.BitsPerSecondSent,
			BitsPerSecondReceived: iperfResult.BitsPerSecondReceived,
			BytesSent:             iperfResult.BytesSent,
			BytesReceived:         iperfResult.BytesReceived,
			Retransmits:           iperfResult.Retransmits,
			DurationSeconds:       iperfResult.DurationSeconds,
			CPUHostTotal:          iperfResult.CPUHostTotal,
			CPURemoteTotal:        iperfResult.CPURemoteTotal,
		})

		r.logger.Info(fmt.Sprintf(
			"Замер завершен: уровень правил %d из %d | скорость приема %s | падение от 0 правил %.2f%% | падение от прошлого шага %.2f%% | TCP повторы %d",
			count,
			config.RulesMax,
			metrics.FormatBPS(result.BitsPerSecondReceived),
			result.DegradationBaselinePercent,
			result.DegradationPreviousPercent,
			result.Retransmits,
		))

		previousCount = count
	}

	results := store.Results()
	reportPath := filepath.Join(config.ReportDir, config.Name+".json")
	if err := metrics.WriteJSON(reportPath, results); err != nil {
		return nil, err
	}

	r.logger.Info("Benchmark-серия завершена", "замеров", len(results))
	r.logger.Info("Отчет сохранен", "путь", reportPath)
	r.logger.Print(metrics.ReportTable(results))

	return results, nil
}

func (c SeriesConfig) withDefaults() SeriesConfig {
	if c.FilterProfilesPath == "" {
		c.FilterProfilesPath = "filters.yaml"
	}
	if c.Port == 0 {
		c.Port = 5201
	}
	if c.DurationSeconds == 0 {
		c.DurationSeconds = 10
	}
	if c.ReportDir == "" {
		c.ReportDir = "reports"
	}

	return c
}

func (c SeriesConfig) validate() error {
	if c.Name == "" {
		return fmt.Errorf("benchmark name not found")
	}
	if c.FilterProfile == "" {
		return fmt.Errorf("filter_profile not found")
	}
	if c.FilterProfilesPath == "" {
		return fmt.Errorf("filter_profiles_path not found")
	}
	if c.ClientNamespace == "" {
		return fmt.Errorf("client_namespace not found")
	}
	if c.ServerNamespace == "" {
		return fmt.Errorf("server_namespace not found")
	}
	if c.ServerIP == "" {
		return fmt.Errorf("server_ip not found")
	}
	if c.DurationSeconds <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if c.RulesStep <= 0 {
		return fmt.Errorf("rules_step must be positive")
	}
	if c.RulesMax < c.RulesStart {
		return fmt.Errorf("rules_max must be greater than or equal to rules_start")
	}

	return nil
}

func formatRulesProgress(
	engineName string,
	currentRules int,
	maxRules int,
	createdRules int,
) string {
	return fmt.Sprintf(
		"Подготовка правил: engine %s | уровень %d из %d | создано %d правил для текущего замера | осталось создать %d | сейчас: применение правил",
		engineName,
		currentRules,
		maxRules,
		createdRules,
		currentRules-createdRules,
	)
}

func formatRulesDone(
	engineName string,
	currentRules int,
	maxRules int,
) string {
	return fmt.Sprintf(
		"Правила готовы: engine %s | уровень %d из %d | создано %d правил для текущего замера",
		engineName,
		currentRules,
		maxRules,
		currentRules,
	)
}

func formatTrafficProgress(
	rules int,
	maxRules int,
	doneSeconds int,
	totalSeconds int,
) string {
	return fmt.Sprintf(
		"Замер трафика: уровень правил %d из %d | прошло %d/%d сек | осталось %d сек | сейчас: iperf3 передает данные",
		rules,
		maxRules,
		doneSeconds,
		totalSeconds,
		totalSeconds-doneSeconds,
	)
}

func formatTrafficDone(
	rules int,
	maxRules int,
	totalSeconds int,
) string {
	return fmt.Sprintf(
		"Замер трафика завершен: уровень правил %d из %d | длительность %d сек",
		rules,
		maxRules,
		totalSeconds,
	)
}
