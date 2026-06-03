package ebpf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/shamil-developer/vrf-filtering-research/internal/command"
	"github.com/shamil-developer/vrf-filtering-research/internal/engine"
	"github.com/shamil-developer/vrf-filtering-research/internal/filterprofile"
)

type Manager struct {
	runner *command.Runner
}

func NewManager(
	runner *command.Runner,
) *Manager {
	return &Manager{
		runner: runner,
	}
}

func (m *Manager) Name() string {
	return "tc+ebpf"
}

func (m *Manager) ApplyRulesWithProgress(
	ctx context.Context,
	profile filterprofile.Profile,
	count int,
	progress engine.ProgressFunc,
) error {
	if err := m.Reset(ctx, profile); err != nil {
		return err
	}

	if count == 0 {
		if progress != nil {
			progress(0, 0)
		}
		return nil
	}

	rules := profile.GenerateRules(count)
	interfaces := interfacesForRules(rules)
	tcRules, ebpfRules := splitRules(rules)

	for _, iface := range interfaces {
		if _, err := m.runner.Run(ctx, "tc", "qdisc", "replace", "dev", iface, "clsact"); err != nil {
			return fmt.Errorf("create clsact on %s: %w", iface, err)
		}
	}

	applied := 0
	for _, rule := range tcRules {
		if err := m.applyTCRule(ctx, rule); err != nil {
			return err
		}
		applied++
		if progress != nil {
			progress(applied, count+1)
		}
	}

	if len(ebpfRules) > 0 {
		if err := m.applyEBPFRules(ctx, ebpfRules, interfaces); err != nil {
			return err
		}
		applied = count
		if progress != nil {
			progress(applied, count+1)
		}
	}

	return nil
}

