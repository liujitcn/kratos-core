package locale

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/language"
)

// Candidates 返回按 q 权重和语言层级排列的语言候选，不包含默认语言。
func Candidates(value string) []string {
	preferences := parsePreferences(value)
	result := make([]string, 0, len(preferences)*2)
	seen := make(map[string]struct{}, len(preferences)*2)
	for _, preference := range preferences {
		result = appendCandidate(result, seen, preference.locale)
		currentLocale := preference.locale
		for {
			index := strings.LastIndexByte(currentLocale, '-')
			if index < 0 {
				break
			}
			currentLocale = currentLocale[:index]
			result = appendCandidate(result, seen, currentLocale)
		}
	}
	return result
}

// Normalize 规范化语言标识，兼容下划线和大小写差异。
func Normalize(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, ','); index >= 0 {
		value = value[:index]
	}
	if index := strings.IndexByte(value, ';'); index >= 0 {
		value = value[:index]
	}
	value = strings.ReplaceAll(strings.TrimSpace(value), "_", "-")
	if value == "" {
		return ""
	}
	if tag, err := language.Parse(value); err == nil {
		return tag.String()
	}
	return strings.ToLower(value)
}

type preference struct {
	locale  string
	quality float64
	order   int
}

// parsePreferences 解析 Accept-Language 风格的语言偏好列表。
func parsePreferences(value string) []preference {
	parts := strings.Split(value, ",")
	preferences := make([]preference, 0, len(parts))
	for order, part := range parts {
		parameters := strings.Split(part, ";")
		locale := Normalize(parameters[0])
		if locale == "" || locale == "*" {
			continue
		}
		quality := 1.0
		for _, parameter := range parameters[1:] {
			name, rawValue, found := strings.Cut(parameter, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(name), "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(rawValue), 64)
			if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) || parsed < 0 || parsed > 1 {
				quality = 0
				break
			}
			quality = parsed
		}
		if quality == 0 {
			continue
		}
		preferences = append(preferences, preference{locale: locale, quality: quality, order: order})
	}
	sort.SliceStable(preferences, func(left, right int) bool {
		if preferences[left].quality == preferences[right].quality {
			return preferences[left].order < preferences[right].order
		}
		return preferences[left].quality > preferences[right].quality
	})
	return preferences
}

// appendCandidate 追加尚未出现的语言标识。
func appendCandidate(result []string, seen map[string]struct{}, value string) []string {
	if value == "" {
		return result
	}
	if _, exists := seen[value]; exists {
		return result
	}
	seen[value] = struct{}{}
	return append(result, value)
}
