package service

const NotificationEmailEventGroupRateChangeNotice = "group.rate_change_notice"

func init() {
	const insertBefore = NotificationEmailEventCyberPolicyNotice
	inserted := false
	for i, event := range notificationEmailEventOrder {
		if event == insertBefore {
			notificationEmailEventOrder = append(notificationEmailEventOrder, "")
			copy(notificationEmailEventOrder[i+1:], notificationEmailEventOrder[i:])
			notificationEmailEventOrder[i] = NotificationEmailEventGroupRateChangeNotice
			inserted = true
			break
		}
	}
	if !inserted {
		notificationEmailEventOrder = append(notificationEmailEventOrder, NotificationEmailEventGroupRateChangeNotice)
	}

	notificationEmailEventDefinitions[NotificationEmailEventGroupRateChangeNotice] = NotificationEmailEventInfo{
		Event:       NotificationEmailEventGroupRateChangeNotice,
		Label:       "Group rate change notice",
		Description: "Optional advance notice sent to recent users before a group rate multiplier changes.",
		Category:    "billing",
		Optional:    true,
		Placeholders: append(append([]string{}, notificationEmailCommonPlaceholders...),
			"group_name", "old_rate_multiplier", "new_rate_multiplier", "effective_at", "window_minutes", "request_count", "actual_cost", "last_used_at", "admin_message", "unsubscribe_url"),
	}
	notificationEmailOfficialTemplates[NotificationEmailEventGroupRateChangeNotice] = map[string]notificationEmailOfficialTemplate{
		notificationEmailDefaultLocale: {
			Subject: "[{{site_name}}] {{group_name}} rate change notice",
			HTML: notificationEmailCard("#0f766e", "Group rate change notice", `
<p>Hello {{recipient_name}},</p>
<p>The rate multiplier for <strong>{{group_name}}</strong> is scheduled to change from <strong>{{old_rate_multiplier}}x</strong> to <strong>{{new_rate_multiplier}}x</strong>.</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>Effective at</td><td>{{effective_at}}</td></tr>
  <tr><td>Recent usage window</td><td>{{window_minutes}} minutes</td></tr>
  <tr><td>Your recent requests</td><td>{{request_count}}</td></tr>
  <tr><td>Your recent billed cost</td><td>${{actual_cost}}</td></tr>
  <tr><td>Last used at</td><td>{{last_used_at}}</td></tr>
</table>
<p>Administrator note: {{admin_message}}</p>
<p class="muted"><a href="{{unsubscribe_url}}">Unsubscribe from optional billing notices</a></p>`),
		},
		notificationEmailLocaleChinese: {
			Subject: "[{{site_name}}] {{group_name}} 费率调整通知",
			HTML: notificationEmailCard("#0f766e", "分组费率调整通知", `
<p>{{recipient_name}}，您好：</p>
<p><strong>{{group_name}}</strong> 分组费率倍数计划从 <strong>{{old_rate_multiplier}}x</strong> 调整为 <strong>{{new_rate_multiplier}}x</strong>。</p>
<table style="width:100%;border-collapse:collapse;">
  <tr><td>生效时间</td><td>{{effective_at}}</td></tr>
  <tr><td>统计窗口</td><td>最近 {{window_minutes}} 分钟</td></tr>
  <tr><td>您的请求数</td><td>{{request_count}}</td></tr>
  <tr><td>您的计费金额</td><td>${{actual_cost}}</td></tr>
  <tr><td>最近使用时间</td><td>{{last_used_at}}</td></tr>
</table>
<p>管理员备注：{{admin_message}}</p>
<p class="muted"><a href="{{unsubscribe_url}}">退订此类计费通知</a></p>`),
		},
	}
}

func notificationEmailSampleVariablesCustom(locale string, variables map[string]string) map[string]string {
	if normalizeNotificationLocale(locale) == notificationEmailLocaleChinese {
		variables["admin_message"] = "本次调整将用于覆盖上游成本变化。"
	} else {
		variables["admin_message"] = "This adjustment reflects upstream cost changes."
	}
	variables["old_rate_multiplier"] = "1"
	variables["new_rate_multiplier"] = "1.25"
	variables["effective_at"] = "2026-05-20 12:30 UTC"
	variables["window_minutes"] = "30"
	variables["request_count"] = "18"
	variables["actual_cost"] = "3.42"
	variables["last_used_at"] = "2026-05-20 12:10 UTC"
	return variables
}

