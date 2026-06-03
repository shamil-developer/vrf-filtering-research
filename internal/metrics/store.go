package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type Result struct {
	SeriesName                 string
	Engine                     string
	Profile                    string
	RuleCount                  int
	BitsPerSecondSent          float64
	BitsPerSecondReceived      float64
	BytesSent                  float64
	BytesReceived              float64
	Retransmits                int
	DurationSeconds            float64
	CPUHostTotal               float64
	CPURemoteTotal             float64
	DeltaBaselinePercent       float64
	DeltaPreviousPercent       float64
	DegradationBaselinePercent float64
	DegradationPreviousPercent float64
}

type Store struct {
	results []Result
}

func NewStore() *Store {
	return &Store{}
}

func (s *Store) Add(
	result Result,
) Result {
	if len(s.results) == 0 {
		s.results = append(s.results, result)
		return result
	}

	baseline := s.results[0]
	previous := s.results[len(s.results)-1]

	result.DeltaBaselinePercent = percentDelta(result.BitsPerSecondReceived, baseline.BitsPerSecondReceived)
	result.DeltaPreviousPercent = percentDelta(result.BitsPerSecondReceived, previous.BitsPerSecondReceived)
	result.DegradationBaselinePercent = -result.DeltaBaselinePercent
	result.DegradationPreviousPercent = -result.DeltaPreviousPercent

	s.results = append(s.results, result)

	return result
}

func (s *Store) Results() []Result {
	results := make([]Result, len(s.results))
	copy(results, s.results)

	return results
}

func ReportTable(
	results []Result,
) string {
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	cellStyle := lipgloss.NewStyle().Padding(0, 1)
	negativeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Padding(0, 1)
	positiveStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Padding(0, 1)

	rows := make([][]string, 0, len(results))
	for _, result := range results {
		rows = append(rows, []string{
			fmt.Sprintf("%d", result.RuleCount),
			FormatBPS(result.BitsPerSecondReceived),
			FormatBPS(result.BitsPerSecondSent),
			formatPercent(result.DegradationBaselinePercent),
			formatPercent(result.DegradationPreviousPercent),
			fmt.Sprintf("%d", result.Retransmits),
			formatPercent(result.CPUHostTotal),
			formatPercent(result.CPURemoteTotal),
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.Padding(0, 1)
			}

			if col == 3 || col == 4 {
				value := results[row].DegradationBaselinePercent
				if col == 4 {
					value = results[row].DegradationPreviousPercent
				}

				if value > 0 {
					return negativeStyle
				}
				if value < 0 {
					return positiveStyle
				}
			}

			return cellStyle
		}).
		Headers(
			"Правил",
			"Скорость приема",
			"Скорость отправки",
			"Падение от 0 правил",
			"Падение от прошлого шага",
			"TCP повторы",
			"CPU клиента",
			"CPU сервера",
		).
		Rows(rows...)

	return "\n" + t.Render() + "\n\n" + reportLegend()
}

func WriteJSON(
	path string,
	results []Result,
) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create report directory: %w", err)
	}

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metrics report: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write metrics report %q: %w", path, err)
	}

	return nil
}

func ReadJSON(
	path string,
) ([]Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read metrics report %q: %w", path, err)
	}

	var results []Result
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, fmt.Errorf("parse metrics report %q: %w", path, err)
	}

	return results, nil
}

func percentDelta(
	current float64,
	base float64,
) float64 {
	if base == 0 {
		return 0
	}

	return (current - base) / base * 100
}

func FormatBPS(
	value float64,
) string {
	units := []string{"bit/s", "Kbit/s", "Mbit/s", "Gbit/s"}
	unit := 0

	for value >= 1000 && unit < len(units)-1 {
		value /= 1000
		unit++
	}

	return fmt.Sprintf("%.2f %s", value, units[unit])
}

func formatPercent(
	value float64,
) string {
	return fmt.Sprintf("%.2f%%", value)
}

func reportLegend() string {
	lines := []string{
		"Что значит каждый столбец:",
		"- Правил — сколько правил текущего engine было создано перед замером.",
		"- Скорость приема — сколько данных принял сервер по результату iperf3.",
		"- Скорость отправки — сколько данных отправил клиент по результату iperf3.",
		"- Падение от 0 правил — насколько скорость приема стала ниже относительно замера без правил.",
		"- Падение от прошлого шага — насколько скорость приема изменилась относительно предыдущего количества правил.",
		"- TCP повторы — сколько TCP retransmits зафиксировал iperf3; чем больше, тем нестабильнее передача.",
		"- CPU клиента — загрузка CPU на стороне iperf3 client.",
		"- CPU сервера — загрузка CPU на стороне iperf3 server.",
		"- Если значение падения отрицательное, значит на этом шаге скорость выросла относительно сравниваемого замера.",
	}

	return strings.Join(lines, "\n")
}
