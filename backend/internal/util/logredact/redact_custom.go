package logredact

import "regexp"

var customSensitiveKeyList = []string{
	"authorization",
	"proxy-authorization",
	"api_key",
	"api-key",
	"apikey",
	"x-api-key",
	"accesstoken",
	"refreshtoken",
	"idtoken",
	"session_token",
	"sessiontoken",
	"clientsecret",
	"credential",
	"credentials",
	"cookie",
	"set-cookie",
}

var (
	reOpenAICustom = regexp.MustCompile(`\bsk-[A-Za-z0-9][A-Za-z0-9_-]{12,}`)
	reBearerCustom = regexp.MustCompile(`(?i)\b(Bearer)\s+[A-Za-z0-9._~+/\-=]{12,}`)
	reHeaderCustom = regexp.MustCompile(`(?i)\b(authorization|proxy-authorization|cookie|set-cookie|x-api-key)\s*:\s*[^,\r\n]+`)
)

func appendLogRedactCustomKeys(extraKeys []string) []string {
	keys := make([]string, 0, len(customSensitiveKeyList)+len(extraKeys))
	keys = append(keys, customSensitiveKeyList...)
	keys = append(keys, extraKeys...)
	return keys
}

func redactTextTokensCustom(input string) string {
	input = reOpenAICustom.ReplaceAllString(input, "sk-***")
	return reBearerCustom.ReplaceAllString(input, `$1 ***`)
}

func redactTextHeadersCustom(input string) string {
	return reHeaderCustom.ReplaceAllString(input, `$1: ***`)
}