func notificationEmailCardCustom(accent, title, content string) string {
	return `<!doctype html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <style>
    body {
      margin: 0;
      padding: 0;
      background: #eef2f7;
      color: #111827;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Arial, "PingFang SC", "Microsoft YaHei", sans-serif;
    }
    .preheader {
      display: none;
      max-height: 0;
      overflow: hidden;
      opacity: 0;
      color: transparent;
    }
    .page {
      width: 100%;
      background:
        radial-gradient(circle at 12% 0%, rgba(255,255,255,0.95) 0, rgba(255,255,255,0) 30%),
        linear-gradient(145deg, #eef2f7 0%, #f8fafc 52%, #e5e7eb 100%);
      padding: 42px 16px;
    }
    .shell {
      max-width: 680px;
      margin: 0 auto;
    }
    .brand {
      margin: 0 0 14px;
      color: #64748b;
      font-size: 12px;
      font-weight: 700;
      letter-spacing: 0.16em;
      text-transform: uppercase;
    }
    .brand-dot {
      display: inline-block;
      width: 10px;
      height: 10px;
      margin-right: 8px;
      border-radius: 999px;
      background: ` + accent + `;
      box-shadow: 0 0 0 6px rgba(15, 23, 42, 0.06);
      vertical-align: -1px;
    }
    .container {
      overflow: hidden;
      background: #ffffff;
      border: 1px solid rgba(148, 163, 184, 0.28);
      border-radius: 24px;
      box-shadow: 0 24px 70px rgba(15, 23, 42, 0.16);
    }
    .hero {
      position: relative;
      background: ` + accent + `;
      background: linear-gradient(135deg, ` + accent + ` 0%, #111827 100%);
      color: #ffffff;
      padding: 34px 40px 36px;
    }
    .eyebrow {
      display: inline-block;
      margin-bottom: 18px;
      padding: 7px 12px;
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.16);
      color: rgba(255, 255, 255, 0.92);
      font-size: 11px;
      font-weight: 800;
      letter-spacing: 0.14em;
      text-transform: uppercase;
    }
    .hero h1 {
      margin: 0;
      font-size: 30px;
      line-height: 1.18;
      font-weight: 800;
      letter-spacing: -0.03em;
    }
    .content {
      padding: 38px 40px 36px;
      font-size: 15px;
      line-height: 1.75;
    }
    .content p {
      margin: 0 0 16px;
    }
    .content strong {
      color: #0f172a;
      font-weight: 750;
    }
    .content table {
      width: 100%;
      margin: 22px 0;
      border-collapse: separate !important;
      border-spacing: 0;
      overflow: hidden;
      border: 1px solid #e5e7eb;
      border-radius: 16px;
      background: #ffffff;
    }
    .content td {
      padding: 13px 16px;
      border-bottom: 1px solid #eef2f7;
      color: #334155;
      vertical-align: top;
    }
    .content tr:last-child td {
      border-bottom: 0;
    }
    .content td:first-child {
      width: 42%;
      background: #f8fafc;
      color: #64748b;
      font-weight: 700;
    }
    .button {
      display: inline-block;
      margin-top: 10px;
      padding: 13px 20px;
      border-radius: 12px;
      background: ` + accent + `;
      color: #ffffff !important;
      text-decoration: none;
      font-weight: 800;
      box-shadow: 0 12px 24px rgba(15, 23, 42, 0.18);
    }
    .muted {
      color: #64748b;
      font-size: 13px;
      line-height: 1.65;
    }
    .muted a {
      color: ` + accent + `;
      font-weight: 700;
    }
    .footer {
      padding: 22px 40px 28px;
      background: #f8fafc;
      border-top: 1px solid #eef2f7;
      color: #94a3b8;
      font-size: 12px;
      line-height: 1.6;
    }
    @media only screen and (max-width: 640px) {
      .page { padding: 0; }
      .shell { max-width: none; }
      .brand { display: none; }
      .container { border-radius: 0; border-left: 0; border-right: 0; }
      .hero { padding: 28px 24px 30px; }
      .hero h1 { font-size: 25px; }
      .content { padding: 30px 24px; }
      .footer { padding: 20px 24px 24px; }
      .content td { display: block; width: auto !important; }
      .content td:first-child { border-bottom: 0; }
    }
  </style>
</head>
<body>
  <div class="preheader">` + title + ` · {{site_name}}</div>
  <div class="page">
    <div class="shell">
      <div class="brand"><span class="brand-dot"></span>{{site_name}}</div>
      <div class="container">
        <div class="hero">
          <div class="eyebrow">{{site_name}} Notice</div>
          <h1>` + title + `</h1>
        </div>
        <div class="content">` + content + `</div>
        <div class="footer">This message was sent automatically by {{site_name}}. Please do not reply directly.</div>
      </div>
    </div>
  </div>
</body>
</html>`
}
