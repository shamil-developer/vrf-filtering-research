package handlers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
	"github.com/shamil-developer/vrf-filtering-research/internal/network"
)

func networkTool(
	tools *chain.Tools,
) (*network.Tool, error) {
	tool, ok := tools.Get("network").(*network.Tool)
	if !ok || tool == nil {
		return nil, fmt.Errorf("network tool not found")
	}

	return tool, nil
}

func logger(
	tools *chain.Tools,
) *log.Logger {
	logger, ok := tools.Get("logger").(*log.Logger)
	if ok && logger != nil {
		return logger
	}

	return log.NewWithOptions(io.Discard, log.Options{})
}

func optionalString(
	request map[string]any,
	name string,
) string {
	value, _ := request[name].(string)

	return value
}

func requiredString(
	request map[string]any,
	name string,
) (string, error) {
	value, ok := request[name].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("%s not found", name)
	}

	return value, nil
}

func optionalInt(
	request map[string]any,
	name string,
) (int, error) {
	value, ok := request[name]
	if !ok {
		return 0, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, nil
	case int64:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case string:
		if typed == "" {
			return 0, nil
		}

		parsed, err := strconv.Atoi(typed)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", name, err)
		}

		return parsed, nil
	default:
		return 0, fmt.Errorf("%s has unsupported type %T", name, value)
	}
}

func displayNamespace(
	namespace string,
) string {
	if namespace == "" {
		return "root"
	}

	return namespace
}

func displayDefault(
	value string,
) string {
	if value == "" {
		return "default"
	}

	return value
}

func displayEmpty(
	value string,
) string {
	if value == "" {
		return "-"
	}

	return value
}

func displayTable(
	table int,
) string {
	if table == 0 {
		return "main"
	}

	return strconv.Itoa(table)
}
