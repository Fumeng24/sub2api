package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSiteDefaultLoginAgreementDocuments(t *testing.T) {
	require.Equal(t, "2026-06-23", defaultLoginAgreementDate)

	docs := defaultLoginAgreementDocuments()
	require.Len(t, docs, 4)
	expectedHeadings := map[string]string{
		"terms":                  "# 服务条款",
		"usage-policy":           "# 使用政策",
		"supported-regions":      "# 支持的国家与地区",
		"service-specific-terms": "# 服务特定条款",
	}
	for _, doc := range docs {
		require.NotEmpty(t, strings.TrimSpace(doc.ContentMD), doc.ID)
		require.Contains(t, doc.ContentMD, expectedHeadings[doc.ID], doc.ID)
	}
}

func TestNormalizeLoginAgreementDocumentsUsesSiteDefaultsForKnownEmptyDocuments(t *testing.T) {
	docs := normalizeLoginAgreementDocuments([]LoginAgreementDocument{
		{ID: "terms", Title: "服务条款"},
		{ID: "supported-regions", Title: "支持的国家和地区"},
	})

	require.Len(t, docs, 2)
	require.Contains(t, docs[0].ContentMD, "# 服务条款")
	require.Contains(t, docs[1].ContentMD, "# 支持的国家与地区")
}

func TestNormalizeLoginAgreementDocumentsPreservesCustomContent(t *testing.T) {
	docs := normalizeLoginAgreementDocuments([]LoginAgreementDocument{
		{ID: "terms", Title: "服务条款", ContentMD: "custom terms"},
	})

	require.Len(t, docs, 1)
	require.Equal(t, "custom terms", docs[0].ContentMD)
}