func (m *Manager) applyEBPFRules(
	ctx context.Context,
	rules []filterprofile.GeneratedRule,
	interfaces []string,
) error {
	ifIndexes, err := m.ifIndexes(ctx, interfaces)
	if err != nil {
		return err
	}
	source := renderProgram(rules, ifIndexes)

	workDir, err := os.MkdirTemp("", "vrf-filtering-ebpf-*")
	if err != nil {
		return fmt.Errorf("create ebpf temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	sourcePath := filepath.Join(workDir, "rules.c")
	objectPath := filepath.Join(workDir, "rules.o")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		return fmt.Errorf("write ebpf source: %w", err)
	}

	targetArch, err := m.targetArch(ctx)
	if err != nil {
		return err
	}
	includePath, err := m.multiarchIncludePath(ctx)
	if err != nil {
		return err
	}

	if _, err := m.runner.Run(
		ctx,
		"clang",
		"-O2",
		"-g",
		"-target",
		"bpf",
		"-D__TARGET_ARCH_"+targetArch,
		"-I",
		includePath,
		"-c",
		sourcePath,
		"-o",
		objectPath,
	); err != nil {
		return fmt.Errorf("compile ebpf rules: %w", err)
	}

	for _, iface := range interfaces {
		if _, err := m.runner.Run(
			ctx,
			"tc",
			"filter",
			"replace",
			"dev",
			iface,
			"ingress",
			"bpf",
			"direct-action",
			"obj",
			objectPath,
			"sec",
			"classifier",
		); err != nil {
			return fmt.Errorf("attach ebpf rules to %s: %w", iface, err)
		}
	}

	return nil
}

func (m *Manager) applyTCRule(
	ctx context.Context,
	rule filterprofile.GeneratedRule,
) error {
	iface := iifFromRule(rule)
	if iface == "" {
		return fmt.Errorf("tc rule %d has no input interface", rule.Index)
	}

	args, ok := tcFlowerArgs(rule)
	if !ok {
		return fmt.Errorf("rule %d cannot be rendered as tc flower", rule.Index)
	}

	commandArgs := append([]string{
		"filter",
		"add",
		"dev",
		iface,
		"ingress",
		"protocol",
		"ip",
		"pref",
		fmt.Sprintf("%d", rule.Index),
		"flower",
	}, args...)

	if _, err := m.runner.Run(ctx, "tc", commandArgs...); err != nil {
		return fmt.Errorf("add tc flower rule %d on %s: %w", rule.Index, iface, err)
	}

	return nil
}

func (m *Manager) targetArch(
	ctx context.Context,
) (string, error) {
	result, err := m.runner.Run(ctx, "uname", "-m")
	if err != nil {
		return "", fmt.Errorf("detect machine architecture: %w", err)
	}

	switch strings.TrimSpace(result.Stdout) {
	case "x86_64", "amd64":
		return "x86", nil
	case "aarch64", "arm64":
		return "arm64", nil
	default:
		return "x86", nil
	}
}

func (m *Manager) multiarchIncludePath(
	ctx context.Context,
) (string, error) {
	result, err := m.runner.Run(ctx, "gcc", "-print-multiarch")
	if err != nil {
		return "", fmt.Errorf("detect multiarch include path: %w", err)
	}

	multiarch := strings.TrimSpace(result.Stdout)
	if multiarch == "" {
		return "/usr/include", nil
	}

	return filepath.Join("/usr/include", multiarch), nil
}

func (m *Manager) ifIndexes(
	ctx context.Context,
	interfaces []string,
) (map[string]int, error) {
	result := make(map[string]int, len(interfaces))
	for _, iface := range interfaces {
		output, err := m.runner.Run(ctx, "ip", "-o", "link", "show", "dev", iface)
		if err != nil {
			return nil, fmt.Errorf("get ifindex for %s: %w", iface, err)
		}

		indexText := strings.TrimSpace(strings.SplitN(output.Stdout, ":", 2)[0])
		var index int
		if _, err := fmt.Sscanf(indexText, "%d", &index); err != nil {
			return nil, fmt.Errorf("parse ifindex for %s from %q: %w", iface, output.Stdout, err)
		}
		result[iface] = index
	}

	return result, nil
}

func (m *Manager) Reset(
	ctx context.Context,
	profile filterprofile.Profile,
) error {
	interfaces := interfacesFromTopology(profile.Topology)
	for _, iface := range interfaces {
		_, _ = m.runner.Run(ctx, "tc", "qdisc", "del", "dev", iface, "clsact")
	}

	return nil
}

func interfacesForRules(
	rules []filterprofile.GeneratedRule,
) []string {
	seen := make(map[string]struct{})
	for _, rule := range rules {
		if rule.IsFinal {
			continue
		}
		iface := iifFromRule(rule)
		if iface != "" {
			seen[iface] = struct{}{}
		}
	}

	return sortedKeys(seen)
}

func interfacesFromTopology(
	topology filterprofile.Topology,
) []string {
	seen := make(map[string]struct{})
	for _, direction := range topology.Directions {
		if direction.IIF != "" {
			seen[direction.IIF] = struct{}{}
		}
	}

	return sortedKeys(seen)
}

func sortedKeys(
	values map[string]struct{},
) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)

	return result
}

func iifFromCommand(
	command string,
) string {
	const marker = `iifname "`
	start := strings.Index(command, marker)
	if start < 0 {
		return ""
	}
	start += len(marker)
	end := strings.Index(command[start:], `"`)
	if end < 0 {
		return ""
	}

	return command[start : start+end]
}

func iifFromRule(
	rule filterprofile.GeneratedRule,
) string {
	if rule.IIF != "" {
		return rule.IIF
	}

	return iifFromCommand(rule.Command)
}

func splitRules(
	rules []filterprofile.GeneratedRule,
) ([]filterprofile.GeneratedRule, []filterprofile.GeneratedRule) {
	tcRules := make([]filterprofile.GeneratedRule, 0, len(rules))
	ebpfRules := make([]filterprofile.GeneratedRule, 0, len(rules))

	for _, rule := range rules {
		if rule.IsFinal {
			continue
		}
		if _, ok := tcFlowerArgs(rule); ok {
			tcRules = append(tcRules, rule)
			continue
		}
		ebpfRules = append(ebpfRules, rule)
	}

	return tcRules, ebpfRules
}

