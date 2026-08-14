package service

import "strings"

var sensitiveCredentialAliasesCustom = func() map[string]struct{} {
	keys := []string{
		"access_token", "refresh_token", "id_token", "authorization",
		"api_key", "x-api-key", "session_key", "session_token",
		"cookie", "set-cookie", "upstream_sub2api_password",
		"client_secret", "password", "passwd", "passphrase",
		"aws_secret_access_key", "aws_session_token",
		"service_account_json", "service_account", "private_key",
	}
	aliases := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		aliases[normalizeSensitiveCredentialKeyCustom(key)] = struct{}{}
	}
	return aliases
}()

func isSensitiveCredentialKeyCustom(key string) bool {
	_, ok := sensitiveCredentialAliasesCustom[normalizeSensitiveCredentialKeyCustom(key)]
	return ok
}

func normalizeSensitiveCredentialKeyCustom(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	return strings.NewReplacer("_", "", "-", "", ".", "", " ", "").Replace(key)
}

func mergePreservingSensitiveCredsCustom(existing, incoming map[string]any) (map[string]any, bool) {
	out := make(map[string]any, len(incoming)+len(SensitiveCredentialKeys))
	incomingSensitiveKeys := make(map[string]struct{}, len(incoming))
	for key, value := range incoming {
		out[key] = value
		if IsSensitiveCredentialKey(key) {
			incomingSensitiveKeys[normalizeSensitiveCredentialKeyCustom(key)] = struct{}{}
		}
	}
	for key, value := range existing {
		if !IsSensitiveCredentialKey(key) {
			continue
		}
		if _, supplied := incomingSensitiveKeys[normalizeSensitiveCredentialKeyCustom(key)]; supplied {
			continue
		}
		out[key] = value
	}
	return out, true
}
