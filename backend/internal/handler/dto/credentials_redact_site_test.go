package dto

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRedactCredentials_StripsCaseVariantsAndSiteSecrets(t *testing.T) {
	in := map[string]any{
		"apiKey":                    "sk-camel-secret",
		"Authorization":             "Bearer secret",
		"upstream_sub2api_password": "sub2api-secret",
		"clientSecret":              "client-secret",
		"privateKey":                "camel-key",
		"base_url":                  "https://api.example.com",
	}

	out, status := RedactCredentials(in)

	for _, key := range []string{"apiKey", "Authorization", "upstream_sub2api_password", "clientSecret", "privateKey"} {
		require.NotContains(t, out, key)
		require.True(t, status["has_"+key])
	}
	require.Equal(t, "https://api.example.com", out["base_url"])
	require.Equal(t, "sub2api-secret", in["upstream_sub2api_password"])
}

func TestRedactSensitiveMap_StripsNestedSensitiveKeys(t *testing.T) {
	in := map[string]any{
		"privacy_mode": "training_off",
		"access_token": "at-secret",
		"nested": map[string]any{
			"apiKey":       "sk-secret",
			"clientSecret": "client-secret",
			"base_url":     "https://api.example.com",
		},
		"items": []any{
			map[string]any{
				"refreshToken": "rt-secret",
				"safe":         "ok",
			},
		},
	}

	out := RedactSensitiveMap(in)

	require.NotContains(t, out, "access_token")
	require.Equal(t, "training_off", out["privacy_mode"])
	require.Equal(t, "at-secret", in["access_token"], "原始 map 不应被修改")

	nested, ok := out["nested"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, nested, "apiKey")
	require.NotContains(t, nested, "clientSecret")
	require.Equal(t, "https://api.example.com", nested["base_url"])

	items, ok := out["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, item, "refreshToken")
	require.Equal(t, "ok", item["safe"])
}
