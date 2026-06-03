package filterprofile

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Topology Topology           `yaml:"topology"`
	Actions  []Action           `yaml:"actions"`
	Profiles map[string]Profile `yaml:"profiles"`
}

type Topology struct {
	Bridge     string      `yaml:"bridge"`
	VRF        string      `yaml:"vrf"`
	Directions []Direction `yaml:"directions"`
}

type Direction struct {
	Name        string `yaml:"name"`
	Segment     string `yaml:"segment"`
	IIF         string `yaml:"iif"`
	OIF         string `yaml:"oif"`
	Explanation string `yaml:"explanation"`
}

type Profile struct {
	Family              string   `yaml:"family"`
	Table               string   `yaml:"table"`
	Chain               string   `yaml:"chain"`
	Hook                string   `yaml:"hook"`
	Priority            int      `yaml:"priority"`
	Policy              string   `yaml:"policy"`
	Topology            Topology `yaml:"topology"`
	Actions             []Action `yaml:"actions"`
	Rules               []Rule   `yaml:"rules"`
	Template            string   `yaml:"template"`
	FinalRule           string   `yaml:"final_rule"`
	MatchEvery          int      `yaml:"match_every"`
	MatchTemplate       string   `yaml:"match_template"`
	Description         string   `yaml:"description"`
	MatchDescription    string   `yaml:"match_description"`
	NonMatchDescription string   `yaml:"non_match_description"`
}

type Rule struct {
	Name             string `yaml:"name"`
	Template         string `yaml:"template"`
	MatchTemplate    string `yaml:"match_template"`
	Explanation      string `yaml:"explanation"`
	MatchExplanation string `yaml:"match_explanation"`
}

type Action struct {
	Name             string `yaml:"name"`
	Template         string `yaml:"template"`
	MatchTemplate    string `yaml:"match_template"`
	Explanation      string `yaml:"explanation"`
	MatchExplanation string `yaml:"match_explanation"`
}

type RuleContext struct {
	Index     int
	Topology  Topology
	Direction Direction
	Action    Action
}

type GeneratedRule struct {
	Index       int
	Type        string
	Segment     string
	Direction   string
	IIF         string
	OIF         string
	Action      string
	Rule        string
	Command     string
	Explanation string
	IsFinal     bool
}

func LoadFile(
	path string,
) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read filter profiles %q: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse filter profiles %q: %w", path, err)
	}

	return &config, nil
}

func (c *Config) Get(
	name string,
) (Profile, error) {
	if c == nil {
		return Profile{}, fmt.Errorf("filter profiles not loaded")
	}

	profile, ok := c.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("filter profile %q not found", name)
	}

	if profile.Family == "" {
		profile.Family = "inet"
	}
	if profile.Table == "" {
		profile.Table = "bench"
	}
	if profile.Chain == "" {
		profile.Chain = "forward"
	}
	if profile.Hook == "" {
		profile.Hook = "forward"
	}
	if profile.Policy == "" {
		profile.Policy = "accept"
	}
	if profile.FinalRule == "" {
		profile.FinalRule = "accept"
	}
	if len(profile.Topology.Directions) == 0 {
		profile.Topology = c.Topology
	}
	profile.Topology = profile.Topology.withDefaults()
	if len(profile.Actions) == 0 {
		profile.Actions = c.Actions
	}
	if len(profile.Actions) == 0 {
		profile.Actions = defaultActions()
	}
	if len(profile.Rules) == 0 && profile.Template == "" {
		return Profile{}, fmt.Errorf("filter profile %q has no rules", name)
	}
	for index, rule := range profile.Rules {
		if rule.Template == "" {
			return Profile{}, fmt.Errorf("filter profile %q rule %d has empty template", name, index+1)
		}
	}

	return profile, nil
}

