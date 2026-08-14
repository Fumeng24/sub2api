package service

import (
	_ "embed"
	"strings"
)

func withSiteDefaultLoginAgreementDocumentContent(id, title, content string) string {
	if content != "" {
		return content
	}
	return siteDefaultLoginAgreementDocumentContent(id, title)
}

const siteDefaultLoginAgreementDate = "2026-06-23"

//go:embed legal/terms.zh.md
var siteDefaultTermsContentMD string

//go:embed legal/usage-policy.zh.md
var siteDefaultUsagePolicyContentMD string

//go:embed legal/supported-regions.zh.md
var siteDefaultSupportedRegionsContentMD string

//go:embed legal/service-specific-terms.zh.md
var siteDefaultServiceSpecificTermsContentMD string

func withSiteDefaultLoginAgreementDocuments(docs []LoginAgreementDocument) []LoginAgreementDocument {
	for i := range docs {
		docs[i].ContentMD = siteDefaultLoginAgreementDocumentContent(docs[i].ID, docs[i].Title)
	}
	return docs
}

func siteDefaultLoginAgreementDocumentContent(id, title string) string {
	id = normalizeLoginAgreementDocumentID(id)
	title = strings.TrimSpace(title)
	switch {
	case id == "terms" || title == "服务条款":
		return siteDefaultTermsContentMD
	case id == "usage-policy" || title == "使用政策":
		return siteDefaultUsagePolicyContentMD
	case id == "supported-regions" || title == "支持的国家与地区" || title == "支持的国家和地区":
		return siteDefaultSupportedRegionsContentMD
	case id == "service-specific-terms" || title == "服务特定条款":
		return siteDefaultServiceSpecificTermsContentMD
	default:
		return ""
	}
}
