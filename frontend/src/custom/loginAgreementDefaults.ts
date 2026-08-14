import type { LoginAgreementDocument } from "@/types";
import serviceSpecificTermsContent from "./loginAgreement/service-specific-terms.zh.md?raw";
import supportedRegionsContent from "./loginAgreement/supported-regions.zh.md?raw";
import termsContent from "./loginAgreement/terms.zh.md?raw";
import usagePolicyContent from "./loginAgreement/usage-policy.zh.md?raw";

export const siteDefaultLoginAgreementUpdatedAt = "2026-06-23";

const siteDefaultsByID: Readonly<
  Record<string, Pick<LoginAgreementDocument, "title" | "content_md">>
> = {
  terms: {
    title: "服务条款",
    content_md: termsContent.trim(),
  },
  "usage-policy": {
    title: "使用政策",
    content_md: usagePolicyContent.trim(),
  },
  "supported-regions": {
    title: "支持的国家与地区",
    content_md: supportedRegionsContent.trim(),
  },
  "service-specific-terms": {
    title: "服务特定条款",
    content_md: serviceSpecificTermsContent.trim(),
  },
};

export function applySiteLoginAgreementDefaults(
  documents: LoginAgreementDocument[],
): LoginAgreementDocument[] {
  return documents.map((document) => {
    const siteDefault = siteDefaultsByID[document.id];
    return siteDefault ? { ...document, ...siteDefault } : { ...document };
  });
}
