import { describe, expect, it } from "vitest";
import type { LoginAgreementDocument } from "@/types";
import {
  applySiteLoginAgreementDefaults,
  siteDefaultLoginAgreementUpdatedAt,
} from "../loginAgreementDefaults";

const officialDefaults = (): LoginAgreementDocument[] => [
  { id: "terms", title: "Terms of Service", content_md: "" },
  { id: "usage-policy", title: "Usage Policy", content_md: "" },
  {
    id: "supported-regions",
    title: "Supported Countries and Regions",
    content_md: "",
  },
  {
    id: "service-specific-terms",
    title: "Service-Specific Terms",
    content_md: "",
  },
];

describe("site login agreement defaults", () => {
  it("preserves the site agreement date and document contents", () => {
    const documents = applySiteLoginAgreementDefaults(officialDefaults());

    expect(siteDefaultLoginAgreementUpdatedAt).toBe("2026-06-23");
    expect(documents.map((document) => document.title)).toEqual([
      "服务条款",
      "使用政策",
      "支持的国家与地区",
      "服务特定条款",
    ]);
    expect(documents.map((document) => document.content_md.split("\n", 1)[0])).toEqual([
      "# 服务条款",
      "# 使用政策",
      "# 支持的国家与地区",
      "# 服务特定条款",
    ]);
  });

  it("keeps future official documents instead of hiding them", () => {
    const futureDocument: LoginAgreementDocument = {
      id: "future-policy",
      title: "Future policy",
      content_md: "official content",
    };

    const documents = applySiteLoginAgreementDefaults([
      ...officialDefaults(),
      futureDocument,
    ]);

    expect(documents.at(-1)).toEqual(futureDocument);
    expect(documents.at(-1)).not.toBe(futureDocument);
  });
});
