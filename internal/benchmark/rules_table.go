package benchmark

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/shamil-developer/vrf-filtering-research/internal/ebpf"
	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
)

func renderAddedRulesTable(
	profile filterprofile.Profile,
	engineName string,
	previousCount int,
	currentCount int,
	maxCount int,
) string {
	if currentCount <= previousCount {
		return ""
	}

	allRules := profile.GenerateRules(currentCount)
	rows := make([][]string, 0, currentCount-previousCount)

	for _, rule := range allRules {
		if rule.IsFinal {
			continue
		}
		if rule.Index <= previousCount || rule.Index > currentCount {
			continue
		}

		rows = append(rows, []string{
			strconv.Itoa(rule.Index),
			rule.Segment,
			rule.Direction,
			rule.Type,
			rule.Action,
			ruleImplementation(engineName, rule),
			rule.Explanation,
		})
	}

	if len(rows) == 0 {
		return ""
	}

	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	cellStyle := lipgloss.NewStyle().Padding(0, 1)

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(borderStyle).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle.Padding(0, 1)
			}

			return cellStyle
		}).
		Headers("№", "Сегмент", "Направление", "Тип правила", "Действие", "Реализация", "Объяснение").
		Rows(rows...)

	return fmt.Sprintf(
		"\nДобавлены новые правила: %d-%d из %d | движок: %s\n%s",
		previousCount+1,
		currentCount,
		maxCount,
		engineDisplayName(engineName),
		t.Render(),
	)
}

func ruleImplementation(
	engineName string,
	rule filterprofile.GeneratedRule,
) string {
	switch engineName {
	case "tc+ebpf":
		return ebpf.RuleImplementation(rule)
	case "nft":
		return rule.Command
	default:
		return rule.Command
	}
}

func engineDisplayName(
	engineName string,
) string {
	switch engineName {
	case "tc+ebpf":
		return "tc+eBPF"
	case "nft":
		return "nftables"
	default:
		return engineName
	}
}
