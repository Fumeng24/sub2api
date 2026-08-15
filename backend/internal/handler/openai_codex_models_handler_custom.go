package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func codexModelsIfNoneMatchCustom(_ *service.Group, _ string) string {
	// Force one complete manifest response so clients replace any cached catalog
	// that still contains Luna. The response can still carry a fresh ETag.
	return ""
}

func (h *OpenAIGatewayHandler) writeConfiguredCodexModelsManifestCustom(c *gin.Context, group *service.Group, body []byte) bool {
	if group == nil || !group.CustomModelsListEnabled() {
		return false
	}
	rewritten, err := restrictCodexModelsManifest(body, group.ModelsListConfig.Models)
	if err != nil {
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Codex models manifest is unavailable")
		return true
	}
	c.Data(http.StatusOK, "application/json", rewritten)
	return true
}

// stripLunaFromCodexModelsManifest prevents Codex clients from discovering a
// model that this site deliberately rejects. This also covers groups without a
// custom models list when the upstream manifest still advertises Luna.
func stripLunaFromCodexModelsManifest(body []byte) ([]byte, error) {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope == nil {
		return nil, errors.New("invalid manifest object")
	}
	rawModels, ok := envelope["models"]
	if !ok {
		return nil, errors.New("missing models array")
	}
	var models []json.RawMessage
	if err := json.Unmarshal(rawModels, &models); err != nil {
		return nil, err
	}

	filtered := make([]json.RawMessage, 0, len(models))
	for _, rawModel := range models {
		var model struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(rawModel, &model); err == nil && isOpenAILunaModel(model.Slug) {
			continue
		}
		filtered = append(filtered, rawModel)
	}
	rawFiltered, err := json.Marshal(filtered)
	if err != nil {
		return nil, err
	}
	envelope["models"] = rawFiltered
	return json.Marshal(envelope)
}

// restrictCodexModelsManifest preserves upstream metadata for configured
// models and synthesizes minimal entries for models absent from this account.
func restrictCodexModelsManifest(body []byte, configuredModels []string) ([]byte, error) {
	models := uniqueCodexManifestModelIDs(configuredModels)
	if len(models) == 0 {
		return body, nil
	}

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, err
	}
	if envelope == nil {
		return nil, errors.New("invalid manifest object")
	}

	rawModels, ok := envelope["models"]
	if !ok {
		return nil, errors.New("missing models array")
	}
	var upstreamModels []json.RawMessage
	if err := json.Unmarshal(rawModels, &upstreamModels); err != nil {
		return nil, err
	}

	upstreamBySlug := make(map[string]json.RawMessage, len(upstreamModels))
	for _, rawModel := range upstreamModels {
		var model struct {
			Slug string `json:"slug"`
		}
		if err := json.Unmarshal(rawModel, &model); err != nil {
			continue
		}
		if slug := strings.TrimSpace(model.Slug); slug != "" {
			upstreamBySlug[slug] = rawModel
		}
	}

	selected := make([]json.RawMessage, 0, len(models))
	for _, modelID := range models {
		if rawModel, ok := upstreamBySlug[modelID]; ok {
			selected = append(selected, rawModel)
			continue
		}
		rawModel, err := json.Marshal(struct {
			Slug string `json:"slug"`
		}{Slug: modelID})
		if err != nil {
			return nil, err
		}
		selected = append(selected, rawModel)
	}

	rawSelected, err := json.Marshal(selected)
	if err != nil {
		return nil, err
	}
	envelope["models"] = rawSelected
	return json.Marshal(envelope)
}

func uniqueCodexManifestModelIDs(models []string) []string {
	seen := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" || isOpenAILunaModel(model) {
			continue
		}
		if _, ok := seen[model]; ok {
			continue
		}
		seen[model] = struct{}{}
		result = append(result, model)
	}
	return result
}
