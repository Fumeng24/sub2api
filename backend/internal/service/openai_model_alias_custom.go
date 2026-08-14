package service

import "strings"

func normalizeKnownOpenAICodexModelCustom(normalized string) (string, bool) {
	key := codexModelLookupKey(normalized)
	for _, item := range codexVersionModelPrefixes {
		if key == item.prefix {
			return item.target, true
		}
		suffix, ok := strings.CutPrefix(key, item.prefix+"-")
		if ok && isKnownCodexModelSuffix(suffix) {
			return item.target, true
		}
	}
	return "", true
}

// normalizeOpenAIBillingCodexModel maps known aliases to the pricing key while
// rejecting unknown future model names instead of charging an arbitrary rate.
func normalizeOpenAIBillingCodexModel(model string) string {
	if normalized := normalizeKnownOpenAICodexModel(model); normalized != "" {
		return normalized
	}

	canonical := canonicalizeOpenAIModelAliasSpelling(model)
	if canonical == "" {
		return ""
	}
	switch {
	case strings.Contains(canonical, "gpt-5.6-sol"):
		return "gpt-5.6-sol"
	case strings.Contains(canonical, "gpt-5.6-terra"):
		return "gpt-5.6-terra"
	case strings.Contains(canonical, "gpt-5.6-luna"):
		return "gpt-5.6-luna"
	case canonical == "gpt-5.6":
		return "gpt-5.6-sol"
	case strings.HasPrefix(canonical, "gpt-5.6-"):
		suffix := strings.TrimPrefix(canonical, "gpt-5.6-")
		if suffix == "max" || isKnownCodexModelSuffix(suffix) {
			return "gpt-5.6-sol"
		}
		return ""
	case strings.Contains(canonical, "gpt-5.5-pro"):
		return "gpt-5.5-pro"
	case strings.Contains(canonical, "gpt-5.5"):
		return "gpt-5.5"
	case strings.Contains(canonical, "gpt-5.4-mini"):
		return "gpt-5.4-mini"
	case strings.Contains(canonical, "gpt-5.4-nano"):
		return "gpt-5.4-nano"
	case strings.Contains(canonical, "gpt-5.4"):
		return "gpt-5.4"
	case strings.Contains(canonical, "gpt-5.3-codex-spark"):
		return "gpt-5.3-codex-spark"
	case strings.Contains(canonical, "gpt-5.3-codex"):
		return "gpt-5.3-codex"
	case strings.Contains(canonical, "gpt-5.3"):
		return "gpt-5.3-codex"
	case strings.Contains(canonical, "gpt-5.2-codex"), strings.Contains(canonical, "gpt-5.2"):
		return "gpt-5.2"
	case strings.Contains(canonical, "gpt-5.1-codex"):
		return "gpt-5.3-codex"
	case strings.Contains(canonical, "gpt-5.1"):
		return "gpt-5.4"
	case strings.Contains(canonical, "codex-mini-latest"),
		strings.Contains(canonical, "gpt-5-codex"),
		strings.Contains(canonical, "codex"):
		return "gpt-5.3-codex"
	case strings.Contains(canonical, "gpt-5"):
		return "gpt-5.5"
	default:
		return ""
	}
}
