// Package flagutil deep-merges unrecognized --key=value CLI args into a ConfigFile.
package flagutil

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/das6ng/xlsxcfg"
	"gopkg.in/yaml.v3"
)

// ApplyOverrides deep-merges unrecognized --key=value args into cfg via YAML round-trip.
func ApplyOverrides(args []string, knownFlags map[string]bool, cfg *xlsxcfg.ConfigFile) error {
	overrides := map[string]string{}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		parts := strings.SplitN(arg, "=", 2)
		if knownFlags[parts[0]] {
			continue
		}
		if len(parts) == 2 {
			overrides[strings.TrimPrefix(parts[0], "--")] = parts[1]
		}
	}
	if len(overrides) == 0 {
		return nil
	}

	yamlBuf, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config for override: %w", err)
	}
	var yamlMap map[string]any
	if err := yaml.Unmarshal(yamlBuf, &yamlMap); err != nil {
		return fmt.Errorf("unmarshal config for override: %w", err)
	}

	for key, val := range overrides {
		DeepMerge(yamlMap, key, val)
	}

	mergedBuf, err := yaml.Marshal(yamlMap)
	if err != nil {
		return fmt.Errorf("marshal merged config: %w", err)
	}
	if err := yaml.Unmarshal(mergedBuf, cfg); err != nil {
		return fmt.Errorf("unmarshal merged config: %w", err)
	}
	return nil
}

// DeepMerge sets a value at a dot-separated key path in a nested map.
func DeepMerge(m map[string]any, key string, value string) {
	parts := strings.Split(key, ".")
	current := m
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		sub, ok := current[part]
		if !ok {
			sub = map[string]any{}
			current[part] = sub
		}
		if subMap, ok := sub.(map[string]any); ok {
			current = subMap
		} else {
			newMap := map[string]any{}
			current[part] = newMap
			current = newMap
		}
	}
	current[parts[len(parts)-1]] = ParseValue(value)
}

// ParseValue converts a string to the appropriate Go type (bool, int, []any, or string).
func ParseValue(s string) any {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.Trim(s, "[]")
		items := strings.Split(inner, ",")
		result := make([]any, len(items))
		for i, item := range items {
			result[i] = ParseValue(strings.TrimSpace(item))
		}
		return result
	}
	return s
}
