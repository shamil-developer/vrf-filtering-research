package metrics

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type CompareRow struct {
	RuleCount           int
	LeftEngine          string
	RightEngine         string
	LeftReceivedBPS     float64
	RightReceivedBPS    float64
	ReceivedDeltaPct    float64
	LeftRetransmits     int
	RightRetransmits    int
	LeftCPUHostTotal    float64
	RightCPUHostTotal   float64
	LeftCPURemoteTotal  float64
	RightCPURemoteTotal float64
	Winner              string
}

func CompareResults(
	left []Result,
	right []Result,
) []CompareRow {
	rightByRules := make(map[int]Result, len(right))
	for _, result := range right {
		rightByRules[result.RuleCount] = result
	}

	rows := make([]CompareRow, 0, len(left))
	for _, leftResult := range left {
		rightResult, ok := rightByRules[leftResult.RuleCount]
		if !ok {
			continue
		}

		leftEngine := leftResult.Engine
		if leftEngine == "" {
			leftEngine = leftResult.SeriesName
		}
		rightEngine := rightResult.Engine
		if rightEngine == "" {
			rightEngine = rightResult.SeriesName
		}

		delta := percentDelta(rightResult.BitsPerSecondReceived, leftResult.BitsPerSecondReceived)
		winner := leftEngine
		if rightResult.BitsPerSecondReceived > leftResult.BitsPerSecondReceived {
			winner = rightEngine
		}
		if rightResult.BitsPerSecondReceived == leftResult.BitsPerSecondReceived {
			winner = "равно"
		}

		rows = append(rows, CompareRow{
			RuleCount:           leftResult.RuleCount,
			LeftEngine:          leftEngine,
			RightEngine:         rightEngine,
			LeftReceivedBPS:     leftResult.BitsPerSecondReceived,
			RightReceivedBPS:    rightResult.BitsPerSecondReceived,
			ReceivedDeltaPct:    delta,
			LeftRetransmits:     leftResult.Retransmits,
			RightRetransmits:    rightResult.Retransmits,
			LeftCPUHostTotal:    leftResult.CPUHostTotal,
			RightCPUHostTotal:   rightResult.CPUHostTotal,
			LeftCPURemoteTotal:  leftResult.CPURemoteTotal,
			RightCPURemoteTotal: rightResult.CPURemoteTotal,
			Winner:              winner,
		})
	}

	return rows
}

func CompareTable(
	rows []CompareRow,
) string {
	if len(rows) == 0 {
		return "Нет общих шагов для сравнения."
	}

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	tableRows := make([][]string, 0, len(rows))
	for _, row := range rows {
		tableRows = append(tableRows, []string{
			fmt.Sprintf("%d", row.RuleCount),
			FormatBPS(row.LeftReceivedBPS),
			FormatBPS(row.RightReceivedBPS),
			formatPercent(row.ReceivedDeltaPct),
			fmt.Sprintf("%d / %d", row.LeftRetransmits, row.RightRetransmits),
			fmt.Sprintf("%s / %s", formatPercent(row.LeftCPUHostTotal), formatPercent(row.RightCPUHostTotal)),
			fmt.Sprintf("%s / %s", formatPercent(row.LeftCPURemoteTotal), formatPercent(row.RightCPURemoteTotal)),
			row.Winner,
		})
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.Padding(0, 1)
			}

			return cellStyle
		}).
		Headers(
			"Правил",
			rows[0].LeftEngine+" прием",
			rows[0].RightEngine+" прием",
			rows[0].RightEngine+" к "+rows[0].LeftEngine,
			"TCP повторы",
			"CPU клиент",
			"CPU сервер",
			"Быстрее",
		).
		Rows(tableRows...)

	return "\n" + t.Render()
}
