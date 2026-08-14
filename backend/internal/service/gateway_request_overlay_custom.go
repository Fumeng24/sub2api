package service

import "strings"

func cloneGatewayRequestScalarStringsCustom(parsed *ParsedRequest) {
	if parsed == nil {
		return
	}
	parsed.Model = strings.Clone(parsed.Model)
	parsed.MetadataUserID = strings.Clone(parsed.MetadataUserID)
	parsed.OutputEffort = strings.Clone(parsed.OutputEffort)
}
