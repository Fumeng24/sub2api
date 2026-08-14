import type { Plugin } from 'vite'

const WEG00_HEAD = `
    <meta name="description" content="Wegoo AI 提供 GPT/Codex、Claude、Gemini 与 AI 生图服务，模型目录、访问凭证、用量记录、余额和服务状态集中呈现。" />
    <meta name="keywords" content="Wegoo AI,AI API,OpenAI API 兼容,ChatGPT API,Claude API,Gemini API,Codex API,GPT-5 API,AI API Key,Claude API Key,Gemini API Key,图像生成 API,AI 生图 API,Cursor API,Cline API,Codex TUI,Claude Code,用量记录,服务状态" />
    <meta name="robots" content="index,follow" />
    <meta name="GPTBot" content="index,follow" />
    <meta name="ClaudeBot" content="index,follow" />
    <meta name="PerplexityBot" content="index,follow" />
    <meta name="theme-color" content="#f5f5f7" media="(prefers-color-scheme: light)" />
    <meta name="theme-color" content="#050505" media="(prefers-color-scheme: dark)" />
    <link rel="canonical" href="https://ai.wegoo.site/home" />
    <link rel="alternate" type="text/plain" href="https://ai.wegoo.site/llms.txt" title="Wegoo AI llms.txt" />
    <link rel="alternate" type="application/rss+xml" href="https://ai.wegoo.site/feed.xml" title="Wegoo AI updates" />
    <meta property="og:site_name" content="Wegoo AI" />
    <meta property="og:title" content="AI 模型服务 - Wegoo AI" />
    <meta property="og:description" content="支持 GPT/Codex、Claude、Gemini 与 AI 生图。模型目录、访问凭证、用量记录和服务状态集中呈现。" />
    <meta property="og:type" content="website" />
    <meta property="og:url" content="https://ai.wegoo.site/home" />
    <meta property="og:image" content="https://ai.wegoo.site/logo.png" />
    <meta name="twitter:card" content="summary" />
    <meta name="twitter:title" content="AI 模型服务 - Wegoo AI" />
    <meta name="twitter:description" content="支持 GPT/Codex、Claude、Gemini 与 AI 生图。模型目录、访问凭证、用量记录和服务状态集中呈现。" />
    <meta name="twitter:image" content="https://ai.wegoo.site/logo.png" />
    <link rel="stylesheet" href="/wegoo-bootstrap.css" />
    <script type="application/ld+json">
      {
        "@context": "https://schema.org",
        "@graph": [
          { "@type": "Organization", "@id": "https://ai.wegoo.site/#organization", "name": "Wegoo AI", "url": "https://ai.wegoo.site/", "logo": "https://ai.wegoo.site/logo.png" },
          { "@type": "WebSite", "@id": "https://ai.wegoo.site/#website", "name": "Wegoo AI", "url": "https://ai.wegoo.site/", "publisher": { "@id": "https://ai.wegoo.site/#organization" }, "inLanguage": "zh-CN" },
          {
            "@type": "WebApplication",
            "@id": "https://ai.wegoo.site/#app",
            "name": "Wegoo AI",
            "applicationCategory": "DeveloperApplication",
            "operatingSystem": "Web",
            "url": "https://ai.wegoo.site/home",
            "description": "AI 模型服务，支持 GPT/Codex、Claude、Gemini 与 AI 生图。模型目录、访问凭证、用量记录和服务状态集中呈现。",
            "offers": { "@type": "Offer", "price": "0", "priceCurrency": "USD" },
            "featureList": [
              "一个 API Key 接入 GPT/Codex、Claude、Gemini 与图像模型",
              "OpenAI SDK 与 Anthropic SDK 兼容入口",
              "模型价格、公开分组和服务状态来自后端事实源",
              "用量日志展示 token、缓存、延迟和扣费元数据",
              "充值、卡密、人工收款、退款、赠送和邀请返佣按业务来源分类"
            ]
          },
          {
            "@type": "FAQPage",
            "@id": "https://ai.wegoo.site/home#faq",
            "mainEntity": [
              { "@type": "Question", "name": "Wegoo AI 是什么？", "acceptedAnswer": { "@type": "Answer", "text": "Wegoo AI 是面向开发者的 AI Gateway，用一个 API Key 管理 GPT/Codex、Claude、Gemini 和图像模型的调用、用量、余额和服务状态。" } },
              { "@type": "Question", "name": "模型价格和状态以哪里为准？", "acceptedAnswer": { "@type": "Answer", "text": "公开模型页、公开状态页和登录后控制台读取后端事实源。公开访客只能看到标准公开分组，登录用户可能看到额外授权分组、专属分组或折扣。" } },
              { "@type": "Question", "name": "哪些余额可以开票？", "acceptedAnswer": { "@type": "Answer", "text": "用户正常充值、卡密充值和管理员确认的人工收款可按净充值口径开票；赠送补偿、邀请返佣和未实际付款来源不可开票，退款会抵扣对应净充值。" } }
            ]
          }
        ]
      }
    </script>`

const WEG00_APP = `<div id="app">
      <main class="boot-shell">
        <section class="boot-panel">
          <div class="boot-spinner"></div>
          <h1>Wegoo AI 正在加载</h1>
          <noscript><p>当前浏览器未执行 JavaScript，请开启后刷新页面。</p></noscript>
        </section>
        <section hidden aria-hidden="true">
          <h1>Wegoo AI - AI Gateway for Developers</h1>
          <p>Wegoo AI 提供 GPT/Codex、Claude、Gemini 与 AI 生图服务。开发者可以用一个 API Key 管理模型调用、服务档位、用量记录、余额、发票、工单和服务状态。</p>
          <h2>核心能力</h2>
          <ul>
            <li>OpenAI SDK 和 Anthropic SDK 兼容入口。</li>
            <li>公开模型价格页读取后端事实源，不在前端硬编码价格。</li>
            <li>公开服务状态页展示用户可见模型族健康、延迟和近期状态。</li>
            <li>用量日志记录模型、token、缓存、延迟和费用元数据，帮助用户理解扣费和错误。</li>
            <li>充值、卡密、人工收款、退款、赠送补偿和邀请返佣按业务来源分类。</li>
          </ul>
          <h2>公开入口</h2>
          <ul>
            <li><a href="/pricing">模型与价格</a></li><li><a href="/docs">接入文档</a></li>
            <li><a href="/status">服务状态</a></li><li><a href="/enterprise">企业服务、人工收款与发票</a></li>
            <li><a href="/key-usage">API Key 用量查询</a></li>
          </ul>
          <h2>公开事实源</h2>
          <ul>
            <li><a href="/api/v1/public/model-pricing">公开模型价格 API</a></li>
            <li><a href="/api/v1/public/channel-monitors">公开服务状态 API</a></li>
            <li><a href="/llms.txt">llms.txt</a></li>
          </ul>
        </section>
      </main>
    </div>`

export function transformWegooIndexHtml(html: string): string {
  return html
    .replace(/<title>[^<]*<\/title>/, `${WEG00_HEAD}
    <title>AI 模型服务 - GPT Claude Gemini Codex API - Wegoo AI</title>`)
    .replace('<div id="app"></div>', WEG00_APP)
    .replace('/src/main.ts', '/src/custom/main.ts')
}
export function wegooIndexHtml(): Plugin {
  return {
    name: 'wegoo-index-html',
    transformIndexHtml: {
      order: 'pre',
      handler(html) {
        return transformWegooIndexHtml(html)
      }
    }
  }
}