func tcFlowerArgs(
	rule filterprofile.GeneratedRule,
) ([]string, bool) {
	if iifFromRule(rule) == "" {
		return nil, false
	}
	actionArgs, ok := tcActionArgs(rule)
	if !ok {
		return nil, false
	}

	args := make([]string, 0, 16)
	command := rule.Command

	if value := firstMatch(ipSaddrPattern, command); value != "" {
		args = append(args, "src_ip", value)
	}
	if value := firstMatch(ipDaddrPattern, command); value != "" {
		args = append(args, "dst_ip", value)
	}
	if strings.Contains(command, "ip protocol tcp") || strings.Contains(command, "meta l4proto tcp") ||
		tcpSportPattern.MatchString(command) || tcpDportPattern.MatchString(command) ||
		strings.Contains(command, "tcp flags") {
		args = append(args, "ip_proto", "tcp")
	}
	if strings.Contains(command, "ip protocol udp") || strings.Contains(command, "meta l4proto udp") ||
		udpSportPattern.MatchString(command) || udpDportPattern.MatchString(command) {
		args = append(args, "ip_proto", "udp")
	}
	if value := firstMatch(tcpSportPattern, command); value != "" {
		args = append(args, "src_port", value)
	}
	if value := firstMatch(tcpDportPattern, command); value != "" {
		args = append(args, "dst_port", value)
	}
	if value := firstMatch(udpSportPattern, command); value != "" {
		args = append(args, "src_port", value)
	}
	if value := firstMatch(udpDportPattern, command); value != "" {
		args = append(args, "dst_port", value)
	}

	if hasUnsupportedTCMatch(command) {
		return nil, false
	}
	if len(args) == 0 {
		return nil, false
	}

	args = append(args, actionArgs...)

	return args, true
}

func RuleImplementation(
	rule filterprofile.GeneratedRule,
) string {
	if args, ok := tcFlowerArgs(rule); ok {
		return fmt.Sprintf(
			"tc filter add dev %s ingress protocol ip pref %d flower %s",
			iifFromRule(rule),
			rule.Index,
			strings.Join(args, " "),
		)
	}

	return fmt.Sprintf(
		"eBPF classifier через tc clsact: %s; действие %s",
		rule.Rule,
		actionFromRule(rule),
	)
}

func RuleBackend(
	rule filterprofile.GeneratedRule,
) string {
	if _, ok := tcFlowerArgs(rule); ok {
		return "tc flower"
	}

	return "eBPF"
}

func tcActionArgs(
	rule filterprofile.GeneratedRule,
) ([]string, bool) {
	switch actionFromRule(rule) {
	case "counter", "accept":
		return []string{"action", "pass"}, true
	case "drop":
		return []string{"action", "drop"}, true
	default:
		return nil, false
	}
}

func hasUnsupportedTCMatch(
	command string,
) bool {
	unsupportedFragments := []string{
		"ct ",
		"fib ",
		"ipsec ",
		"socket ",
		"rt ip ",
		"rt classid",
		"numgen ",
		"ip frag-off",
		"ip dscp",
		"ip ttl",
		"ip length",
		"meta length",
		"meta priority",
		"meta skuid",
		"meta skgid",
		"icmp ",
		"limit rate",
		"vmap",
		" map ",
		"{",
		"}",
	}
	for _, fragment := range unsupportedFragments {
		if strings.Contains(command, fragment) {
			return true
		}
	}
	if strings.Contains(command, "meta mark ") && !strings.Contains(command, "meta mark set") {
		return true
	}

	return false
}

func firstMatch(
	pattern *regexp.Regexp,
	command string,
) string {
	match := pattern.FindStringSubmatch(command)
	if len(match) < 2 {
		return ""
	}

	return match[1]
}