func (p Profile) GenerateRules(
	count int,
) []GeneratedRule {
	rules := make([]GeneratedRule, 0, count+1)

	for i := 1; i <= count; i++ {
		template, ruleType, explanation, matched := p.ruleForIndex(i)
		direction := p.directionForIndex(i)
		action := p.actionForIndex(i, matched)

		rule := renderTemplate(template, RuleContext{
			Index:     i,
			Topology:  p.Topology,
			Direction: direction,
			Action:    action,
		})
		rules = append(rules, GeneratedRule{
			Index:       i,
			Type:        ruleType,
			Segment:     direction.Segment,
			Direction:   direction.Name,
			IIF:         direction.IIF,
			OIF:         direction.OIF,
			Action:      action.Name,
			Rule:        rule,
			Command:     p.CommandForIndex(rule, i),
			Explanation: explanation,
		})
	}

	if p.FinalRule != "" {
		rules = append(rules, GeneratedRule{
			Index:       count + 1,
			Type:        "финальное правило",
			Rule:        p.FinalRule,
			Command:     p.Command(p.FinalRule),
			Explanation: "Финальное правило пропускает трафик, если предыдущие правила его не остановили.",
			IsFinal:     true,
		})
	}

	return rules
}

func (p Profile) ruleForIndex(
	index int,
) (string, string, string, bool) {
	if len(p.Rules) == 0 {
		return p.legacyRuleForIndex(index)
	}

	pattern := p.Rules[(index-1)%len(p.Rules)]
	template := pattern.Template
	ruleType := pattern.Name
	explanation := pattern.Explanation
	if explanation == "" {
		explanation = "Правило проверяет пакет, но параметры подобраны так, чтобы основной iperf-трафик прошел дальше."
	}

	if p.MatchEvery > 0 && index%p.MatchEvery == 0 && pattern.MatchTemplate != "" {
		template = pattern.MatchTemplate
		if pattern.MatchExplanation != "" {
			explanation = pattern.MatchExplanation
		}
		return template, ruleType, explanation, true
	}

	return template, ruleType, explanation, false
}

func (t Topology) withDefaults() Topology {
	if t.Bridge == "" {
		t.Bridge = "br0"
	}
	if t.VRF == "" {
		t.VRF = "vrf-blue"
	}
	if len(t.Directions) == 0 {
		t.Directions = []Direction{
			{
				Name:        "client-to-bridge",
				Segment:     "bridge",
				IIF:         "veth-c",
				OIF:         "veth-b",
				Explanation: "Участок от клиента к bridge br0.",
			},
			{
				Name:        "bridge-to-vrf",
				Segment:     "bridge-vrf",
				IIF:         "veth-b",
				OIF:         "veth-vrf",
				Explanation: "Переход между bridge br0 и VRF vrf-blue.",
			},
			{
				Name:        "vrf-to-server",
				Segment:     "vrf",
				IIF:         "veth-vrf",
				OIF:         "veth-s",
				Explanation: "Участок VRF vrf-blue к серверу.",
			},
			{
				Name:        "server-to-vrf",
				Segment:     "vrf",
				IIF:         "veth-s",
				OIF:         "veth-vrf",
				Explanation: "Обратный участок от сервера к VRF vrf-blue.",
			},
			{
				Name:        "vrf-to-bridge",
				Segment:     "bridge-vrf",
				IIF:         "veth-vrf",
				OIF:         "veth-b",
				Explanation: "Обратный переход между VRF vrf-blue и bridge br0.",
			},
			{
				Name:        "bridge-to-client",
				Segment:     "bridge",
				IIF:         "veth-b",
				OIF:         "veth-c",
				Explanation: "Обратный участок bridge br0 к клиенту.",
			},
		}
	}

	return t
}

func (p Profile) directionForIndex(
	index int,
) Direction {
	topology := p.Topology.withDefaults()
	return topology.Directions[(index-1)%len(topology.Directions)]
}

func (p Profile) actionForIndex(
	index int,
	matched bool,
) Action {
	if len(p.Actions) == 0 {
		p.Actions = defaultActions()
	}

	action := p.Actions[(index-1)%len(p.Actions)]
	if matched {
		template := action.MatchTemplate
		if template == "" {
			template = "counter"
		}
		explanation := action.MatchExplanation
		if explanation == "" {
			explanation = "Для совпадающего iperf-трафика используется безопасное действие, чтобы benchmark не оборвал поток."
		}

		return Action{
			Name:        action.Name,
			Template:    template,
			Explanation: explanation,
		}
	}

	if action.Template == "" {
		action.Template = "counter"
	}
	if action.Explanation == "" {
		action.Explanation = "Действие применяется к непопадающему правилу."
	}

	return action
}

