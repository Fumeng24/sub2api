package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func extractCustomUpstreamModelIDs(body []byte) ([]string, bool, error) {
	var directIDs []string
	if err := json.Unmarshal(body, &directIDs); err == nil {
		return normalizeCustomUpstreamModelIDs(dedupeAndSortModelIDs(directIDs)), true, nil
	}

	var objectResponse map[string]json.RawMessage
	if err := json.Unmarshal(body, &objectResponse); err != nil {
		return nil, false, nil
	}
	containsStringArray := false
	models := make([]string, 0)
	for _, key := range []string{"data", "models"} {
		raw, ok := objectResponse[key]
		if !ok {
			continue
		}
		var ids []string
		if err := json.Unmarshal(raw, &ids); err == nil {
			containsStringArray = true
			models = append(models, ids...)
			continue
		}
		var entries []upstreamModelEntry
		if err := json.Unmarshal(raw, &entries); err != nil {
			if containsStringArray {
				return nil, true, fmt.Errorf("parse upstream model list %q: %w", key, err)
			}
			return nil, false, nil
		}
		for _, entry := range entries {
			models = append(models, upstreamModelEntryID(entry))
		}
	}
	if !containsStringArray {
		return nil, false, nil
	}
	return normalizeCustomUpstreamModelIDs(dedupeAndSortModelIDs(models)), true, nil
}

func normalizeCustomUpstreamModelIDs(models []string) []string {
	for i := range models {
		models[i] = strings.TrimPrefix(strings.TrimSpace(models[i]), "models/")
	}
	return dedupeAndSortModelIDs(models)
}
