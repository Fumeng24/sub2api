// Package dto provides data transfer objects for HTTP handlers.
package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

// RedactCredentials 复制一份 in，剥离 service.SensitiveCredentialKeys 列出的所有敏感子键，
// 并产出一个 has_<key> 状态 map 表示哪些敏感键存在且非零值。
//
// 输入 nil 时返回 nil, nil（避免响应里出现空对象）。
// 不修改入参；调用方拿到的 out 可安全序列化进 JSON 返回前端。
func RedactCredentials(in map[string]any) (out map[string]any, status map[string]bool) {
	if in == nil {
		return nil, nil
	}
	out = make(map[string]any, len(in))
	for k, v := range in {
		if service.IsSensitiveCredentialKey(k) {
			if isCredentialValuePresent(v) {
				if status == nil {
					status = make(map[string]bool, 4)
				}
				status["has_"+k] = true
			}
			continue
		}
		out[k] = v
	}
	return out, status
}

// RedactSensitiveMap 复制一份任意响应 map，并递归剥离常见敏感凭据键。
//
// Account.Extra 理论上只保存展示/调度配置，但管理员接口允许传入任意 JSON；
// 这里作为响应层兜底，避免误把 token、cookie、私钥等放进 Extra 后被列表接口带回前端。
func RedactSensitiveMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if service.IsSensitiveCredentialKey(k) {
			continue
		}
		out[k] = redactSensitiveValue(v)
	}
	return out
}

func redactSensitiveValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return RedactSensitiveMap(x)
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = redactSensitiveValue(x[i])
		}
		return out
	default:
		return v
	}
}

// isCredentialValuePresent 判断值是否"存在且非零"。空字符串、nil、false 均视为未配置；
// 其余非零类型（数字、对象、字符串等）视为已配置。
func isCredentialValuePresent(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return x != ""
	case bool:
		return x
	default:
		return true
	}
}