func defaultActions() []Action {
	return []Action{
		{
			Name:             "counter",
			Template:         "counter",
			MatchTemplate:    "counter",
			Explanation:      "Считает пакеты, не меняя решение по трафику.",
			MatchExplanation: "Считает совпавший iperf-трафик.",
		},
		{
			Name:             "accept",
			Template:         "accept",
			MatchTemplate:    "accept",
			Explanation:      "Явно пропускает только непопадающий синтетический трафик.",
			MatchExplanation: "Явно пропускает совпавший iperf-трафик.",
		},
	}
}

func (p Profile) legacyRuleForIndex(
	index int,
) (string, string, string, bool) {
	template := p.Template
	ruleType := "не совпадает с iperf"
	explanation := p.NonMatchDescription
	if explanation == "" {
		explanation = "Правило создает нагрузку на nftables, но не совпадает с тестовым iperf-трафиком."
	}

	if p.MatchTemplate != "" && p.MatchEvery > 0 && index%p.MatchEvery == 0 {
		template = p.MatchTemplate
		ruleType = "совпадает с iperf"
		explanation = p.MatchDescription
		if explanation == "" {
			explanation = "Правило совпадает с тестовым iperf-трафиком."
		}
		return template, ruleType, explanation, true
	}

	return template, ruleType, explanation, false
}

func (p Profile) Command(
	rule string,
) string {
	return fmt.Sprintf(
		"nft add rule %s %s %s %s",
		p.Family,
		p.Table,
		p.Chain,
		rule,
	)
}

func (p Profile) CommandForIndex(
	rule string,
	index int,
) string {
	commentedRule := strings.Replace(
		rule,
		" counter",
		fmt.Sprintf(` counter comment "bench-rule-%d"`, index),
		1,
	)

	return p.Command(commentedRule)
}

func renderTemplate(
	template string,
	ctx RuleContext,
) string {
	a := (ctx.Index / 254) % 254
	b := ctx.Index % 254
	if b == 0 {
		b = 254
	}
	if a == 0 {
		a = 1
	}

	port := 10000 + (ctx.Index % 50000)
	length := 64 + (ctx.Index % 1400)
	ttl := 32 + (ctx.Index % 32)
	mark := ctx.Index % 1024
	zone := 1000 + (ctx.Index % 500)
	spi := 100000 + ctx.Index
	reqID := 200000 + ctx.Index
	path := fmt.Sprintf(
		`iifname "%s" oifname "%s"`,
		ctx.Direction.IIF,
		ctx.Direction.OIF,
	)

	replacements := map[string]string{
		"{index}":     fmt.Sprintf("%d", ctx.Index),
		"{bridge}":    ctx.Topology.Bridge,
		"{vrf}":       ctx.Topology.VRF,
		"{segment}":   ctx.Direction.Segment,
		"{direction}": ctx.Direction.Name,
		"{iif}":       ctx.Direction.IIF,
		"{oif}":       ctx.Direction.OIF,
		"{path}":      path,
		"{a}":         fmt.Sprintf("%d", a),
		"{b}":         fmt.Sprintf("%d", b),
		"{port}":      fmt.Sprintf("%d", port),
		"{length}": fmt.Sprintf(
			"%d",
			length,
		),
		"{ttl}":   fmt.Sprintf("%d", ttl),
		"{mark}":  fmt.Sprintf("%d", mark),
		"{zone}":  fmt.Sprintf("%d", zone),
		"{spi}":   fmt.Sprintf("%d", spi),
		"{reqid}": fmt.Sprintf("%d", reqID),
	}
	action := renderText(ctx.Action.Template, replacements)
	replacements["{action}"] = action

	result := renderText(template, replacements)
	if !strings.Contains(result, "iifname") && !strings.Contains(result, "oifname") {
		result = path + " " + result
	}
	if !strings.Contains(template, "{action}") {
		result = applyAction(result, action)
	}

	return result
}

func renderText(
	text string,
	replacements map[string]string,
) string {
	result := text
	for key, value := range replacements {
		result = strings.ReplaceAll(result, key, value)
	}

	return result
}

func applyAction(
	rule string,
	action string,
) string {
	if hasTerminalStatement(rule) {
		return rule
	}
	if strings.Contains(rule, " counter") {
		return strings.Replace(rule, " counter", " "+action, 1)
	}

	return rule + " " + action
}

func hasTerminalStatement(
	rule string,
) bool {
	terminalStatements := []string{
		" accept",
		" drop",
		" queue",
		" vmap ",
	}
	for _, statement := range terminalStatements {
		if strings.Contains(rule, statement) {
			return true
		}
	}

	return false
}
