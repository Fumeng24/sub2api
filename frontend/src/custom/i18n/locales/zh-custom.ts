// Site-specific translations layered over the official locale baseline.
const custom = {
  "home": {
    "viewDocs": "查看接入教程",
    "docs": "接入教程",
    "heroSubtitle": "主流模型统一接入",
    "heroDescription": "GPT、Claude、Gemini 与图像模型统一接入，模型状态、价格、余额和用量明细清晰呈现。",
    "tags": {
      "subscriptionToApi": "模型服务",
      "stickySession": "访问记录",
      "realtimeBilling": "余额可查"
    },
    "painPoints": {
      "title": "平台概览",
      "items": {
        "expensive": {
          "title": "用量费用",
          "desc": "模型价格、服务倍率、Token 和扣费明细清晰呈现"
        },
        "complex": {
          "title": "模型目录",
          "desc": "GPT、Claude、Gemini 和图像模型一并列出"
        },
        "unstable": {
          "title": "服务状态",
          "desc": "可用性、延迟、近期状态和模型列表持续可见"
        },
        "noControl": {
          "title": "访问与记录",
          "desc": "账户状态和访问状态清晰可见"
        }
      }
    },
    "solutions": {
      "title": "平台概览",
      "subtitle": "模型、账户、服务状态和用量明细共用一个入口"
    },
    "features": {
      "unifiedGateway": "模型服务",
      "unifiedGatewayDesc": "GPT、Claude、Gemini 和图像模型统一接入，模型状态和服务档位一并呈现。",
      "multiAccount": "服务状态",
      "multiAccountDesc": "按模型、服务档位和实时状态展示可用路径。",
      "balanceQuota": "用量记录",
      "balanceQuotaDesc": "按实际用量扣费，余额、请求和 Token 消耗在仪表盘呈现。"
    },
    "comparison": {
      "title": "产品口径",
      "headers": {
        "us": "WegooAI"
      },
      "items": {
        "models": {
          "us": "支持多模型选择"
        },
        "management": {
          "us": "统一访问凭证"
        },
        "stability": {
          "official": "服务商各自展示状态",
          "us": "服务状态页面可见"
        },
        "control": {
          "official": "按各服务规则设置",
          "us": "支持配额与明细"
        }
      }
    },
    "providers": {
      "description": "覆盖常用模型服务和图像能力"
    },
    "cta": {
      "title": "模型服务",
      "description": "模型、价格、状态和用量明细清晰呈现。",
      "button": "注册账户"
    },
    "docsBadge": "教程",
    "landing": {
      "nav": {
        "home": "首页",
        "primary": "首页导航",
        "docs": "接入教程",
        "continue": "继续使用",
        "signIn": "登录",
        "noDilution": "模型",
        "billing": "用量",
        "privacy": "账户",
        "stability": "服务"
      },
      "hero": {
        "eyebrow": "为开发者构建的多模型入口",
        "title": "一个账户，连接主流 AI 模型。",
        "lead": "GPT、Claude、Gemini 与图像模型统一接入，账户内呈现模型状态、用量、余额和服务入口。",
        "meta": "模型目录、访问凭证、用量明细和服务状态在同一套账户体系中呈现。",
        "capabilitiesLabel": "平台能力",
        "primaryAction": "开始使用",
        "statusAction": "查看模型状态"
      },
      "workflow": {
        "application": "你的应用",
        "gateway": "Wegoo API 网关",
        "gatewayDetail": "鉴权、路由、计费与状态",
        "models": "主流模型服务"
      },
      "platform": {
        "kicker": "统一工作台",
        "title": "从接入到核对，每一步都清楚。",
        "lead": "模型、访问凭证、用量、余额与服务状态保持在同一个账户上下文里。"
      },
      "integration": {
        "kicker": "接入教程",
        "title": "保留熟悉的 SDK，只替换 Base URL。",
        "lead": "沿用 OpenAI、Anthropic 与常见命令行工具的调用方式，快速接入现有项目。",
        "steps": {
          "key": "创建访问凭证并选择服务档位",
          "endpoint": "将客户端 Base URL 指向 Wegoo API",
          "request": "选择模型并发起第一个请求"
        }
      },
      "cta": {
        "kicker": "准备就绪",
        "title": "把下一次模型调用，交给一个更清晰的入口。",
        "lead": "注册账户后即可创建访问凭证、查看模型与倍率，并按接入教程完成配置。"
      },
      "claims": {
        "official": "多模型统一接入",
        "noDilution": "服务状态",
        "noChats": "用量记录",
        "billingCovered": "账户余额"
      },
      "product": {
        "statusLabel": "平台概览",
        "statusValue": "模型、访问、用量和服务状态一览",
        "statusBadge": "统一工作台",
        "modelIntegrity": "模型目录",
        "modelIntegrityDetail": "GPT、Claude、Gemini 和图像模型按服务档位展示。",
        "privacyBoundary": "访问凭证",
        "privacyBoundaryDetail": "API 访问、分组权限和调用入口在账户内统一管理。",
        "billingProtection": "用量记录",
        "billingProtectionDetail": "请求、Token、倍率和余额变化形成账户记录。",
        "serviceContinuity": "服务状态",
        "serviceContinuityDetail": "可用模型、近期状态和账户记录保持可见。",
        "gpt": "GPT 服务",
        "claude": "Claude 服务",
        "gemini": "Gemini 服务",
        "images": "图像",
        "imagesDetail": "生图能力按模型开放",
        "live": "在线",
        "ready": "可用"
      },
      "metrics": {
        "official": "模型服务",
        "officialValue": "多模型",
        "storedChats": "API 接入",
        "storedChatsValue": "兼容",
        "chargeExceptions": "账户记录",
        "covered": "集中"
      },
      "sections": {
        "models": {
          "kicker": "模型服务",
          "title": "主流模型放在同一个入口。",
          "description": "GPT、Claude、Gemini 和图像模型按服务档位呈现，列出模型名、能力范围和可用状态。"
        },
        "billing": {
          "kicker": "用量记录",
          "title": "余额、调用和 Token 明细一并呈现。",
          "description": "模型、倍率、Token、请求和余额变化按时间呈现。"
        },
        "privacy": {
          "kicker": "账户空间",
          "title": "访问凭证、订单和账户记录各在其位。",
          "description": "账户内保留必要的运行记录，用于扣费核对、服务状态展示和售后处理。"
        },
        "stability": {
          "kicker": "服务状态",
          "title": "服务状态按模型呈现。",
          "description": "模型列表、可用状态和请求状态各自清晰。"
        }
      },
      "privacyVisual": {
        "label": "运行记录"
      },
      "lineup": {
        "gpt": "文本、代码和工具调用场景",
        "claude": "长文、代码和 Agent 任务",
        "gemini": "多模态理解与生成",
        "images": "图片生成能力按模型开放"
      },
      "ledger": {
        "modelSource": "模型目录",
        "official": "GPT / Claude / Gemini / 图像",
        "usageTrail": "用量链路",
        "trace": "Token、倍率、余额一并展示",
        "billingExceptions": "余额变化",
        "covered": "账户记录",
        "serviceState": "服务状态",
        "online": "可用状态清楚展示"
      },
      "commitment": {
        "kicker": "平台承诺",
        "title": "模型、隐私、稳定和扣费。",
        "description": "WegooAI 对模型来源、隐私边界、服务稳定和扣费异常的统一承诺。",
        "items": {
          "models": {
            "title": "官方满血模型",
            "detail": "GPT、Claude、Gemini 和图像模型按官方能力接入，永远不掺水。"
          },
          "privacy": {
            "title": "隐私边界",
            "detail": "平台不保存用户聊天对话，不主动泄漏用户信息。"
          },
          "stability": {
            "title": "稳定性行业内极佳",
            "detail": "服务状态、模型可用性和请求记录保持清楚可见。"
          },
          "billing": {
            "title": "扣费异常处理",
            "detail": "确认异常扣费后，按可追溯记录包赔。"
          }
        }
      }
    },
    "terminal": {
      "scheduling": "模型服务状态同步中..."
    },
    "valueRail": {
      "credit": {
        "value": "1 元 = 1 刀",
        "label": "余额口径清楚",
        "description": "充值、兑换、赠送额度和余额变动按统一口径呈现。"
      },
      "models": {
        "value": "GPT / Claude / Gemini",
        "label": "模型服务",
        "description": "模型、服务档位和可用状态在同一处呈现。"
      },
      "compatible": {
        "value": "OpenAI 兼容",
        "label": "访问详情清楚",
        "description": "常见 API 工具、Codex、Claude Code、Gemini CLI 等访问详情一并呈现。"
      },
      "billing": {
        "value": "明细可查",
        "label": "用量记录",
        "description": "余额、请求、Token、缓存命中和消费明细在仪表盘呈现。"
      }
    },
    "cost": {
      "kicker": "价格先讲清楚",
      "title": "不是月费订阅，用多少扣多少",
      "description": "账户余额按统一口径展示，实际消耗按模型价格与服务档位倍率生成记录。倍率越低，同样余额可覆盖的官方原价额度越多。",
      "facts": {
        "creditValue": "1 元 = 1 刀",
        "creditLabel": "余额展示口径",
        "multiplierValue": "倍率计费",
        "multiplierLabel": "官方价 × 服务倍率",
        "recordsValue": "明细可查",
        "recordsLabel": "用量与扣费明细"
      },
      "formulaTitle": "官方 API 倍率怎么估算？",
      "formula": "官方 API 计费倍率 ≈ 服务倍率 ÷ 当前汇率。比如 0.148x 在 1 USD≈¥7.2 时，约为官方 API 计费的 0.02 倍。",
      "note": "倍率是便于理解的估算，实际扣费以使用记录、模型价格和当前服务倍率为准。"
    },
    "quickStart": {
      "kicker": "清晰账户",
      "title": "模型、账户和用量放在同一处",
      "register": "账户余额",
      "createKey": "服务档位",
      "readGuide": "服务状态",
      "steps": {
        "register": {
          "title": "余额与权益",
          "description": "充值、赠送额度、订阅权益和余额变动按时间列明。"
        },
        "group": {
          "title": "模型服务",
          "description": "可用服务档位、模型列表和启用状态一并呈现。"
        },
        "connect": {
          "title": "访问与账户",
          "description": "访问凭证、分组权限和用量明细在账户内管理。"
        }
      }
    },
    "integration": {
      "kicker": "账户信息",
      "title": "模型、状态、用量和订单一览",
      "description": "WegooAI 将模型服务、服务状态、用量明细、余额变动和订单进度放在固定入口。",
      "cards": {
        "key": {
          "title": "服务接入与访问凭证",
          "description": "访问凭证、服务档位、可用模型和访问详情一并呈现。",
          "action": "访问凭证"
        },
        "status": {
          "title": "服务状态公开展示",
          "description": "不同模型服务的可用性、延迟和近期状态清晰可见。",
          "action": "查看服务状态"
        },
        "models": {
          "title": "可用模型",
          "description": "可用模型、服务档位和价格信息一并呈现。",
          "action": "查看可用模型"
        },
        "billing": {
          "title": "用量记录",
          "description": "请求、Token、缓存命中和扣费明细保留在用量记录中。",
          "action": "查看用量记录"
        }
      }
    },
    "useCases": {
      "title": "使用场景",
      "description": "开发、写作、自动化和图像生成场景共享同一个账户记录。",
      "coding": {
        "title": "代码开发和 Vibe Coding",
        "description": "适合 Codex、Claude Code、Cursor、OpenCode 等工作流，用于项目搭建、Bug 修复、代码解释和 Agent 任务。"
      },
      "writing": {
        "title": "写作、翻译和论文润色",
        "description": "Claude / Gemini 服务适合长文总结、资料整理、翻译润色、报告生成和复杂分析。"
      },
      "automation": {
        "title": "脚本自动化和 API 调用",
        "description": "OpenAI 兼容接口方便接到现有脚本、Bot、内部工具和批量处理任务里。"
      },
      "image": {
        "title": "AI 生图和素材生成",
        "description": "生图服务档位用于图片生成，适合视觉素材、配图、创意草图和批量出图。"
      }
    },
    "trust": {
      "title": "账户信息",
      "description": "模型、价格、状态、用量和账户状态清晰呈现。",
      "points": {
        "modelTruth": "GPT、Claude、Gemini 和图像模型按服务档位展示。",
        "billing": "每次请求的用量和扣费明细按时间呈现。",
        "status": "服务状态、可用性、延迟和模型列表持续更新。",
        "privacy": "账户状态和访问状态清晰可见。"
      }
    },
    "firstRun": {
      "kicker": "账户记录",
      "title": "状态、费用和明细",
      "description": "不同服务档位的价格、速度、状态和可用模型按页面呈现。",
      "tips": {
        "trySmall": {
          "title": "余额记录",
          "description": "账户余额、赠送额度、充值记录和订阅权益一并呈现。"
        },
        "serviceStatus": {
          "title": "服务状态",
          "description": "可用性、延迟、模型列表和服务状态公开展示。"
        },
        "groupChoice": {
          "title": "模型服务",
          "description": "GPT、Claude、Gemini 和图像模型按服务档位展示。"
        },
        "records": {
          "title": "访问与记录",
          "description": "账户状态和访问状态清晰可见；用量、Token、缓存和消费明细按时间保留。"
        }
      }
    }
  },
  "keyUsage": {
    "title": "访问凭证用量记录",
    "subtitle": "余额、额度、Token、模型分布和费用明细一并呈现。",
    "query": "查看记录",
    "querying": "读取中...",
    "privacyNote": "访问凭证仅用于本次浏览器查询，不会保存到页面。",
    "dateRange": "统计范围",
    "apply": "更新范围",
    "quotaMode": "凭证限额模式",
    "enterApiKey": "访问凭证为空",
    "querySuccess": "记录已更新",
    "queryFailed": "记录读取失败",
    "queryFailedRetry": "记录读取失败，稍后会恢复。",
    "heroKicker": "用量记录",
    "queryPanel": {
      "title": "当前凭证记录",
      "description": "余额、限额和扣费明细按当前访问凭证呈现。"
    },
    "trust": {
      "sessionQuery": "本次查询",
      "sessionQueryDesc": "访问凭证仅用于当前浏览器请求，不在页面内持久保存。",
      "auditableRecords": "记录明细",
      "auditableRecordsDesc": "按日期、模型与费用展示扣费明细。",
      "privacyBoundary": "请求信息",
      "privacyBoundaryDesc": "页面展示用量与状态，不展示请求正文。"
    }
  },
  "common": {
    "ready": "就绪",
    "language": "语言",
    "apply": "应用",
    "clear": "清除",
    "creating": "创建中...",
    "login": "登录",
    "required": "必填",
    "sending": "发送中...",
    "tryAgain": "重试",
    "details": "详情",
    "download": "下载",
    "remove": "移除"
  },
  "legal": {
    "notFoundDescription": "当前文档暂不可用或已更新，请返回登录页重新查看。",
    "gateway": {
      "kicker": "Legal Document",
      "officialRecord": "平台正式记录",
      "markdownSource": "Markdown 文档"
    }
  },
  "nav": {
    "apiKeys": "访问凭证",
    "redeem": "卡密兑换",
    "availableChannels": "可用模型",
    "buySubscription": "余额充值",
    "docs": "接入教程",
    "channelStatus": "服务状态",
    "accountOverview": "账户概览",
    "tickets": "我的工单",
    "siteMessages": "站内信",
    "ticketManagement": "工单管理",
    "imageGeneration": "AI 生图",
    "cardCodePurchase": "卡密购买",
    "schedulerManagement": "调度管理",
    "upstreams": "上游管理",
    "userPricing": "折扣管理",
    "docsBadge": "教程",
    "invoices": "开票申请",
    "invoiceManagement": "发票管理",
    "businessSettings": "业务设置"
  },
  "auth": {
    "welcomeBack": "账户登录",
    "signInToAccount": "账户内可查看模型、余额、用量和服务状态。",
    "signUpToStart": "{siteName} 提供模型服务、访问凭证、用量明细和账户余额。",
    "registrationDisabled": "注册暂未开放，已有账户仍可登录。",
    "emailPlaceholder": "邮箱地址",
    "passwordPlaceholder": "密码",
    "createPasswordPlaceholder": "设置账户密码",
    "emailRequired": "邮箱不能为空",
    "invalidEmail": "邮箱地址格式无效",
    "passwordRequired": "密码不能为空",
    "loginFailed": "登录失败，凭据未通过验证。",
    "registrationFailed": "注册未完成。",
    "emailSuffixNotAllowed": "该邮箱域名不在注册范围内。",
    "emailSuffixNotAllowedWithAllowed": "该邮箱域名不在注册范围内。可用域名：{suffixes}",
    "emailSuffixAllowedMore": "另有 {count} 项",
    "loginSuccess": "登录成功。",
    "accountCreatedSuccess": "{siteName} 账户已创建。",
    "reloginRequired": "会话已过期，登录状态需要更新。",
    "turnstileExpired": "验证已过期",
    "turnstileFailed": "验证未通过",
    "completeVerification": "验证尚未完成",
    "verifyYourEmail": "验证邮箱",
    "sessionExpiredDesc": "注册会话已失效。",
    "verificationCodeHint": "邮箱中的 6 位验证码",
    "clickToResend": "重新发送验证码",
    "codeSentSuccess": "验证码已发送。",
    "sendCodeFailed": "验证码发送失败。",
    "verifyFailed": "验证未通过。",
    "codeRequired": "验证码不能为空",
    "invalidCode": "验证码需为 6 位数字",
    "promoCodePlaceholder": "优惠码",
    "promoCodeValid": "优惠码有效，注册后获得 ${amount} 赠送余额",
    "promoCodeValidating": "优惠码正在验证中",
    "promoCodeInvalidCannotRegister": "优惠码无效，注册暂未继续",
    "invitationCodePlaceholder": "邀请码",
    "invitationCodeRequired": "邀请码不能为空",
    "invitationCodeInvalidCannotRegister": "邀请码无效，注册暂未继续",
    "oauthOrContinue": "或使用其他方式继续",
    "linuxdo": {
      "callbackProcessing": "正在验证登录信息...",
      "callbackHint": "页面长时间停留时，登录页可重新发起授权。",
      "callbackMissingToken": "登录信息缺失。",
      "invitationRequired": "该 Linux.do 账号尚未注册，当前注册需要邀请码。",
      "invalidPendingToken": "注册凭证已失效。",
      "completeRegistrationFailed": "注册未完成，邀请码未通过验证。"
    },
    "dingtalk": {
      "callbackProcessing": "正在验证钉钉登录信息...",
      "callbackHint": "页面长时间停留时，登录页可重新发起授权。",
      "callbackMissingToken": "登录信息缺失。",
      "invitationRequired": "该钉钉账号尚未注册，当前注册需要邀请码。",
      "invalidPendingToken": "注册凭证已失效。",
      "completeRegistrationFailed": "注册未完成，邀请码未通过验证。",
      "registrationDisabledRedirectToBind": "当前已禁止注册新账户，已有账户可绑定钉钉登录",
      "error": {
        "csrf": "登录会话已过期，需要重新扫码",
        "corp_rejected": "该钉钉账号不属于本企业，平台支持可协助核验",
        "upstream_error": "钉钉服务暂时不可用",
        "missing_browser_session": "浏览器会话丢失，登录状态需要更新",
        "session_error": "会话创建失败"
      }
    },
    "oidc": {
      "callbackProcessing": "正在验证 {providerName} 登录信息...",
      "callbackHint": "页面长时间停留时，登录页可重新发起授权。",
      "callbackMissingToken": "登录信息缺失。",
      "invitationRequired": "该 {providerName} 账号尚未注册，当前注册需要邀请码。",
      "invalidPendingToken": "注册凭证已失效。",
      "completeRegistrationFailed": "注册未完成，邀请码未通过验证。"
    },
    "oauthFlow": {
      "reviewProfileBeforeContinue": "{providerName} 资料确认后继续。",
      "chooseAccountActionHint": "可绑定已有账户，也可创建新账户。",
      "suggestedEmail": "邮箱：{email}",
      "createAccountHint": "邮箱地址用于创建账户并继续。",
      "bindLoginHint": "已有账户登录后会绑定此次 {providerName} 登录。",
      "signInThenBindDescription": "已有账户登录后，此次 {providerName} 登录会绑定到该账户。",
      "totpHint": "{account} 的 6 位验证码用于完成此次 {providerName} 登录绑定。",
      "wechatAvailabilityUnknown": "微信登录可用性暂时无法确认。",
      "wechatSystemBrowserOnly": "当前微信登录流程仅在系统浏览器中可用。",
      "wechatBrowserOnly": "当前微信登录流程仅在微信内置浏览器中可用。",
      "wechatNotConfigured": "微信登录暂未开放。",
      "wechatNativeAppRequired": "当前仅配置微信移动应用登录，授权由原生 App 的微信 SDK 发起。"
    },
    "oauth": {
      "callbackHint": "返回发起授权的页面继续完成登录。",
      "invalidCallbackHint": "当前页面缺少有效的授权结果，登录页可重新发起快捷登录。"
    },
    "forgotPasswordHint": "若邮箱已注册，系统会发送密码重置链接。",
    "sendResetLinkFailed": "重置链接发送失败。",
    "resetEmailSentHint": "若该邮箱已注册，重置链接会发送到对应邮箱。",
    "resetPasswordHint": "为当前账户设置新的登录密码。",
    "newPasswordPlaceholder": "新密码",
    "confirmPasswordPlaceholder": "再次确认新密码",
    "confirmPasswordRequired": "确认密码不能为空",
    "resetPasswordFailed": "密码重置未完成。",
    "passwordResetSuccessHint": "新密码已生效，可用于登录。",
    "invalidResetLinkHint": "此密码重置链接无效或已过期。",
    "requestNewResetLink": "获取新的重置链接",
    "invalidOrExpiredToken": "密码重置链接无效或已过期。",
    "valueProps": {
      "official": "模型服务",
      "privacy": "用量记录",
      "coverage": "账户余额"
    },
    "gateway": {
      "kicker": "欢迎回到 Wegoo AI",
      "title": "欢迎回来。\n继续把想法变成产品。",
      "subtitle": "Wegoo AI 将模型、访问凭证、用量、余额和服务状态放在同一个控制台里。登录后即可创建 Key、查看价格、核对扣费并接入 SDK。",
      "routeLabel": "从应用到模型的请求路径",
      "route": {
        "application": "你的应用",
        "gateway": "统一网关",
        "models": "主流模型"
      },
      "note": "一个账户连接 GPT、Claude、Gemini 与图像模型，调用记录和费用在同一处核对。",
      "footer": "为每一次构建保持清晰",
      "mobileGreeting": "登录后继续你的工作。",
      "highlights": {
        "key": {
          "label": "统一 Key",
          "detail": "OpenAI、Claude、Gemini 与图像接口集中管理。"
        },
        "routing": {
          "label": "分组路由",
          "detail": "按服务档位选择分组，价格和可用性清晰可见。"
        },
        "billing": {
          "label": "透明账单",
          "detail": "请求、token、缓存和费用记录可追溯。"
        },
        "status": {
          "label": "状态监控",
          "detail": "模型族健康、延迟和异常提示在账户内展示。"
        }
      }
    },
    "registrationSupportTitle": "注册遇到问题？",
    "registrationSupportDesc": "加入官方群获取帮助：{contact}",
    "emailDeliveryHint": "没有收到邮件请检查垃圾箱",
    "showPassword": "显示密码",
    "hidePassword": "隐藏密码",
    "loginAgreementRequired": "当前账号尚未完成最新条款确认，暂不能登录。",
    "registerAgreementRequired": "当前注册尚未完成最新条款确认，暂不能创建账户。",
    "loginAgreementRejected": "最新条款确认尚未完成，登录保持暂停。",
    "registerAgreementRejected": "最新条款确认尚未完成，注册保持暂停。",
    "agreementPrompt": {
      "checkboxLabel": "已阅读并同意",
      "bannerTitle": "服务范围与条款确认",
      "bannerDescription": "模型服务、账户记录、访问状态和支持范围说明。",
      "openButton": "查看条款",
      "modalTitle": "服务范围与条款确认",
      "modalDescription": "{date} 更新后的条款说明平台服务范围、账户记录和账号安全边界。",
      "guaranteesLabel": "平台说明",
      "guarantees": {
        "official": "模型服务、价格和可用状态会在平台内展示。",
        "privacy": "账户只保留必要的运行、账单和支持记录。",
        "billing": "订单、用量和余额变化保留记录，便于核对。"
      },
      "documentsLabel": "相关条款",
      "documentSeparator": "、",
      "reject": "不同意",
      "accept": "同意",
      "recent": "近期"
    },
    "emailSuffixListSeparator": "、",
    "registerOnboarding": {
      "title": "账户状态",
      "description": "余额、访问凭证和接入信息在账户内呈现，模型调用会形成使用记录。",
      "steps": {
        "credit": {
          "title": "账户余额",
          "description": "体验额度和充值余额都会在仪表盘展示。"
        },
        "key": {
          "title": "访问凭证",
          "description": "访问凭证所属服务档位会关联对应费用记录。"
        },
        "connect": {
          "title": "访问详情",
          "description": "Codex、Claude Code、Gemini CLI 和 API 工具访问详情一并提供。"
        }
      }
    }
  },
  "dashboard": {
    "title": "账户概览",
    "welcomeMessage": "余额、费用、用量和服务状态一览。",
    "apiKeys": "访问凭证",
    "todayCost": "今日扣费",
    "modelDistribution": "模型用量",
    "groupDistribution": "服务档位使用分布",
    "platformBreakdown": "按模型服务拆分",
    "platformBreakdownEmpty": "暂无模型服务用量",
    "platformCount": "{count} 个服务",
    "group": "服务档位",
    "noGroup": "暂无服务档位",
    "actual": "实际扣费",
    "standard": "官方原价",
    "noUsageRecords": "暂无近期请求",
    "startUsingApi": "请求、模型和扣费记录会按时间显示在这里。",
    "viewAllUsage": "全部记录",
    "quickActions": "账户中心",
    "createApiKey": "访问凭证",
    "generateNewKey": "凭证状态与权限",
    "viewUsage": "使用记录",
    "checkDetailedLogs": "请求、模型和扣费明细",
    "addBalanceWithCode": "兑换额度记录",
    "hero": {
      "description": "余额、今日用量、访问凭证和模型服务状态在账户首页展示。",
      "note": "模型服务、访问凭证和用量明细在这里。"
    },
    "assurance": {
      "official": "模型服务",
      "noRetention": "访问凭证",
      "billingCover": "用量记录",
      "stability": "服务状态"
    },
    "gateway": {
      "kicker": "AI Gateway Console",
      "title": "一个 Key 接入多模型服务",
      "description": "余额、访问凭证、模型状态和扣费记录集中在同一个控制台，适合直接接入 OpenAI 兼容 SDK。"
    },
    "quickstart": {
      "kicker": "快速接入",
      "title": "复制 Base URL 后即可调用",
      "description": "创建访问凭证后，把 Base URL 配到现有 OpenAI SDK 或兼容客户端即可开始请求。",
      "baseUrl": "Base URL",
      "codeLabel": "Node.js 示例",
      "createKey": "创建 Key"
    },
    "loadFailed": "账户状态加载失败",
    "loadFailedDesc": "当前状态暂时不可用。",
    "retryLoad": "重新加载",
    "todayOverview": "今日概览",
    "accountSnapshot": "账户状态与用量快照",
    "usageAndSpend": "使用与费用",
    "usageAndSpendDesc": "请求、Token、实际扣费和服务状态保持同步。",
    "liveSnapshot": "实时快照",
    "balanceApproxCny": "≈ {amount} CNY",
    "keyStatus": "凭证状态",
    "activeKeys": "{count} 个启用",
    "totalUsage": "累计请求",
    "totalSpend": "累计扣费",
    "serviceStable": "正常",
    "modelStatusAction": "模型状态",
    "workspace": {
      "kicker": "账户概览",
      "title": "账户工作台",
      "description": "余额、今日调用、访问凭证和服务状态放在账户首页。",
      "balanceLabel": "当前余额",
      "balanceMeta": "充值、赠送和余额变动统一结算",
      "todayRequestsLabel": "今日调用",
      "todayRequestsMeta": "今日实扣 {cost}",
      "keysLabel": "启用凭证",
      "keysMeta": "启用 / 全部访问凭证",
      "serviceLabel": "服务状态",
      "serviceMeta": "平均响应 {latency}",
      "actionsKicker": "关键入口",
      "actionsTitle": "常用入口",
      "actionsDescription": "模型状态、用量核对、工单和充值入口在这里。",
      "actions": {
        "models": {
          "title": "查看模型状态",
          "description": "可用模型、服务档位和能力范围"
        },
        "usage": {
          "title": "查看用量扣费",
          "description": "请求、Token、倍率和实扣金额"
        },
        "support": {
          "title": "提交服务问题",
          "description": "服务失败、扣费疑问和配额问题"
        },
        "billing": {
          "title": "余额充值",
          "description": "充值、订单和余额记录"
        }
      }
    },
    "nextSteps": {
      "label": "账户下一步动作",
      "requiredBadge": "需要处理",
      "balanceBadge": "余额提醒",
      "errorBadge": "近期失败",
      "noKey": {
        "title": "先创建访问凭证",
        "description": "当前没有启用的 API Key。创建后即可复制 Base URL 并接入 SDK。",
        "action": "创建 Key"
      },
      "lowBalance": {
        "title": "余额偏低",
        "description": "当前余额已低于 {threshold}，建议先充值，避免请求中断。",
        "action": "去充值"
      },
      "recentError": {
        "title": "近期有失败请求",
        "description": "近 7 天有 {count} 条失败记录，最近涉及 {model} / {status}。先查看错误分类再决定切换分组或提交工单。",
        "action": "查看错误记录",
        "unknownModel": "未知模型"
      }
    },
    "serviceStatusHint": "请求通道当前可用，响应和缓存命中会形成明细。",
    "usageTrend": "费用与 Token 趋势",
    "usageTrendHint": "按所选时间范围呈现实际扣费、官方原价和 Token 变化。",
    "recentUsageHint": "近期请求、模型名称和实际扣费按时间保留。",
    "trustPanel": {
      "kicker": "账户信息",
      "title": "账户里的关键记录",
      "description": "模型、访问、费用和服务状态一览。",
      "items": {
        "official": {
          "title": "模型服务",
          "description": "GPT、Claude、Gemini 与图像模型按服务档位展示。"
        },
        "privacy": {
          "title": "访问与账户",
          "description": "访问凭证、分组权限和账户状态在账户内管理。"
        },
        "billing": {
          "title": "用量记录",
          "description": "模型、倍率、Token 和余额变化同步展示。"
        },
        "stability": {
          "title": "服务状态",
          "description": "服务状态和可用性在页面展示。"
        }
      }
    },
    "quickActionsKicker": "常用入口",
    "quickActionsDescription": "访问凭证、账单和订单位于同一区域。",
    "quickActionsBadge": "直达",
    "quickLinks": {
      "keys": {
        "title": "访问凭证",
        "description": "凭证状态、权限和轮换记录。"
      },
      "usage": {
        "title": "用量记录",
        "description": "请求、Token 与实际扣费明细。"
      },
      "plans": {
        "title": "余额充值",
        "description": "充值入口、订单和余额记录。"
      },
      "orders": {
        "title": "订单记录",
        "description": "支付、入账和订单状态。"
      },
      "profile": {
        "title": "账户安全",
        "description": "安全、绑定和余额提醒。"
      },
      "support": {
        "title": "支持与工单",
        "description": "扣费核对、配额和服务问题。"
      },
      "redeem": {
        "title": "兑换记录",
        "description": "兑换额度与到账记录。"
      },
      "affiliate": {
        "title": "邀请返利",
        "description": "邀请链接和返利余额。"
      }
    },
    "serviceStatus": "服务状态",
    "checkGroupHealth": "可用性、延迟和近期状态",
    "buySubscription": "余额充值",
    "manageBalanceAndPlans": "余额、订单和开票记录",
    "accountSettings": "账户设置",
    "accountSettingsHint": "提醒、安全和绑定信息",
    "affiliateInvite": "邀请好友 / 返利",
    "affiliateInviteHint": "邀请链接与返利额度",
    "modelDistributionHint": "按模型汇总请求、Token、实际扣费和官方原价。",
    "serviceTransparency": "模型服务明细",
    "serviceTransparencyDesc": "按模型服务汇总请求、Token、扣费和配额。",
    "balanceEquivalent": {
      "show": "官方额度估算",
      "hide": "收起估算",
      "title": "官方额度估算",
      "description": "按当前账户余额和可用服务档位的有效价格，折算成对应官方 API 原价美元额度。图片生成服务不参与这里的美元额度换算。",
      "rate": "{rate} 倍率",
      "quota": "官方 API 额度",
      "officialQuota": "官方美元额度",
      "officialAmount": "官方 {amount}",
      "apiFormula": "公式：账户余额 {balance} ÷ 有效价格系数 {rate} = 对应官方原价额度 {quota}",
      "empty": "暂无可用服务档位"
    },
    "retention": {
      "lowBalanceBanner": {
        "criticalTitle": "账户余额为 0",
        "lowTitle": "账户余额低于提醒线",
        "balanceLabel": "当前 {balance}",
        "criticalMessage": "当前余额为 0。",
        "lowMessage": "当前余额不高于 {threshold}。",
        "primaryAction": "充值",
        "docsAction": "接入教程",
        "usageAction": "查看用量",
        "dismiss24h": "24 小时内隐藏"
      }
    }
  },
  "keys": {
    "title": "访问凭证",
    "description": "访问凭证、分组、额度和接入信息一览。",
    "searchPlaceholder": "搜索名称或凭证...",
    "endpoints": {
      "title": "服务端点",
      "clickToCopy": "复制端点"
    },
    "createKey": "新增访问凭证",
    "editKey": "编辑访问凭证",
    "deleteKey": "删除访问凭证",
    "apiKey": "访问凭证",
    "noGroup": "暂无分组",
    "namePlaceholder": "我的访问凭证",
    "noKeysYet": "暂无访问凭证",
    "createFirstKey": "当前账户暂无访问凭证。",
    "keyCreatedSuccess": "访问凭证已新增",
    "keyUpdatedSuccess": "访问凭证更新成功",
    "keyDeletedSuccess": "访问凭证删除成功",
    "keyEnabledSuccess": "访问凭证已启用",
    "keyDisabledSuccess": "访问凭证已禁用",
    "failedToLoad": "加载访问凭证失败",
    "failedToSave": "保存访问凭证失败",
    "failedToDelete": "删除访问凭证失败",
    "failedToUpdateStatus": "更新访问凭证状态失败",
    "clickToChangeGroup": "更换分组",
    "useKey": "访问详情",
    "useKeyModal": {
      "title": "访问详情",
      "description": "当前访问凭证的客户端接入信息已整理在下方。",
      "note": "环境变量内容已按当前访问凭证生成。",
      "noGroupTitle": "尚未分配分组",
      "noGroupDescription": "分配分组后会显示对应客户端访问详情。",
      "openai": {
        "description": "Codex CLI 访问配置。",
        "configTomlHint": "config.toml 配置",
        "note": "目录信息：~/.codex。",
        "noteWindows": "Windows 目录信息：%userprofile%\\.codex。"
      },
      "cliTabs": {
        "openaiSdk": "OpenAI SDK",
        "anthropicSdk": "Anthropic SDK"
      },
      "antigravity": {
        "description": "Antigravity 分组的 API 访问详情。",
        "claudeNote": "环境变量内容已按当前访问凭证生成。",
        "geminiNote": "环境变量内容已按当前访问凭证生成。"
      },
      "gemini": {
        "description": "Gemini CLI 访问所需环境变量。",
        "modelComment": "Gemini 3 可用时可填：gemini-3-pro-preview",
        "note": "环境变量内容已按当前访问凭证生成。"
      },
      "opencode": {
        "title": "OpenCode 访问示例",
        "hint": "OpenCode 配置位置：~/.config/opencode/opencode.json（或 opencode.jsonc）。生成内容包含 apiKey、baseURL、模型 limit/options/variants 等关键参数。"
      },
      "connection": {
        "group": "分组",
        "groupHint": "当前访问凭证只按这个分组路由。",
        "platform": "接口类型",
        "platformHint": "下方示例会按接口类型切换客户端。",
        "endpoints": "可用端点",
        "endpointCount": "{count} 个端点",
        "primaryEndpoint": "默认端点",
        "customEndpoint": "自定义端点",
        "defaultEndpoint": "默认"
      },
      "sdk": {
        "description": "SDK 接入示例，包含当前访问凭证和 Base URL。",
        "note": "示例只展示最小调用方式；实际模型和价格以当前分组展示为准。",
        "openaiHint": "OpenAI SDK 示例",
        "anthropicHint": "Anthropic SDK 示例"
      }
    },
    "customKeyLabel": "自定义访问凭证",
    "customKeyPlaceholder": "输入自定义访问凭证（至少16个字符）",
    "customKeyTooShort": "自定义访问凭证至少需要16个字符",
    "customKeyInvalidChars": "自定义访问凭证只能包含字母、数字、下划线和连字符",
    "customKeyRequired": "请输入自定义访问凭证",
    "ipWhitelistHint": "每行一个 IP 或 CIDR，设置后仅允许这些 IP 使用此访问凭证",
    "ipBlacklistHint": "每行一个 IP 或 CIDR，这些 IP 将被禁止使用此访问凭证",
    "ipRestrictionEnabled": "IP 限制已启用",
    "ccSwitchNotInstalled": "未检测到 CC Switch 协议处理程序，可使用手动复制方式。",
    "ccsClientSelect": {
      "description": "CC Switch 可用客户端类型：",
      "claudeCodeDesc": "Claude Code 访问",
      "geminiCliDesc": "Gemini CLI 访问"
    },
    "quotaAmountHint": "设置此访问凭证可消费的最大金额。0 = 无限制。",
    "resetQuotaConfirmMessage": "确定要将访问凭证 \"{name}\" 的已用额度（{used}）重置为 0 吗？此操作不可撤销。",
    "rateLimitHint": "设置此访问凭证在指定时间窗口内的最大消费额。0 = 无限制。",
    "resetRateLimitConfirmMessage": "确定要重置访问凭证 \"{name}\" 的速率限制用量吗？所有时间窗口的已用额度将归零。此操作不可撤销。",
    "expiration": "凭证有效期",
    "expirationDateHint": "选择此访问凭证的过期时间。",
    "hero": {
      "title": "访问凭证",
      "description": "服务接入、可用分组、额度与消耗明细一并呈现。",
      "signals": {
        "official": "模型档位",
        "billing": "扣费明细",
        "status": "服务状态"
      }
    },
    "integration": {
      "kicker": "Gateway Credentials",
      "description": "创建 Key 后即可接入 OpenAI 兼容接口；分组、额度、状态和用量都在这里管理。",
      "totalKeys": "全部 Key",
      "activeKeys": "启用中",
      "baseUrl": "Base URL",
      "codeLabel": "OpenAI SDK 接入"
    },
    "category": "服务类型",
    "categoryLabel": "服务类型",
    "categoryHint": "按所选分组的模型服务展示，仅用于选择分组。",
    "selectCategory": "选择服务类型",
    "allCategories": "全部平台",
    "categories": {
      "openai": "OpenAI",
      "anthropic": "Anthropic",
      "other": "其他"
    },
    "balanceExplain": {
      "title": "余额和官方价格怎么对应？",
      "description": "1 元按 1 刀余额展示；实际扣费约等于官方模型价格 × 当前分组倍率。倍率越低，同样余额能使用的官方原价额度越多。"
    },
    "groupChoiceTips": {
      "cost": {
        "title": "费用明细",
        "description": "不同分组展示各自倍率和费用明细。"
      },
      "stability": {
        "title": "服务状态",
        "description": "服务状态与访问凭证相邻展示。"
      },
      "image": {
        "title": "模型能力区分明确",
        "description": "文本、代码、多模态和图像服务按分组呈现。"
      }
    },
    "groupMatrix": {
      "title": "按服务类型选择分组",
      "description": "先按模型服务分组，再选择具体分组；下方搜索框可继续精确查找。",
      "count": "{count} 个分组",
      "empty": "暂无分组"
    },
    "groupCostPreview": {
      "title": "{group} 计费预览",
      "description": "当前有效倍率约 {rate}，按 USD/CNY≈{cny} 估算，约为官方 API 计费的 {multiple} 倍。",
      "note": "这是帮助理解的估算；实际扣费以页面展示的模型价格、当前倍率和使用记录为准。",
      "unavailable": "当前分组倍率暂时无法计算，请以分组说明和使用记录为准。"
    },
    "serviceStatusTip": {
      "title": "服务状态与接入信息",
      "description": "可用率、延迟和近期状态与分组对应展示。",
      "action": "查看服务状态"
    },
    "viewServiceStatus": "查看服务状态",
    "overview": "凭证概览",
    "cost": "消费",
    "unlimited": "无限制",
    "neverUsed": "未使用",
    "ccSwitchDialog": {
      "title": "导入到 CCS",
      "description": "按当前访问凭证和分组展示可用客户端访问详情。",
      "currentGroup": "当前分组",
      "noGroup": "分配分组后会显示 CC Switch 访问详情。",
      "endpoint": "端点",
      "model": "模型",
      "protocol": "协议",
      "reasoning": "推理",
      "import": "导入",
      "copyLink": "复制链接",
      "linkCopied": "链接已复制",
      "openOfficial": "打开 CC Switch",
      "installHint": "协议链接和访问链接已生成，可在客户端侧查看状态。",
      "protocols": {
        "anthropicMessages": "Anthropic Messages",
        "openaiResponses": "OpenAI Responses",
        "geminiNative": "Gemini Native",
        "openaiCompatible": "OpenAI Compatible",
        "openaiCompletions": "OpenAI Completions",
        "chatCompletions": "Chat Completions"
      },
      "targets": {
        "claudeCode": {
          "name": "Claude Code",
          "description": "Claude Code 服务访问详情，使用 Anthropic Messages。"
        },
        "claudeDesktop": {
          "name": "Claude Desktop",
          "description": "CC Switch Claude 服务访问详情，使用 Anthropic Messages。"
        },
        "codex": {
          "name": "Codex",
          "description": "Codex auth.json 和 config.toml 访问详情，CC Switch 原生使用 Responses。"
        },
        "geminiCli": {
          "name": "Gemini CLI",
          "description": "Gemini CLI 环境访问详情，使用 Gemini Native v1beta 端点。"
        },
        "opencode": {
          "name": "OpenCode",
          "description": "OpenCode provider；CC Switch deeplink 当前使用 OpenAI Compatible。"
        },
        "openclaw": {
          "name": "OpenClaw",
          "description": "OpenClaw provider；CC Switch deeplink 当前使用 OpenAI Completions。"
        },
        "hermes": {
          "name": "Hermes Agent",
          "description": "Hermes 自定义 provider；CC Switch deeplink 当前使用 chat_completions。"
        }
      }
    },
    "statusToggle": {
      "on": "开启",
      "off": "关闭"
    }
  },
  "usage": {
    "description": "模型、Token、费用和请求状态一览。",
    "apiKeyFilter": "访问凭证",
    "allApiKeys": "全部凭证",
    "upstreamModel": "实际模型",
    "upstream": "模型服务",
    "upstreamEndpoint": "服务端点",
    "imageSizeSourceOutput": "服务输出",
    "noRecords": "未找到使用记录，可调整筛选条件。",
    "tabs": {
      "errors": "问题记录"
    },
    "errors": {
      "status": "服务状态码",
      "platform": "模型服务",
      "message": "问题摘要",
      "keyName": "凭证名称",
      "allKeys": "全部凭证",
      "empty": "暂无问题记录",
      "failedToLoad": "加载问题记录失败",
      "categories": {
        "invalid_request": "请求参数",
        "upstream": "模型服务返回",
        "internal": "平台问题"
      },
      "detail": {
        "title": "问题记录详情",
        "responseBody": "服务返回内容",
        "upstreamStatus": "服务状态码"
      },
      "title": "问题记录"
    },
    "gateway": {
      "kicker": "使用记录",
      "auditKicker": "用量概览",
      "auditTitle": "当前筛选",
      "auditDescription": "时间范围、访问凭证、缓存命中和问题记录开关会影响下方明细与导出结果。",
      "dateRange": "时间范围",
      "credential": "访问凭证",
      "issueRecords": "问题记录",
      "issueRecordsOn": "已开启",
      "issueRecordsOff": "未开启"
    },
    "trust": {
      "transparentUsage": "用量记录",
      "transparentUsageDesc": "每次请求按时间、模型、Token 与费用列明。",
      "auditableBilling": "费用明细",
      "auditableBillingDesc": "实际扣费与标准费用分开展示，支持导出。",
      "recoverableIssues": "请求状态",
      "recoverableIssuesDesc": "失败请求保留状态与摘要，便于处理。"
    },
    "inputCost": "输入费用",
    "outputCost": "输出费用",
    "cacheCreationCost": "缓存写入费用",
    "cacheReadCost": "缓存读取费用",
    "inputTokens": "输入 Token",
    "outputTokens": "输出 Token",
    "cacheCreationTokens": "缓存写入 Token",
    "cacheCreation5mTokens": "缓存写入",
    "cacheCreation1hTokens": "缓存写入",
    "cacheReadTokens": "缓存读取 Token",
    "billingMode": "计费方式",
    "billingModeToken": "按 Token",
    "billingModePerRequest": "按次",
    "billingModeImage": "按图片",
    "billingModeVideo": "按视频",
    "allModels": "全部模型",
    "allTypes": "全部类型",
    "requestsShort": "次",
    "openaiCacheCreateNote": "GPT/OpenAI 只返回缓存命中 token，不返回“缓存创建”字段；缓存创建为 0 属于正常口径。",
    "openaiCacheCreateShortNote": "提示：GPT/OpenAI 不返回缓存创建字段，显示 0 属正常。",
    "exportHeaders": {
      "time": "时间",
      "credentialName": "访问凭证名称",
      "group": "分组",
      "model": "模型",
      "reasoningEffort": "推理强度",
      "inboundEndpoint": "入站端点",
      "type": "类型",
      "billingMode": "计费方式",
      "inputTokens": "输入 Token",
      "outputTokens": "输出 Token",
      "cacheReadTokens": "缓存读取 Token",
      "cacheCreationTokens": "缓存写入 Token",
      "totalTokens": "总 Token",
      "rateMultiplier": "倍率",
      "billedCost": "实际扣费（{currency}）",
      "originalCost": "标准费用（{currency}）",
      "firstTokenMs": "首 Token（ms）",
      "durationMs": "耗时（ms）"
    },
    "insights": {
      "analytics": "用量分析",
      "analyticsTitle": "模型、分组、端点与 Token 趋势",
      "modelRanking": "模型排行",
      "groupDistribution": "分组分布",
      "avgCost": "单次均费",
      "noBreakdown": "暂无分布数据",
      "noModel": "暂无模型",
      "noGroup": "暂无分组",
      "noEndpoint": "暂无端点"
    },
    "copyDiagnostic": "复制诊断",
    "diagnosticCopied": "诊断信息已复制",
    "diagnostic": {
      "requestId": "请求 ID",
      "usageId": "用量记录 ID",
      "model": "模型",
      "apiKey": "访问凭证",
      "group": "分组",
      "endpoint": "端点",
      "type": "请求类型",
      "tokens": "Token",
      "cost": "费用",
      "latency": "延迟",
      "createdAt": "请求时间"
    }
  },
  "monitorCommon": {
    "status": {
      "error": "异常"
    }
  },
  "channelStatus": {
    "title": "模型服务状态",
    "description": "按模型服务和服务档位展示可用率、延迟与近期状态。",
    "searchPlaceholder": "搜索模型或服务...",
    "allProviders": "全部服务",
    "loadError": "加载服务状态失败",
    "detailLoadError": "加载服务详情失败",
    "detailTitle": "模型服务详情",
    "overall": {
      "operational": "运行正常",
      "degraded": "有波动",
      "unavailable": "暂不可用"
    },
    "columns": {
      "provider": "模型服务",
      "groupName": "服务档位"
    },
    "empty": {
      "title": "暂无状态数据",
      "description": "服务状态数据生成后会在这里展示。"
    },
    "hero": {
      "title": "服务状态",
      "description": "可用率、延迟和近期响应按时间呈现。",
      "assurance": "服务状态页面呈现模型服务可用性、延迟、近期响应和服务档位状态。",
      "signals": {
        "stability": "服务状态",
        "full": "模型状态",
        "transparent": "响应走势",
        "privacy": "近期状态",
        "billing": "状态时间线"
      }
    },
    "gateway": {
      "kicker": "Service Health",
      "description": "查看各模型服务的可用性、响应速度和近期状态。",
      "totalServices": "服务项",
      "totalServicesMeta": "当前可见服务项",
      "operationalServices": "运行正常",
      "operationalServicesMeta": "近期状态正常",
      "degradedServices": "需要关注",
      "noIncidents": "暂无异常服务",
      "needAttention": "存在波动或错误",
      "avgLatency": "平均延迟",
      "avgLatencyMeta": "{window} 窗口"
    }
  },
  "availableChannels": {
    "title": "可用模型与价格",
    "description": "可用分组、支持模型、价格倍率和模型服务信息一览。",
    "searchPlaceholder": "搜索模型、分组或服务...",
    "empty": "暂无匹配的模型或服务",
    "noModels": "暂无可用模型",
    "noPricing": "暂无价格信息",
    "exclusive": "专属可用",
    "public": "公开可用",
    "exclusiveTooltip": "已为你开通的专属服务",
    "publicTooltip": "当前公开可用的服务",
    "columns": {
      "name": "服务名",
      "platform": "模型服务",
      "groups": "服务档位",
      "endpoints": "端点能力",
      "model": "模型"
    },
    "pricing": {
      "billingMode": "计价方式",
      "billingModeToken": "按 token",
      "intervals": "阶梯价格"
    },
    "eyebrow": "模型列表",
    "assurance": "可用服务、服务档位、价格倍率和支持模型一并呈现。",
    "gateway": {
      "kicker": "Model Catalog",
      "description": "按可用服务档位展示模型、倍率和价格信息。"
    },
    "filters": {
      "allPlatforms": "全部平台"
    },
    "trustSignals": {
      "full": "模型列表",
      "stable": "服务状态",
      "noRetention": "支持模型",
      "privacy": "可用分组",
      "transparent": "价格和倍率",
      "billing": "用量明细"
    },
    "channel": "模型服务",
    "visible": "可用",
    "defaultServiceDescription": "该模型服务当前可用，分组、倍率和支持模型一并呈现。",
    "serviceSummary": "服务概览",
    "groupsTitle": "可用分组",
    "modelsTitle": "可用模型",
    "platformSectionTitle": "可用分组",
    "groupModelsTitle": "可用模型",
    "modelTable": {
      "title": "模型价格表",
      "selectGroup": "选择分组",
      "effectiveRate": "当前倍率",
      "inputPrice": "输入价格",
      "outputPrice": "输出价格",
      "cacheWritePrice": "缓存写入",
      "cacheReadPrice": "缓存读取",
      "unitPrice": "图片 / 按次",
      "unitPerImage": "/ 张",
      "noUnitPrice": "无按次价格",
      "vendorTitle": "厂商分类",
      "modelCount": "{count} 个模型",
      "scrollLeft": "向左查看更多分组",
      "scrollRight": "向右查看更多分组",
      "groupSummary": "共 {groups} 个可用分组，当前展示 {models} 个模型"
    },
    "subscription": "订阅可用",
    "stats": {
      "channels": "模型来源",
      "platforms": "平台",
      "groups": "可用分组",
      "models": "可用模型"
    }
  },
  "affiliate": {
    "description": "邀请链接、返利比例、可转余额和邀请关系一览。",
    "stats": {
      "rebateRateHint": "被邀请账号每次充值后产生的返利比例",
      "invitedUsersHint": "已通过你的链接完成注册的账号",
      "availableQuotaHint": "可转入账户余额的返利额度",
      "frozenQuotaLine": "另有 {amount} 正在冻结中",
      "totalQuotaHint": "累计产生的返利额度"
    },
    "transfer": {
      "description": "将当前可用返利额度转入账户余额"
    },
    "bindFailed": "绑定邀请码失败",
    "hero": {
      "kicker": "邀请体系",
      "title": "邀请链接和返利明细",
      "description": "邀请链接、绑定状态、返利比例、冻结额度和可转余额一览。",
      "pillTrial": "体验额度可见",
      "pillCheap": "返利比例可查",
      "pillStable": "明细保留",
      "shareHint": "Codex、Cursor、Claude、Gemini 和 API 工具相关信息可用于分享。"
    },
    "sharePanel": {
      "kicker": "专属邀请信息",
      "title": "专属邀请信息",
      "description": "邀请链接、邀请码和可复制文案在这里。"
    },
    "bind": {
      "title": "绑定邀请码领取礼包",
      "description": "邀请绑定窗口为注册后 1 天内。绑定状态、礼包额度和领取状态会显示在这里。",
      "codePlaceholder": "输入邀请码",
      "bonusHint": "绑定成功后可手动领取 {amount}",
      "button": "绑定邀请码",
      "binding": "绑定中...",
      "success": "邀请码绑定成功",
      "successWithBonus": "邀请码绑定成功，礼包状态已更新",
      "claimTitle": "领取绑定礼包",
      "claimDescription": "当前账号已绑定邀请人，{amount} 领取后会加入账户余额。",
      "claimButton": "领取 {amount}",
      "claiming": "领取中...",
      "claimSuccess": "绑定礼包已领取：{amount}",
      "errors": {
        "AFFILIATE_BIND_WINDOW_EXPIRED": "注册超过 1 天后不能再绑定邀请信息。",
        "AFFILIATE_CODE_INVALID": "邀请码无效，请检查后重试。",
        "AFFILIATE_ALREADY_BOUND": "当前账号已经绑定过邀请人。",
        "AFFILIATE_BIND_BONUS_UNAVAILABLE": "当前没有可领取的绑定礼包。",
        "AFFILIATE_BIND_BONUS_ALREADY_CLAIMED": "绑定礼包已经领取过。"
      }
    },
    "rules": {
      "title": "返利规则说明",
      "line1": "邀请码和邀请链接会记录邀请关系。",
      "line2": "被邀请账号充值后，会产生 {rate} 的返利额度。",
      "durationPermanent": "返利有效期：永久有效。邀请关系绑定后，后续充值按当前规则产生返利。",
      "durationLimited": "返利有效期：注册后 {days} 天内的充值会产生返利，超过后充值不再产生返利。",
      "line4": "可用返利额度可转入账户余额。",
      "line5": "新产生的返利需要经过冻结期后才能转入余额。"
    },
    "friendBenefits": {
      "kicker": "邀请权益",
      "title": "邀请权益",
      "description": "体验额度、模型范围、价格明细和服务状态一览。",
      "trialTitle": "体验额度",
      "trialDescription": "体验额度和余额变动会显示在账户记录中。",
      "valueTitle": "价格明细",
      "valueDescription": "服务档位、模型价格、倍率和扣费明细一并呈现。",
      "stabilityTitle": "状态可见",
      "stabilityDescription": "模型服务状态和近期可用性保持可查。",
      "rangeTitle": "模型范围",
      "rangeDescription": "GPT、Claude、Gemini 和图像模型统一呈现。"
    },
    "audiences": {
      "kicker": "可复制展示信息",
      "title": "可复制展示信息",
      "description": "围绕模型范围、价格明细和服务状态的展示文案。",
      "copyButton": "复制文案",
      "copied": "文案已复制",
      "developerTitle": "给开发者 / Codex 用户",
      "developerDescription": "适合 Codex、Cursor、Cline 或 API 工具场景。",
      "developerCopy": "WegooAI 支持 GPT、Claude、Gemini 和 API 工具接入，可查看模型状态、价格、余额和用量明细。邀请链接：{link}",
      "heavyUserTitle": "给 GPT / Claude 场景",
      "heavyUserDescription": "适合持续使用 GPT、Claude 或 Gemini 的场景。",
      "heavyUserCopy": "WegooAI 支持 GPT、Claude、Gemini 和生图，可查看模型状态、价格、余额和用量明细。邀请链接：{link}",
      "newcomerTitle": "AI 入门场景",
      "newcomerDescription": "适合查看模型范围、体验额度和账户记录的场景。",
      "newcomerCopy": "WegooAI 支持 GPT、Claude、Gemini 和生图，账户内可查看模型状态、价格、余额和用量明细。邀请链接：{link}",
      "groupTitle": "发到群里 / 社区",
      "groupDescription": "适合开发者群、工具群或 AI 交流群展示。",
      "groupCopy": "WegooAI 支持 GPT、Claude、Gemini 和生图，也支持 Codex、Cursor 和 API 工具接入；账户内可查看模型状态、价格、余额和用量明细。邀请链接：{link}"
    },
    "promo": {
      "kicker": "可复制展示信息",
      "title": "模型、价格和用量",
      "description": "平台支持 GPT、Claude、Gemini 和图像模型，并显示模型状态、价格、余额和用量明细。",
      "highlightTrial": "体验额度可见",
      "highlightCheap": "价格和倍率可查",
      "highlightRouting": "服务状态可见",
      "previewTitle": "默认分享文案",
      "copyButton": "复制文案",
      "copied": "文案已复制",
      "shareText": "WegooAI 支持 GPT、Claude、Gemini 和生图，账户内可查看模型状态、价格、余额和用量明细。邀请链接：{link}"
    }
  },
  "redeem": {
    "description": "兑换权益、余额变化和历史明细一览。",
    "redeemCodePlaceholder": "兑换码",
    "redeeming": "处理中",
    "redeemButton": "确认兑换",
    "redeemSuccess": "兑换已完成",
    "redeemFailed": "兑换未完成",
    "aboutCodes": "兑换说明",
    "codeRule3": "兑换问题可通过工单处理",
    "historyWillAppear": "兑换历史会显示在这里",
    "balanceAddedAdmin": "余额充值（平台调整）",
    "balanceDeductedAdmin": "余额扣除（平台调整）",
    "concurrencyAddedAdmin": "并发增加（平台调整）",
    "concurrencyReducedAdmin": "并发减少（平台调整）",
    "adminAdjustment": "平台调整",
    "subscriptionAssignedDesc": "您已获得 {groupName} 的服务权限",
    "codeRedeemSuccess": "兑换已完成",
    "failedToRedeem": "兑换未完成，兑换码或当前权益状态未通过校验。",
    "pleaseEnterCode": "兑换码不能为空",
    "trust": {
      "transparentBalance": "余额记录",
      "transparentBalanceDesc": "兑换后的余额、并发和订阅权益会同步展示。",
      "auditableHistory": "历史记录",
      "auditableHistoryDesc": "历史兑换、平台调整和订阅分配会保留为记录。",
      "recoverableIssues": "处理记录",
      "recoverableIssuesDesc": "兑换失败或权益不一致时，可关联明细处理。"
    },
    "assurance": {
      "official": "可用模型和服务档位会在账户中展示。",
      "singleUse": "每个兑换码只会结算一次。",
      "coverage": "余额、并发和订阅权益使用同一套变动口径。",
      "privacy": "兑换记录和到账状态会保留在账户内。",
      "instantUpdate": "兑换成功后账户状态会即时更新。"
    },
    "contactLine": "支持联系"
  },
  "profile": {
    "description": "账户资料、登录绑定、安全设置和提醒状态一览。",
    "overviewDescription": "账号状态、资料来源与常用设置。",
    "basicsDescription": "头像与昵称等公开展示信息。",
    "linkedProfileSourcesDescription": "部分头像和昵称按第三方登录方式同步展示。",
    "securityDescription": "密码、双因素认证和通知提醒。",
    "administrator": "平台支持",
    "totp": {
      "featureDisabledHint": "双因素认证暂未开放",
      "disableWarning": "禁用后，登录时将不再需要验证码，登录保护状态会同步更新。",
      "loginHint": "认证器应用中的 6 位验证码",
      "loginFailed": "验证未通过",
      "verifyEmailFirst": "邮箱验证完成后可继续",
      "verifyPasswordFirst": "身份验证完成后可继续",
      "codeSent": "验证码已发送到您的邮箱。没有收到邮件请检查垃圾箱"
    },
    "balanceNotify": {
      "extraEmailsHint": "已验证邮箱会接收余额不足提醒邮件",
      "codeSent": "验证码已发送。没有收到邮件请检查垃圾箱",
      "codeSentTo": "验证码已发送到 {email}。没有收到邮件请检查垃圾箱"
    },
    "avatar": {
      "uploadRequired": "请选择头像图片"
    },
    "authBindings": {
      "description": "当前绑定状态和可用的第三方登录方式。",
      "codeSentTo": "验证码已发送到 {email}。没有收到邮件请检查垃圾箱",
      "notes": {
        "canUnbind": "此登录方式支持解除绑定。",
        "bindAnotherBeforeUnbind": "保留至少一种登录方式后，可解除当前绑定。"
      }
    },
    "gateway": {
      "kicker": "Account Control",
      "panelTitle": "登录与安全边界",
      "securityScope": "Security Scope",
      "accountId": "账户 ID",
      "securityMeta": "账户级权限",
      "supportRole": "支持人员"
    },
    "trust": {
      "privacyBoundary": "登录资料",
      "privacyBoundaryDesc": "第三方资料来源只显示必要状态，不展示登录凭据。",
      "auditableBindings": "绑定记录",
      "auditableBindingsDesc": "邮箱和登录方式的绑定状态一览。",
      "accountSecurity": "账户安全",
      "accountSecurityDesc": "密码、余额提醒和双因素认证在账户侧处理。"
    }
  },
  "errors": {
    "pageNotFoundDescription": "你访问的页面不存在，或已经被移动。",
    "goBack": "返回上一页",
    "goHome": "回到首页"
  },
  "imageGeneration": {
    "title": "AI 生图",
    "description": "生图服务档位、可用模型、价格预估和生成结果一览。",
    "hero": {
      "kicker": "Image Studio",
      "title": "AI 生图",
      "description": "仅展示当前账户已开放生图的服务档位，模型、单张扣费、本次预估和余额一并展示。",
      "assurance": "生图能力按已开放服务档位呈现，模型、尺寸、单张扣费、本次预估和生成状态一并显示。",
      "signals": {
        "official": "模型可选",
        "noRetention": "明细可查",
        "privacy": "参考图本地预览",
        "billing": "扣费明细",
        "enabled": "按开放档位展示",
        "stable": "服务状态可见"
      }
    },
    "apiKey": "生图访问",
    "selectKey": "选择生图访问",
    "noImageKey": "当前没有可用的生图访问能力。",
    "createImageKey": "生图访问状态",
    "group": "生图服务档位",
    "selectGroup": "选择生图服务档位",
    "noImageGroup": "当前没有可用的生图服务档位。",
    "autoAccessName": "生图访问 · {group}",
    "autoCreatingKeys": "正在准备这个服务档位的生图访问能力...",
    "groupKeyUnavailable": "这个服务档位的生图访问能力暂未就绪。",
    "autoCreatedKey": "{group} 的生图访问能力已就绪",
    "autoCreateKeyFailed": "{group} 的生图访问能力暂未准备完成",
    "model": "模型",
    "selectModel": "选择生图模型",
    "noImageModel": "这个服务档位暂无可用图片模型。",
    "size": "尺寸",
    "geminiFlashSizeHint": "当前所选 Gemini 生图模型仅开放 1K，已隐藏不可用的 2K/4K 选项。",
    "quality": "质量",
    "count": "数量",
    "promptPlaceholder": "输入你想生成的画面，也可以粘贴或拖入参考图",
    "editPlaceholder": "描述你希望如何修改参考图",
    "uploadReference": "上传参考图",
    "addReference": "添加参考图",
    "dropImages": "松开上传图片",
    "generateMode": "文字生图",
    "editMode": "参考图编辑",
    "submitGenerate": "生成图片",
    "submitEdit": "编辑图片",
    "continueEdit": "继续编辑",
    "generating": "生成中",
    "empty": "生成结果会显示在这里",
    "loadKeysFailed": "加载生图访问能力失败",
    "invalidReference": "请选择图片文件",
    "referenceTooLarge": "单张参考图不能超过 10MB",
    "referenceLimit": "最多可添加 {count} 张参考图",
    "noImagesReturned": "暂未收到图片结果",
    "generatedSuccess": "图片生成完成",
    "generateFailed": "图片生成失败",
    "errorSupportHint": "生成问题可通过工单处理；请求状态会用于定位。",
    "networkDisconnected": "图片生成耗时较长，浏览器连接已断开。图片可能仍在处理中；用量明细和生成状态会继续更新。",
    "cancelled": "已取消生成",
    "referenceAdded": "已加入参考图",
    "referenceFromResultFailed": "读取结果图失败，无法加入编辑",
    "localCacheWarning": "最近 5 张生成图片会在当前浏览器保留；清理站点数据后不再显示。",
    "qualityOptions": {
      "auto": "自动",
      "low": "低",
      "medium": "中",
      "high": "高"
    },
    "pricing": {
      "title": "价格预估",
      "unitCost": "单张扣费",
      "batchCost": "本次预估",
      "remainingImages": "还能生成",
      "balance": "当前余额",
      "noEstimate": "选择生图服务档位后显示价格",
      "noKey": "服务访问暂未就绪",
      "autoCreatingKey": "正在准备服务访问",
      "groupPrice": "档位价",
      "defaultPrice": "默认价",
      "balanceLimited": "按余额估算",
      "keyQuotaLimited": "受服务访问额度限制",
      "summary": "{source} {base} × 倍率 {multiplier} · {tier}",
      "imageCountValue": "{count} 张"
    }
  },
  "admin": {
    "dashboard": {
      "userUsageTrend": "用户用量趋势（前 12）",
      "err": "异常"
    },
    "users": {
      "roles": {
        "support": "客服"
      },
      "form": {
        "supportRoleHint": "客服只能进入工单后台：接单、回复普通工单，无法管理用户、余额和系统设置。"
      },
      "support": "客服",
      "passwordCopied": "密码已复制",
      "businessCategory": "业务分类"
    },
    "announcements": {
      "form": {
        "conditionUser": "指定用户",
        "selectUsers": "选择接收用户",
        "selectedUsers": "已选择 {count} / {max} 个用户",
        "userSearchPlaceholder": "搜索邮箱、用户名或用户 ID",
        "userSearchEmpty": "没有找到匹配的用户",
        "userSearchFailed": "用户搜索失败，请重试",
        "userSelectionLimit": "单个条件最多选择 {max} 个用户",
        "removeUser": "移除用户",
        "invalidTargeting": "请先完善站内信投放条件"
      }
    },
    "userPricing": {
      "kicker": "折扣管理",
      "title": "用户折扣与加价",
      "description": "先选择用户，再一次性维护该用户所有标准计费分组的折扣或加价系数。低于 1 为折扣，高于 1 为加价。",
      "group": "分组",
      "groupRate": "分组倍率",
      "fixedRate": "固定倍率",
      "user": "用户",
      "searchUser": "查找要编辑或新增定价的用户",
      "selectUser": "选择用户后管理该用户的分组定价",
      "select": "选择",
      "findUserPlaceholder": "邮箱、用户 ID 或昵称",
      "configuredUsersTitle": "定价有变动的用户",
      "configuredUsersDescription": "这里只列出实际生效倍率与分组默认倍率不同的用户，点击编辑即可集中修改该用户的全部分组配置。",
      "configuredUsersCount": "共 {count} 个有变动用户",
      "noConfiguredUsers": "暂无用户存在专属定价变动",
      "configuredGroups": "变动分组",
      "configuredCountLabel": "变动数",
      "fixedShort": "固定",
      "coefficientShort": "系数",
      "edit": "编辑",
      "coefficient": "计费系数",
      "finalRate": "最终倍率",
      "status": "状态",
      "actions": "操作",
      "groupTableTitle": "标准分组定价",
      "groupTableDescription": "在一个表格里维护该用户所有余额计费分组的折扣或加价。已停用分组也会保留显示，避免隐藏历史配置。",
      "groupCount": "共 {count} 个分组",
      "configuredCount": "已配置 {count} 个",
      "active": "启用",
      "inactive": "停用",
      "default": "默认",
      "discount": "折扣",
      "markup": "加价",
      "fixedRateLocked": "固定倍率优先",
      "clearRow": "清除此系数",
      "emptyGroups": "暂无标准计费分组",
      "invalidCoefficient": "系数必须在 0.0001 到 100 之间",
      "saved": "用户定价系数已保存并立即生效",
      "clearUser": "清空该用户折扣",
      "clearConfirm": "确认清空「{user}」所有分组的折扣和加价系数？固定倍率不会受到影响。",
      "fixedConflictTitle": "固定倍率优先",
      "fixedConflictDescription": "固定最终倍率仅作参考，优先级高于相对系数，不能通过此处编辑。"
    },
    "upstreams": {
      "title": "上游管理",
      "description": "统一维护上游站点、真实钱包、分组倍率，以及各平台运行账号的绑定关系。",
      "search": "搜索上游名称或 Base URL",
      "add": "添加上游",
      "edit": "编辑上游",
      "empty": "暂无上游，添加后可探测分组、倍率和支持模型。",
      "saved": "上游已保存",
      "savedProbeFailed": "上游已保存，但首次探测未完成：{error}",
      "deleted": "上游已删除",
      "deleteConfirm": "确认删除上游「{name}」？存在绑定账号时需要先解绑。",
      "probe": "立即探测",
      "probeDone": "上游探测已完成",
      "openDetail": "打开详情",
      "notAvailable": "未获取",
      "notProbed": "尚未探测",
      "notBound": "未绑定",
      "renameAccounts": "统一自动命名",
      "renameDialog": {
        "title": "统一命名已绑定账号",
        "willRename": "将改名",
        "willSkip": "将跳过",
        "rename": "改名",
        "apply": "确认应用",
        "applying": "应用中...",
        "resultDone": "已自动命名 {count} 个账号",
        "resultPartial": "已改名 {renamed} 个账号，{failed} 个账号更新失败",
        "reasons": {
          "groupNotVerified": "上游分组尚未验证",
          "alreadyNamed": "已经是自动名称",
          "updateFailed": "账号更新失败",
          "unknown": "不符合自动命名条件"
        }
      },
      "keyUnlimited": "Key 额度无限",
      "refreshPending": "等待后台首次刷新",
      "refreshFresh": "后台已刷新 · {time}",
      "refreshPartial": "部分刷新，本次有数据未获取 · {time}",
      "refreshStale": "刷新失败，未展示历史钱包或倍率 · {time}",
      "refreshFailed": "刷新失败，暂时没有已验证数据 · {time}",
      "failureReasonsTitle": "失败原因",
      "failureSummaryMore": "另有 {count} 项失败",
      "failureScopeManagement": "管理探测",
      "failureScopeProtocol": "{platform} 协议探测",
      "failureScopeAccount": "账号 #{id}",
      "failureScopeGeneral": "上游探测",
      "failureCodes": {
        "missing_management_credentials": "缺少管理凭证",
        "missing_management_user_id": "缺少 NewAPI 用户 ID",
        "missing_login": "缺少登录凭证",
        "cloudflare_blocked": "Cloudflare 拦截",
        "public_api_unavailable": "公共接口不可用",
        "request_failed": "请求失败",
        "key_not_found": "未找到 API Key",
        "rate_unavailable": "分组倍率缺失",
        "error": "探测错误",
        "unavailable": "不可用",
        "unknown": "未知失败"
      },
      "failureMessages": {
        "missingManagement": "缺少管理凭证，或管理接口当前无法访问。",
        "missingManagementUserId": "已配置 NewAPI 管理 Token，但缺少有效的 NewAPI 用户 ID。",
        "missingLogin": "绑定账号没有完整配置上游面板登录凭证。",
        "cloudflareBlocked": "上游面板被 Cloudflare 拦截，请配置可用代理或让上游放行服务器出口 IP。",
        "keyNotFound": "已连接上游，但没有找到这个 API Key。",
        "rateUnavailable": "API Key 已验证，但上游没有返回所属分组倍率。"
      },
      "columns": {
        "upstream": "上游",
        "kind": "站点类型",
        "wallet": "真实钱包",
        "localGroups": "本站分组",
        "accounts": "绑定账号",
        "actions": "操作"
      },
      "kind": {
        "auto": "自动识别"
      },
      "status": {
        "unknown": "未探测",
        "healthy": "正常",
        "degraded": "部分可用",
        "error": "异常"
      },
      "tabs": {
        "capabilities": "生成新账号",
        "binding": "绑定现有账号",
        "accounts": "已绑定账号"
      },
      "form": {
        "name": "上游名称",
        "namePlaceholder": "例如：主力 NewAPI",
        "kind": "站点类型",
        "baseUrl": "Base URL",
        "proxy": "访问代理",
        "direct": "直连",
        "credentials": "调用与管理凭证",
        "blankPreserves": "编辑时留空表示保持原值",
        "apiKey": "默认 API Key",
        "managementToken": "NewAPI 管理 Access Token",
        "managementUserId": "NewAPI 用户 ID",
        "username": "面板用户名",
        "password": "面板密码",
        "protocolKeys": "按协议覆盖 API Key",
        "openaiKey": "OpenAI API Key",
        "anthropicKey": "Anthropic API Key",
        "geminiKey": "Gemini API Key",
        "grokKey": "Grok API Key",
        "configuredPlaceholder": "已配置，留空保持不变",
        "emptyPlaceholder": "未配置",
        "clearCredential": "清除已保存凭证"
      },
      "detail": {
        "status": "状态",
        "wallet": "真实钱包",
        "groups": "上游分组",
        "accounts": "绑定账号",
        "duplicateBaseUrl": "另有 {count} 个上游使用相同 Base URL。它们仍保持独立，历史账号不会自动归并。",
        "managementMissing": "管理元数据尚未获取，请检查站点类型和管理凭证。",
        "generateTitle": "从上游能力生成账号",
        "generateDescription": "选择协议、上游分组、模型和本站分组，确认预览后创建独立运行账号并自动绑定到当前上游。",
        "protocol": "协议平台",
        "upstreamGroup": "上游分组",
        "upstreamRate": "上游倍率",
        "applyGroupChange": "应用上游分组",
        "groupChangeDone": "已切换到上游分组「{group}」",
        "groupChangeSnapshotWarning": "上游分组已切换，倍率快照将在下次刷新时补齐",
        "groupCatalogueStale": "上游分组目录已过期，请先立即探测",
        "groupDataStale": "当前账号的上游分组状态已过期，切换时会重新回读确认",
        "groupChangeReasons": {
          "contextUnavailable": "缺少上游管理上下文",
          "apiKeyOnly": "只有 API Key 账号支持切换上游分组",
          "missingApiKey": "账号没有可管理的 API Key",
          "missingManagementUserId": "缺少有效的 NewAPI 用户 ID",
          "missingNewAPICredentials": "缺少 NewAPI 管理凭证",
          "missingSub2APICredentials": "缺少 Sub2API 面板凭证",
          "unknownKind": "请先探测并确认上游站点类型",
          "unavailable": "当前账号暂不支持切换上游分组"
        },
        "selectGroup": "选择上游分组",
        "models": "模型白名单",
        "testModel": "发送实际请求验证",
        "testSelectedModels": "验证全部候选",
        "testingModels": "实际请求中...",
        "loadingModels": "正在读取所选分组的模型目录...",
        "modelsLoadFailed": "无法读取所选分组的模型目录",
        "modelNotTested": "尚未发送实际模型请求",
        "testSuccess": "{model} 实际请求成功，耗时 {latency} ms",
        "testFailed": "{model} 暂未通过实际请求：{message}",
        "batchTestSuccess": "{count} 个候选模型实际请求均成功，耗时 {latency} ms",
        "batchTestFailed": "{total} 个候选模型中有 {failed} 个未通过：{message}",
        "batchTestRequestFailed": "批量实际请求失败：{message}",
        "selectGroupForModels": "选择上游分组后读取待验证候选模型",
        "noModels": "所选分组没有返回候选模型",
        "localGroups": "绑定到本站分组",
        "noLocalGroups": "没有可绑定的本站分组",
        "accountName": "账号名称",
        "accountNamePlaceholder": "留空则按上游、分组和协议自动命名",
        "concurrency": "并发数",
        "priority": "优先级",
        "keyOverride": "本次账号专用 API Key",
        "keyOverridePlaceholder": "可选；留空使用上游凭证或自动创建分组 Key",
        "addToPlan": "加入生成计划",
        "plan": "待生成账号",
        "modelsShort": "个模型",
        "localGroupsShort": "个本站分组",
        "preview": "生成预览",
        "previewing": "预览中...",
        "previewTitle": "变更预览",
        "previewValid": "可以执行",
        "previewInvalid": "需要修正",
        "previewSkip": "已存在，跳过",
        "previewCreateKey": "创建上游分组 Key 后新增",
        "previewCreate": "新增账号",
        "confirmGenerate": "确认生成账号",
        "generating": "生成中...",
        "generateDone": "账号已生成并自动绑定",
        "generatePartial": "生成完成，但有 {count} 项失败",
        "bindTitle": "手动绑定历史账号",
        "bindOnlyApiKeys": "这里只列出尚未绑定上游的 API Key 账号；不会按 Base URL 自动匹配。",
        "bindPreservesAccount": "绑定只写入上游关联，不修改账号凭证、模型、分组、状态或调度设置。",
        "bindSelected": "绑定所选 {count} 个账号",
        "binding": "绑定中...",
        "searchAccounts": "搜索未绑定账号",
        "noCandidates": "没有可手动绑定的未关联 API Key 账号",
        "bindDone": "所选账号已绑定，原有账号配置未改变",
        "boundAccounts": "当前上游的运行账号",
        "manageBinding": "管理绑定",
        "account": "账号",
        "accountStatus": "账号状态",
        "accountOrigin": "来源",
        "generated": "上游管理生成",
        "manual": "手动绑定",
        "unbind": "解除绑定",
        "deleteOnUnbind": "解绑时同时删除账号",
        "unbindDeleteConfirm": "确认解除绑定并删除这个运行账号？账号的调度配置会一并清理，此操作不可撤销。",
        "unbindPreserveConfirm": "确认只解除绑定并保留这个运行账号？账号将继续存在，但不再由当前上游统一管理。",
        "unbindDeleted": "已解除绑定并删除账号",
        "unbindPreserved": "已解除绑定，账号已保留",
        "noBoundAccounts": "当前上游还没有绑定运行账号"
      }
    },
    "groups": {
      "accountFilters": {
        "title": "账号准入限制",
        "oauthOnly": "仅 OAuth 账号",
        "oauthOnlyEnabled": "只允许 OAuth 类型账号参与该分组",
        "privacySetOnly": "仅隐私已设置账号",
        "privacySetOnlyEnabled": "只允许已完成 Privacy 设置的账号参与该分组"
      },
      "modelsList": {
        "hint": "候选来自当前分组已绑定账号支持的模型；用户侧 /v1/models 和可用渠道页只展示这里勾选的模型。",
        "selectedSummary": "已选择 {selected} / {total} 个模型"
      },
      "openaiMessages": {
        "opusModelPlaceholder": "例如: gpt-5.5",
        "sonnetModelPlaceholder": "例如: gpt-5.5",
        "targetModelPlaceholder": "例如: gpt-5.5",
        "forcePriority": "强制启用 priority",
        "forcePriorityHint": "启用后，此 OpenAI 分组的 /v1/responses 请求都会以 priority 发往上游，本地计费仍按普通倍率计算。"
      },
      "rateNotice": {
        "title": "调价邮件通知",
        "subtitle": "统计最近使用该分组的用户，确认后发送提前通知",
        "windowMinutes": "统计窗口（分钟）",
        "effectiveAt": "预计生效时间",
        "messagePlaceholder": "管理员备注（可选）",
        "preview": "预览用户",
        "previewing": "预览中...",
        "send": "发送通知",
        "sending": "发送中...",
        "rateRequired": "请先把费率倍数改成新的值",
        "previewSummary": "可通知 {count} 人，跳过 {skipped} 个无效邮箱",
        "resultSummary": "已发送 {sent} 封，失败 {failed} 封",
        "sentSummary": "调价通知发送完成：成功 {sent}，失败 {failed}",
        "failedToPreview": "预览调价通知失败",
        "failedToSend": "发送调价通知失败"
      },
      "claudeMaxSimulation": {
        "title": "Claude Max 用量模拟",
        "tooltip": "启用后，未返回上游缓存写入用量的 Claude 模型会按确定性规则映射为少量输入加 1h 缓存创建，总 Token 数保持不变。",
        "enabled": "已启用（模拟 1h 缓存）",
        "disabled": "已禁用",
        "hint": "仅调整用量计费日志中的 Token 分类，不保存单请求映射状态。"
      }
    },
    "gateway": {
      "dashboardKicker": "Admin Console",
      "usersKicker": "User Directory",
      "accountsKicker": "Account Pools",
      "schedulerKicker": "Scheduler Control",
      "paymentDashboardKicker": "Payment Overview",
      "paymentPlansKicker": "Payment Plans",
      "settingsKicker": "Settings Hub",
      "ticketsKicker": "Support Queue",
      "subscriptionsKicker": "Subscription Access",
      "groupsKicker": "Routing Groups",
      "channelsKicker": "Pricing Channels",
      "channelMonitorKicker": "Channel Health",
      "usageKicker": "Usage Intelligence",
      "riskControlKicker": "Policy Guard",
      "proxiesKicker": "Proxy Network",
      "announcementsKicker": "Broadcast Center",
      "promoCodesKicker": "Growth Codes",
      "backupKicker": "Recovery Vault",
      "opsKicker": "Operations Radar",
      "affiliatesKicker": "Affiliate Ledger",
      "opsScope": "Operations Scope",
      "accessScope": "Access Scope",
      "capacityScope": "Capacity Scope",
      "schedulerScope": "Routing Scope",
      "paymentScope": "Payment Scope",
      "catalogScope": "Catalog Scope",
      "settingsScope": "Configuration Scope",
      "supportScope": "Support Scope",
      "routingScope": "Routing Scope",
      "pricingScope": "Pricing Scope",
      "monitorScope": "Monitor Scope",
      "observabilityScope": "Observability Scope",
      "riskScope": "Risk Scope",
      "networkScope": "Network Scope",
      "communicationScope": "Communication Scope",
      "growthScope": "Growth Scope",
      "recoveryScope": "Recovery Scope",
      "affiliateScope": "Affiliate Scope",
      "dashboardPanelTitle": "系统运行总览",
      "dashboardPanelDescription": "请求、Token、账号池和用户增长集中在这里，图表仍按下方时间范围刷新。",
      "usersPanelTitle": "权限与余额统一管理",
      "usersPanelDescription": "筛选、列设置、余额、授权分组和用户属性保持在同一张高密度表格中。",
      "accountsPanelTitle": "上游账号池控制面",
      "accountsPanelDescription": "自动刷新、批量验活、导入导出和调度状态保留在账号列表工具栏中。",
      "schedulerPanelTitle": "分组调度与账号健康",
      "schedulerPanelDescription": "拖拽顺序、自动排序、上游余额、倍率、监控开关和账号级调度配置保留在同一个紧凑控制面。",
      "paymentDashboardDescription": "在线支付、卡密兑换、管理员充值退款和净收入口径集中在支付概览里观察。",
      "paymentDashboardPanelTitle": "收入与订单事实源",
      "paymentDashboardPanelDescription": "今日收入、总收入、支付方式分布和高消费用户继续使用后台统计接口，不在前端重新计算账务。",
      "paymentPlansDescription": "维护面向用户展示的充值/订阅套餐，保留分组绑定、有效期、排序和上下架控制。",
      "paymentPlansPanelTitle": "套餐目录控制台",
      "paymentPlansPanelDescription": "套餐只管理展示和购买入口，真实分组能力、倍率与可用模型仍来自分组配置。",
      "settingsDescription": "集中维护站点、认证、功能、网关、支付、邮件和备份配置。",
      "settingsPanelTitle": "全局配置控制台",
      "settingsPanelDescription": "设置项继续按原 tab 分组保存；独立保存的网关、支付服务商和备份流程保持原有行为。",
      "ticketsPanelTitle": "工单队列与售后处理",
      "ticketsPanelDescription": "搜索、状态筛选、优先级、分配、批量更新、自动关闭和详情回复仍在原表格流程中处理。",
      "subscriptionsPanelTitle": "订阅授权控制台",
      "subscriptionsPanelDescription": "订阅分配、延期、撤销、额度重置、分组筛选和列设置保持原有高密度操作方式。",
      "groupsPanelTitle": "服务分组与路由边界",
      "groupsPanelDescription": "分组创建、平台绑定、倍率、订阅限制、账号复制和排序继续在原表格与弹窗中处理；账号调度统一在调度管理页查看。",
      "channelsPanelTitle": "模型价格与渠道能力",
      "channelsPanelDescription": "渠道状态、分组绑定、模型映射、价格规则和账号统计价格规则保持在原创建/编辑流程中。",
      "channelMonitorPanelTitle": "用户可见渠道健康源",
      "channelMonitorPanelDescription": "监控项排序、模板管理、手动探测、启停和删除保持原流程，用户侧状态页继续读取同一套监控数据。",
      "usagePanelTitle": "请求、Token 与成本审计",
      "usagePanelDescription": "时间范围、模型、分组、账号、计费类型和错误日志保持联动，用于核对扣费、倍率和上游请求表现。",
      "riskControlPanelTitle": "内容审计运行控制面",
      "riskControlPanelDescription": "运行模式、Key 健康、审核范围、日志筛选和阈值设置继续由原配置流程保存。",
      "proxiesPanelTitle": "代理池与账号绑定状态",
      "proxiesPanelDescription": "导入导出、批量检测、质量检查、账号绑定和备用代理配置保留在同一张高密度表格里。",
      "announcementsPanelTitle": "用户公告与定向触达",
      "announcementsPanelDescription": "公告创建、状态、弹窗模式、投放条件、已读情况和删除确认继续在原表格与弹窗中处理。",
      "promoCodesPanelTitle": "注册激励与使用追踪",
      "promoCodesPanelDescription": "优惠码生成、注册链接复制、使用记录、过期限制和删除确认保持原流程。",
      "backupPanelTitle": "备份、存储与恢复控制台",
      "backupPanelDescription": "S3/R2 存储、定时策略、手动备份、下载、恢复和删除仍走原备份接口与轮询逻辑。",
      "opsPanelTitle": "实时运维与排障入口",
      "opsPanelDescription": "时间窗口、平台、分组、告警规则、错误明细、请求明细和全屏模式保持原运维监控流程。",
      "affiliatesPanelTitle": "邀请返佣记录事实源",
      "affiliatesPanelDescription": "邀请关系、充值返佣和返利提取记录保持分表展示，用户概览弹窗继续读取后台接口。",
      "today": "今日",
      "currentFilter": "当前筛选",
      "currentPage": "当前页",
      "selectedRange": "已选时间范围",
      "billingAudit": "成本审计口径",
      "tableDensity": "表格密度",
      "totalUsers": "全部用户",
      "totalAccounts": "全部账号",
      "totalProxies": "全部代理",
      "activeOnPage": "本页启用",
      "boundAccounts": "已绑定账号",
      "averageLatency": "平均延迟",
      "schedulableAccounts": "参与调度",
      "monitoringAccounts": "监控开启",
      "coolingAccounts": "冷却中",
      "filteredRequests": "筛选请求",
      "totalTokens": "Token 总量",
      "actualCost": "实际成本",
      "selectedRowsCount": "已选 {count} 行",
      "latencySamples": "{count} 个延迟样本",
      "proxyPoolUsage": "当前页代理池占用",
      "totalAnnouncements": "全部公告",
      "popupMode": "弹窗模式",
      "targetedRules": "定向规则",
      "segmentedDelivery": "按条件投放",
      "totalPromoCodes": "全部优惠码",
      "promoUses": "使用次数",
      "promoBonus": "赠送金额",
      "exhaustedCodesCount": "{count} 个已过期/用完",
      "noExhaustedCodes": "暂无过期/用完",
      "totalBackups": "备份记录",
      "completedBackups": "已完成备份",
      "runningBackupsCount": "{count} 个执行中",
      "noRunningBackups": "无执行中任务",
      "storageTarget": "存储目标",
      "s3Configured": "S3 凭据已配置",
      "s3NotConfigured": "S3 凭据未配置",
      "backupSchedule": "定时策略",
      "opsRequests": "请求总量",
      "opsSla": "SLA",
      "upstreamErrorRate": "上游错误率",
      "groupScoped": "分组 #{id}",
      "allGroupsScope": "全部分组",
      "affiliateInvitesTitle": "邀请关系",
      "affiliateRebatesTitle": "返佣记录",
      "affiliateTransfersTitle": "返利提取",
      "totalAffiliateRecords": "全部记录",
      "currentPageRows": "本页记录",
      "totalRebate": "累计返利",
      "rebateAmount": "返利金额",
      "transferAmount": "提取金额",
      "paidAmount": "支付金额",
      "dateRange": "日期范围",
      "activePlans": "上架套餐",
      "linkedGroups": "绑定分组",
      "missingGroups": "缺失分组",
      "settingsSections": "设置分区",
      "currentSection": "当前分区",
      "adminApiKey": "管理 API Key",
      "saveState": "保存状态",
      "operationalMonitors": "正常监控",
      "visibleColumns": "可见列",
      "hiddenColumns": "列已隐藏",
      "selectedRows": "已选行",
      "ordersKicker": "Billing Records",
      "invoicesKicker": "Invoice Desk",
      "redeemKicker": "Redeem Codes",
      "revenueScope": "Revenue Scope",
      "invoiceScope": "Invoice Scope",
      "redeemScope": "Redeem Scope",
      "ordersDescription": "统一查看线上订单、卡密兑换、管理员充值退款和返佣来源。",
      "redeemDescription": "生成、筛选、导出和批量维护兑换码，业务分类继续跟随余额流水口径。",
      "ordersPanelTitle": "订单和退款处理台",
      "ordersPanelDescription": "退款、重试、取消和审计日志仍在订单详情中处理。",
      "invoicesPanelTitle": "开票审核与服务费确认",
      "invoicesPanelDescription": "服务费冻结、确认扣除、驳回释放和发票号记录保持原处理流程。",
      "redeemPanelTitle": "卡密和兑换码控制台",
      "redeemPanelDescription": "支持余额、订阅、并发和邀请码类型；批量修改状态、过期时间、备注和分组。",
      "totalOrders": "全部订单",
      "totalInvoices": "全部发票",
      "totalRedeemCodes": "全部兑换码"
    },
    "channels": {
      "form": {
        "cacheWritePriceShort": "写入",
        "cacheReadPriceShort": "读取",
        "minTokens": "最小 Token",
        "maxTokens": "最大 Token",
        "inclusive": "包含边界"
      },
      "noGroupsSelected": "请至少选择一个分组",
      "emptyModelsInPricing": "请至少配置一个模型定价"
    },
    "riskControl": {
      "defaultBlockMessage": "请求内容未通过安全审核，请调整后重试。"
    },
    "channelMonitor": {
      "columns": {
        "sortOrder": "排序"
      },
      "form": {
        "apiModeResponsesHint": "使用 /v1/responses，默认带 instructions + input；适合平台自检/Codex。",
        "primaryModelPlaceholder": "gpt-5.4-mini",
        "sortOrder": "排序",
        "sortOrderHint": "越小越靠前",
        "linkedKeyHint": "已绑定 API Key #{id}，监控会读取当前 Key，不依赖手动密钥快照。",
        "manualKeyHint": "建议使用「我的 Key」绑定；手动 Key 会加密保存，若未配置固定 TOTP_ENCRYPTION_KEY，重启后可能需要重填。"
      },
      "sortOrder": "排序",
      "sortOrderHint": "拖拽监控调整显示顺序，排在前面的监控会优先显示",
      "sortOrderUpdated": "排序已更新",
      "failedToLoadSortOrder": "加载排序列表失败",
      "failedToUpdateSortOrder": "更新排序失败"
    },
    "subscriptions": {
      "resetWithCost": "重置（扣 1 天）",
      "resetWithCostTitle": "确认提前重置",
      "resetWithCostConfirm": "此操作将立即刷新当日额度，代价是扣除 {cost} 订阅时间。\n用户：{user}\n当前剩余：{before}\n重置后剩余：{after}\n确定继续？",
      "resetWithCostSuccess": "重置成功，已扣除 1 天，当前剩余 {days} 天",
      "resetWithCostFailed": "重置失败",
      "resetWithCostError": {
        "timeInsufficient": "剩余时间不足 1 天，无法重置",
        "inactive": "订阅已过期或暂停，无法重置",
        "notFound": "订阅不存在"
      },
      "durationLessThanMinute": "少于 1 分钟",
      "durationDays": "{n} 天",
      "durationHours": "{n} 小时",
      "durationMinutes": "{n} 分",
      "remainingPrefix": "剩余"
    },
    "accounts": {
      "status": {
        "rate_limited": "限流中",
        "temp_unschedulable": "临时不可调度",
        "quota_exceeded": "配额超限",
        "insufficientBalance": "余额不足",
        "schedulingPriority": "调度优先级：{priority}",
        "schedulingEnabled": "调度：启用",
        "schedulingDisabled": "调度：停用",
        "nextAvailableAt": "下次可用：{time}",
        "lastUsedAt": "最近使用：{time}",
        "schedulingReason": "原因：{reason}",
        "lastError": "最近错误：{error}"
      },
      "usageWindow": {
        "grokUnknown": "未知",
        "grokRetryAfter": "重试等待",
        "grokProbeTooltip": "主动探测 Grok 用量窗口",
        "grokResetUnsupportedTooltip": "当前 Grok 上游未返回可用的重置窗口",
        "grokNoHeaders": "未返回用量头",
        "grokLastStatus": "最近状态",
        "grokLastProbe": "最近探测",
        "grokLastHeadersSeen": "最近响应头"
      },
      "ineligibleWarning": "该账号无 Antigravity 使用权限，但 API 转发仍可用，账号测试结果已标记该状态。",
      "bulkActions": {
        "testModels": "批量测试模型"
      },
      "openai": {
        "cacheAffinityGroup": "缓存兼容组",
        "cacheAffinityGroupPlaceholder": "例如 codex-prompt-a",
        "cacheAffinityGroupDesc": "同一值表示这些上游账号的内置提示词一致。Codex 长上下文切号时会优先在同组内切换，留空则默认按账号隔离。"
      },
      "grok": {
        "baseUrlHint": "留空使用默认 Grok API 地址",
        "apiKeyHint": "填写 Grok / xAI API Key"
      },
      "oauth": {
        "openai": {
          "codexPatAuth": "Codex PAT",
          "codexPatDesc": "导入 Codex Personal Access Token，系统会验证后创建账号。",
          "codexPatPlaceholder": "粘贴 Codex PAT...",
          "codexPatHint": "支持从 Codex 获取的个人访问令牌。",
          "codexPatImportAndCreate": "导入并创建账号",
          "codexPatEmpty": "请输入 Codex PAT",
          "codexPatImportFailed": "导入 Codex PAT 失败",
          "mobileRefreshTokenAuth": "移动端 Refresh Token",
          "accessTokenAuth": "Access Token"
        },
        "grok": {
          "title": "Grok 账户授权",
          "followSteps": "请按照以下步骤完成 Grok 账户的授权：",
          "step1GenerateUrl": "点击下方按钮生成授权链接",
          "openUrlDesc": "请在新标签页中打开授权链接，登录您的 xAI / Grok 账户并授权。",
          "importantNotice": "重要提示：授权后页面可能会加载较长时间，请耐心等待。当浏览器地址栏变为 http://localhost... 开头时，表示授权已完成。",
          "authCodeDesc": "授权完成后，当页面地址变为 http://localhost:xxx/auth/callback?code=... 时：",
          "authCodePlaceholder": "方式1：复制完整的链接\n(http://localhost:xxx/auth/callback?code=...)\n方式2：仅复制 code 参数的值",
          "authCodeHint": "您可以直接复制整个链接或仅复制 code 参数值，系统会自动识别",
          "refreshTokenDesc": "输入您已有的 Grok Refresh Token，支持批量输入（每行一个），系统将自动验证并创建账号。",
          "refreshTokenPlaceholder": "粘贴您的 Grok Refresh Token...\n支持多个，每行一个",
          "missingExchangeParams": "缺少 code / session_id / state",
          "failedToValidateRT": "验证 Refresh Token 失败",
          "oauthOnlyHint": "Grok 账号目前仅支持 OAuth / Refresh Token 授权导入。"
        }
      },
      "gemini": {
        "oauthType": {
          "googleOneDesc": "适合个人 Google One / Gemini 账号，无需 GCP 项目。",
          "codeAssistDesc": "适合 Gemini CLI / Code Assist，需要可用 GCP 项目。",
          "codeAssistRequirement": "请准备 Project ID，并确保账号已启用相关服务。",
          "showAdvanced": "显示高级选项",
          "hideAdvanced": "收起高级选项",
          "badges": {
            "individuals": "个人账号",
            "enterprise": "企业/项目"
          }
        },
        "setupGuide": {
          "links": {
            "countryChange": "修改账号地区"
          }
        }
      },
      "bulkVerify": {
        "button": "批量验活",
        "title": "批量账号验活（wham/usage）",
        "description": "通过 ChatGPT 后端 wham/usage 接口批量检查 OAuth 账号健康度与限流窗口。不消耗任何额度，结果写回账号 Extra（与常规 Codex 用量字段一致）。",
        "concurrency": "并发",
        "concurrencyHint": "建议 16-64，上限 128。并发过高可能被 ChatGPT 限流。",
        "scopeHint": "默认扫描全部 OpenAI OAuth 账号。",
        "multiInstanceWarning": "⚠ 任务状态仅存在于启动它的实例上。多实例部署时，轮询需打到同一实例（否则会得到 404）。",
        "alreadyRunning": "已有验活任务在运行中，已切换到该任务的进度视图。",
        "applyTitle": "结果处理",
        "applyDescription": "按照网关正常调用失败的方式处理这些账号：刷新失效令牌 → 若刷新失败则标记 error；已限流的按 7d 重置时间写入 rate_limited_at。",
        "applyRefresh": "刷新失效令牌（{n} 个账号，刷新失败会被标记为 error）",
        "applyMarkExhausted": "标记限流账号（{n} 个账号，写入 7d 重置时间，调度器自动避开）",
        "applyDryRun": "仅试跑（不实际写库）",
        "applyBtn": "执行处理",
        "applyDryRunBtn": "试跑",
        "applyResult": "处理结果",
        "applyResultDryRun": "试跑结果（未写库）",
        "applyRefreshed": "刷新成功",
        "applyRefreshFailed": "刷新失败",
        "applyMarked": "已标记限流",
        "applyMarkFailed": "标记失败",
        "applySkippedNoReset": "跳过（无 reset 时间）",
        "applyFailuresTitle": "失败详情（{n} 条）",
        "start": "开始验活",
        "cancel": "取消任务",
        "reset": "再跑一次",
        "statusRunning": "进行中",
        "statusFinished": "已完成",
        "statusCancelled": "已取消",
        "startedAt": "开始时间",
        "finishedAt": "完成时间",
        "high": "健康 (<30%)",
        "medium": "一般 (30-70%)",
        "low": "吃紧 (70-100%)",
        "exhausted": "已限流",
        "expired": "令牌失效",
        "missing": "缺字段",
        "unknown": "无窗口数据",
        "error": "出错"
      },
      "fromModel": "请求模型",
      "toModel": "实际模型",
      "unnamed": "未命名",
      "messages": {
        "accountCreated": "账号创建成功"
      },
      "copyAccount": "复制账号",
      "copyAccountSuccess": "已复制账号「{name}」",
      "copyAccountFailed": "复制账号失败",
      "passwordCopied": "密码已复制",
      "rateScale": "上游倍率换算",
      "rateScaleHint": "仅影响上游倍率展示与排序，默认 1；例如上游 1 元 = 10 刀可填 0.1",
      "manualRate": "手动展示倍率",
      "manualRateHint": "显式覆盖调度页展示与排序倍率；留空则按账号倍率或上游倍率计算",
      "manualRatePlaceholder": "例如 0.15",
      "upstreamSub2API": {
        "title": "上游面板",
        "loginTitle": "上游面板登录信息",
        "loginHint": "New API 会先通过 API Key 接口校验可用性；已配置的面板登录会补充钱包、分组和当前倍率。公开接口不提供这些元数据。",
        "panelType": "上游类型",
        "probeSource": "探测来源",
        "probeSources": {
          "apiKey": "API Key 接口",
          "panelLogin": "面板登录核验"
        },
        "panelTypes": {
          "auto": "自动识别"
        },
        "email": "上游登录账号",
        "emailPlaceholder": "邮箱或用户名",
        "password": "上游登录密码",
        "passwordPlaceholder": "留空保持当前密码",
        "passwordCreatePlaceholder": "上游账号密码",
        "statusOk": "当前倍率 {rate}",
        "statusKeyUnlimited": "Key 额度无限",
        "statusKeyRemaining": "Key 剩余 {balance}",
        "statusMetadataUnavailable": "分组倍率未获取",
        "statusError": "上游信息异常",
        "statusCloudflareBlocked": "Cloudflare 拦截",
        "status": "状态",
        "proxyUsed": "已使用账号代理",
        "proxyNotUsed": "未使用账号代理",
        "key": "API Key",
        "group": "分组",
        "platform": "平台",
        "defaultRate": "默认倍率",
        "effectiveRate": "当前倍率",
        "balance": "用户余额",
        "remaining": "Key 剩余额度",
        "metadata": "钱包/分组倍率",
        "message": "错误",
        "fetchedAt": "获取时间",
        "cached": "缓存"
      },
      "bulkModelTest": {
        "title": "批量测试模型",
        "selectedAccounts": "已选择 {count} 个账号",
        "reloadModels": "重新加载模型",
        "modelChecklist": "从账号模型清单选择",
        "modelCount": "{count} 个模型",
        "searchModels": "搜索模型 ID",
        "selectVisible": "选择当前可见",
        "clearModels": "清空",
        "loadingModels": "正在加载所选账号模型...",
        "noModels": "没有加载到模型，可在右侧手动输入模型 ID",
        "manualModels": "手动输入模型",
        "manualModelsPlaceholder": "每行一个模型 ID，例如：\ngpt-5.5\nclaude-sonnet-4-6",
        "manualModelsHint": "会和左侧勾选的模型合并去重，支持空格、逗号或换行分隔。",
        "prompt": "测试提示词",
        "promptPlaceholder": "留空使用默认 hi；生图模型可填写生图提示词。",
        "promptHint": "普通文本模型留空即可；选择生图模型时这里会作为生图提示词。",
        "mode": "测试模式",
        "modeDefault": "普通测试",
        "modeCompact": "Compact 测试",
        "concurrency": "并发数",
        "resultSummary": "结果：总计 {total}，成功 {success}，失败 {failed}",
        "taskPreview": "{accounts} 个账号 × {models} 个模型 = {total} 个任务",
        "accountColumn": "账号",
        "modelColumn": "模型",
        "statusColumn": "状态",
        "latencyColumn": "耗时",
        "messageColumn": "返回信息",
        "success": "成功",
        "failed": "失败",
        "testing": "测试中...",
        "start": "开始批量测试",
        "partialDone": "批量测试完成：成功 {success}，失败 {failed}",
        "allDone": "批量测试完成：{count} 个任务全部成功",
        "failedToRun": "批量测试失败",
        "noSelectedAccounts": "请先勾选要测试的账号"
      }
    },
    "redeem": {
      "columns": {
        "businessCategory": "业务分类"
      },
      "batchFields": {
        "businessCategory": "业务分类"
      },
      "amount": "账户余额金额",
      "form": {
        "balanceHint": "余额兑换码按账户余额一比一到账，例如 10 元卡填写 10"
      }
    },
    "usage": {
      "inputCost": "输入成本",
      "outputCost": "输出成本",
      "cacheCreationCost": "缓存创建成本",
      "cacheReadCost": "缓存读取成本",
      "billingModeVideo": "按秒(视频)",
      "tokenRanking": {
        "subtitle": "按当前筛选与时间范围统计每个用户的 Token 用量，点击行可下钻到该用户",
        "title": "用户 Token 排行",
        "searchPlaceholder": "搜索用户邮箱..."
      }
    },
    "tickets": {
      "title": "工单管理",
      "description": "查看用户工单并跟进处理",
      "searchPlaceholder": "搜索工单、标题或用户...",
      "viewDetail": "查看",
      "empty": "暂无工单",
      "emptyDescription": "当前没有需要处理的用户工单",
      "detailTitle": "工单详情",
      "unassigned": "未指派",
      "internalNote": "内部备注",
      "unreadCount": "{count} 条未读",
      "noUnread": "无未读",
      "stats": {
        "total": "总数",
        "unassigned": "未指派",
        "assignedToMe": "我的工单",
        "handledByMe": "我已处理",
        "escalated": "已升级",
        "slaOverdue": "SLA 超时"
      },
      "queue": {
        "all": "全部队列",
        "mine": "我处理的",
        "support": "客服队列",
        "allNormal": "全部普通工单",
        "superAdmin": "超级管理员队列"
      },
      "escalated": "已升级",
      "claim": "接单",
      "claimed": "已接单",
      "escalate": "转给超级管理员",
      "escalateReasonPrompt": "请输入转交原因（可选）",
      "escalatedDone": "已转给超级管理员",
      "viewImage": "查看图片",
      "balance": {
        "operation": "余额操作",
        "amount": "金额",
        "notes": "备注",
        "notesPlaceholder": "例如：充值未到账补款",
        "add": "加余额",
        "subtract": "扣余额",
        "set": "设为指定余额",
        "apply": "执行",
        "done": "余额已调整",
        "failed": "余额调整失败"
      },
      "bulk": {
        "selected": "已选择 {count} 个",
        "apply": "批量应用",
        "updated": "已更新 {count} 个工单",
        "keepStatus": "保持状态",
        "keepPriority": "保持优先级",
        "keepCategory": "保持分类"
      },
      "autoClose": {
        "button": "自动关闭",
        "title": "自动关闭已解决工单",
        "description": "将已解决时间早于指定天数的工单自动关闭。",
        "days": "已解决至少多少天",
        "confirm": "关闭工单",
        "done": "已关闭 {count} 个工单"
      },
      "saveChanges": "保存变更",
      "updated": "工单已更新",
      "replied": "回复已发送",
      "reply": "回复",
      "internalReply": "仅管理员可见",
      "replyPlaceholder": "输入回复内容，勾选后将作为内部备注保存...",
      "sendReply": "发送回复",
      "replyAttachments": {
        "title": "附件",
        "add": "添加附件",
        "namePlaceholder": "名称",
        "urlPlaceholder": "https://... 或选择图片",
        "chooseImage": "选择图片",
        "imageSelected": "已选择图片"
      },
      "failedToLoad": "加载工单失败",
      "failedToLoadDetail": "加载工单详情失败",
      "failedToUpdate": "更新工单失败",
      "failedToReply": "发送回复失败",
      "columns": {
        "subject": "主题",
        "user": "用户",
        "status": "状态",
        "priority": "优先级",
        "category": "分类",
        "assignee": "指派",
        "lastMessageAt": "最后消息",
        "unread": "未读"
      },
      "form": {
        "status": "状态",
        "priority": "优先级",
        "category": "分类",
        "assigneeId": "指派管理员 ID",
        "assigneePlaceholder": "0",
        "assigneeHint": "填 0 表示取消指派。"
      },
      "filters": {
        "allStatus": "全部状态",
        "allPriority": "全部优先级",
        "allCategory": "全部分类",
        "allAssignees": "全部指派",
        "allUnread": "全部",
        "onlyUnread": "仅未读",
        "unassigned": "未指派",
        "assignedToMe": "指派给我"
      },
      "status": {
        "open": "待处理",
        "pending": "已回复",
        "resolved": "已解决",
        "closed": "已关闭"
      },
      "priority": {
        "low": "低",
        "normal": "普通",
        "high": "高",
        "urgent": "紧急"
      },
      "category": {
        "general": "通用",
        "billing": "计费",
        "usage": "用量",
        "technical": "技术",
        "account": "账户"
      },
      "sender": {
        "user": "用户",
        "admin": "管理员",
        "system": "系统"
      },
      "errors": {
        "TICKET_INPUT_REQUIRED": "工单参数不能为空",
        "TICKET_SUBJECT_INVALID": "工单标题不能为空，且不能超过 200 个字符",
        "TICKET_BODY_REQUIRED": "消息内容不能为空",
        "TICKET_STATUS_INVALID": "工单状态不正确",
        "TICKET_PRIORITY_INVALID": "优先级不正确",
        "TICKET_CATEGORY_INVALID": "工单分类不正确",
        "TICKET_PERMISSION_DENIED": "没有权限处理该工单",
        "TICKET_ASSIGNEE_INVALID": "只能指派给启用状态的管理员或客服",
        "TICKET_IDS_REQUIRED": "请选择至少一个工单",
        "TICKET_ATTACHMENT_INVALID": "附件名称和链接必须是有效 http(s) 地址，最多 5 个",
        "TICKET_CLOSED": "工单已关闭",
        "TICKET_NOT_FOUND": "工单不存在"
      }
    },
    "ops": {
      "autoRefreshRemaining": "{seconds}s 后刷新",
      "runtime": {
        "metricThresholds": "指标阈值",
        "metricThresholdsHint": "用于首页健康摘要和告警建议，不会直接拦截请求。",
        "slaMinPercent": "SLA 最低值 (%)",
        "slaMinPercentHint": "低于该值时标记为需要关注。",
        "ttftP99MaxMs": "TTFT P99 最大值 (ms)",
        "ttftP99MaxMsHint": "首字 P99 超过该值时标记为慢。",
        "requestErrorRateMaxPercent": "请求错误率最大值 (%)",
        "requestErrorRateMaxPercentHint": "客户端/平台错误率超过该值时提示异常。",
        "upstreamErrorRateMaxPercent": "上游错误率最大值 (%)",
        "upstreamErrorRateMaxPercentHint": "上游错误率超过该值时提示上游不稳定。"
      },
      "openAIScheduler": {
        "title": "OpenAI 调度状态",
        "model": "模型",
        "endpoint": "接口",
        "accounts": "账号",
        "available": "可用",
        "blocked": "跳过",
        "circuit": "熔断",
        "halfOpen": "探测",
        "full": "满载",
        "switchRate": "切号率",
        "stickyRate": "粘连率",
        "latency": "调度耗时",
        "disabled": "OpenAI 调度状态不可用",
        "empty": "暂无 OpenAI 调度账号",
        "loadFailed": "加载 OpenAI 调度状态失败",
        "reason": {
          "model_unsupported": "模型不支持",
          "endpoint_unsupported": "接口不支持",
          "compact_unsupported": "compact 不支持",
          "inactive": "未启用",
          "manual_unschedulable": "手动停用",
          "expired": "已过期",
          "overloaded": "过载",
          "rate_limited": "限流",
          "temp_unschedulable": "临时摘除",
          "quota_exceeded": "额度用尽",
          "model_rate_limited": "模型限流",
          "privacy_not_set": "隐私未设置",
          "runtime_circuit_open": "运行时熔断",
          "runtime_half_open_in_flight": "运行时探测中",
          "scheduler_circuit_open": "调度熔断",
          "scheduler_half_open": "调度探测",
          "scheduler_half_open_in_flight": "调度探测中",
          "partial_model_circuit": "部分模型冷却",
          "partial_model_probe": "部分模型探测中",
          "transient_transport": "连接临时异常",
          "transient_timeout": "请求超时",
          "openai_transient_5xx": "上游 5xx",
          "openai_transport_error": "连接临时异常",
          "openai_timeout": "请求超时",
          "openai_stream_error": "流式中断",
          "openai_request_error": "请求连接错误",
          "openai_temp_unschedulable": "临时摘除",
          "network_or_stream_interruption": "网络/流中断",
          "concurrency_full": "并发满",
          "unavailable": "不可用"
        }
      }
    },
    "scheduler": {
      "title": "调度管理",
      "description": "按分组查看账号调度顺序、上游状态与健康探测",
      "platform": "平台",
      "group": "分组",
      "selectGroup": "选择分组",
      "autoSort": "自动排序",
      "autoSortHint": "后端会对所有分组统一按稳定性和模型成功率持续排序，再用延迟和负载作为同层决胜条件",
      "autoSortPolicy": "稳定性 + 模型成功率",
      "refreshOrder": "刷新排序",
      "refreshOrderHint": "重新加载后端自动排序任务已保存的顺序",
      "refreshUpstream": "刷新上游",
      "saveOrder": "保存调度",
      "orderHint": "拖拽调整顺序后，点击“保存调度”写回账号优先级和分组调度配置。",
      "pickGroupFirst": "请先选择一个分组",
      "loading": "加载中...",
      "empty": "当前分组暂无可调度账号",
      "noFilteredAccounts": "当前筛选条件下没有账号",
      "currentConcurrency": "当前并发",
      "concurrency": "当前并发",
      "rateMultiplier": "最终倍率",
      "cachedRateMultiplier": "上次核验倍率",
      "recentFirstToken5m": "首字",
      "recentFirstToken5mTitle": "最近 5 分钟用户请求平均首字，样本 {count}",
      "recentFirstTokenNoSamples": "最近 5 分钟暂无用户首字样本",
      "recentFirstTokenGroupText": "近 5 分钟平均首字 {value} · {count}",
      "recentFirstTokenGroupTitle": "当前分组最近 5 分钟用户请求平均首字 {value}，样本 {count}",
      "recentFirstTokenGroupNoSamples": "近 5 分钟首字 -",
      "balanceWalletKey": "钱包余额 / Key 剩余",
      "walletBalance": "钱包余额",
      "cachedWalletBalance": "上次核验钱包余额",
      "keyRemaining": "Key 剩余额度",
      "keyQuotaUnlimited": "Key 额度无限",
      "insufficientBalance": "余额不足",
      "insufficientBalanceTitle": "上游账号余额不足，当前已停止参与调度",
      "insufficientBalanceSummary": "余额不足 {count}",
      "insufficientBalanceSummaryTitle": "当前分组检测到余额不足的上游账号数量",
      "tempUnsched": "临时不可调度",
      "stateLabel": "调度状态",
      "baseStateLabel": "基础调度状态",
      "stateActive": "运行中",
      "stateStopped": "已停止",
      "stateError": "异常",
      "stateExpired": "已过期",
      "stateRateLimited": "限流中",
      "stateOverloaded": "过载中",
      "stateTempUnschedulable": "临时摘除",
      "stateAccountCooldown": "账号冷却",
      "stateModelUnavailable": "模型摘除",
      "stateQuotaExceeded": "配额耗尽",
      "stateUnknown": "未知",
      "model": "模型",
      "remaining": "剩余",
      "modelCooldownTitle": "模型冷却",
      "modelCooldownOne": "{model} 冷却",
      "modelCooldownMany": "{count} 个模型冷却",
      "modelCooldownNone": "无模型冷却",
      "modelCooldownNoActive": "当前没有模型级冷却",
      "modelCooldownSummary": "{count} 个模型正在避让",
      "accountCooldown": "账号冷却",
      "accountCooldownTitle": "账号级冷却，影响该账号所在的所有分组",
      "groupReserve": "分组保底",
      "groupReserveTitle": "当前分组保底",
      "groupReserveActive": "当前分组无正常兼容候选时可作为最终候选",
      "groupReserveNone": "当前不参与分组保底",
      "groupReserveNoneDetail": "仅账号级临时冷却中的兼容账号才会作为保底；不会绕过余额、鉴权、限流、过载或模型限制。",
      "monitorModel": "监控模型",
      "bulkMonitorModel": "批量检测模型",
      "monitorStateLabel": "探针状态",
      "monitorStateShort": "探针",
      "monitorNoData": "暂无监控数据",
      "monitorOff": "监控已关闭",
      "monitorStatusOperational": "正常",
      "monitorStatusDegraded": "降级",
      "monitorStatusFailed": "失败",
      "monitorStatusError": "异常",
      "monitorSlowResponse": "响应缓慢 {ms}ms",
      "monitorFailedProbe": "探测失败",
      "visibleAccounts": "当前显示 {shown}/{total} 个账号",
      "selectVisible": "选择当前可见",
      "selectedCount": "已选 {count} 个",
      "selectAccount": "选择账号",
      "dragDisabledWhenFiltered": "筛选视图下禁用拖拽",
      "viewDetail": "查看详情",
      "detailTitle": "账号详情",
      "monitorPing": "Ping",
      "monitorAvgLatency1h": "近 1h 平均延迟",
      "recentFailed": "最近失败",
      "totals": "摘要",
      "recentErrorHold": "冷却剩余",
      "timeline": "最近探测",
      "monitorTimeline": "探测时间线",
      "historyTitle": "历史日志",
      "historyHint": "最近 50 条调度和监控事件，已过滤后台同步噪声",
      "historyLoading": "加载历史中...",
      "historyEmpty": "暂无历史日志",
      "historySearch": "搜索事件",
      "historyLoadFailed": "加载历史日志失败",
      "historyAccount": "账号",
      "historyGroup": "分组",
      "historyType": {
        "account_changed": "账号变更",
        "account_groups_changed": "账号分组变更",
        "account_bulk_changed": "批量账号变更",
        "account_last_used": "账号使用更新",
        "scheduling_blocked": "自动摘除",
        "scheduling_block_skipped": "跳过摘除",
        "account_monitor": "账号监控",
        "group_changed": "分组变更",
        "full_rebuild": "全量重建",
        "other": "其他"
      },
      "historyBlockedByMonitor": "账号监控连续失败，已自动摘除调度",
      "historyBlockedByPolicy": "调度策略已自动摘除账号",
      "historyModelCooldown": "模型不支持/不可用，已按模型避让",
      "historyRuntimeCooldown": "运行时避让，短时间内不再调度此账号",
      "historyFailureThreshold": "连续失败 {count} 次",
      "historyCooldownMinutes": "冷却 {minutes} 分钟",
      "historyCooldownHours": "冷却 {hours} 小时",
      "historyCooldownHoursMinutes": "冷却 {hours} 小时 {minutes} 分钟",
      "historyFailureTransient": "临时上游异常",
      "historyFailureInsufficientBalance": "上游余额不足",
      "historyFailureAuthFailed": "上游鉴权失败",
      "historyFailureUpstreamGroupUnavailable": "上游分组不可用",
      "historyFailureModelUnsupported": "上游不支持该模型",
      "historyFailureStickyTTFT": "首字响应过慢",
      "historyFailureStickyErrorRate": "近期错误率过高",
      "historyFailureStickyConcurrencyFull": "并发槽位已满",
      "historyErrorRate": "错误率",
      "historyUntil": "恢复时间",
      "historyRepeated": "已合并重复事件 {count} 次",
      "historyRepeatedShort": "重复 {count} 次",
      "historyFirstAt": "首次",
      "historyLastAt": "最近",
      "historyBlockSkippedLastAccount": "历史调度记录：自动摘除曾因最后候选被跳过",
      "historyBlockSkippedSingleCandidate": "同账号重试记录",
      "historyMonitorMeta": "监控",
      "historyBatch": "批次",
      "bulkClearTempUnsched": "批量清账号冷却",
      "bulkEnableMonitor": "批量开监控",
      "bulkRefreshUpstream": "批量刷上游",
      "bulkClearTempUnschedDone": "已清除 {count} 个冷却",
      "bulkMonitorEnabled": "已开启 {count} 个监控",
      "bulkRefreshDone": "已刷新 {count} 个上游状态",
      "bulkActionFailed": "批量操作失败",
      "schedulable": "是否参与调度",
      "monitor": "账号监控",
      "accountConfig": "调度配置",
      "capacityLimit": "容量上限",
      "loadFactor": "负载因子",
      "useCapacityDefault": "留空使用容量",
      "billingRateMultiplier": "账号计费倍率",
      "rateDisplayConfig": "倍率展示与排序",
      "manualRate": "手动倍率",
      "rateScale": "上游倍率换算",
      "emptyAuto": "留空自动",
      "invalidRateMultiplier": "账号计费倍率必须是大于等于 0 的数字",
      "accountConfigSaved": "调度配置已保存",
      "saveAccountConfigFailed": "保存调度配置失败",
      "clearTempUnsched": "清账号冷却",
      "autoSortOn": "已启用自动排序",
      "autoSortOff": "已关闭自动排序",
      "autoSortSaveFailed": "保存自动排序配置失败",
      "loadGroupsFailed": "加载分组失败",
      "loadFailed": "加载调度数据失败",
      "monitorStopped": "已关闭账号监控",
      "monitorStarted": "已开启账号监控",
      "monitorPlatformUnsupported": "该平台暂不支持账号监控",
      "monitorToggleFailed": "切换账号监控失败",
      "monitorModelUpdated": "监控模型已更新",
      "monitorModelUpdateFailed": "更新监控模型失败",
      "refreshOrderSuccess": "已显示最新的后端调度顺序",
      "tooManyAccounts": "当前分组账号数超过 100，无法按优先级保存",
      "saved": "调度顺序已保存",
      "saveFailed": "保存调度顺序失败",
      "toggleFailed": "切换调度状态失败",
      "tempUnschedCleared": "已清除临时不可调度状态",
      "clearTempUnschedFailed": "清除临时不可调度状态失败",
      "statusUnknown": "未知",
      "statusStopped": "已停止",
      "statusActive": "运行中",
      "statusInactive": "未激活",
      "statusError": "异常",
      "priorityLabel": "优先级",
      "notSchedulable": "未参与调度",
      "monitorLatest": "最新探测",
      "monitorAvailability1h": "近 1 小时可用率"
    },
    "settings": {
      "features": {
        "affiliate": {
          "bindBonusAmount": "绑定礼包领取金额",
          "bindBonusAmountDesc": "用户在注册 1 天内绑定邀请人后，可在邀请返利页手动领取的余额。0 = 不赠送。"
        }
      },
      "site": {
        "apiBaseUrlHint": "用于\"客户端配置\"和\"CC Switch 配置\"功能，留空则使用当前站点地址",
        "docUrlHint": "支持外部 http(s) 链接、站内路径或 md:<slug> 自定义页面。留空则隐藏教程链接。",
        "docUrlPlaceholder": "https://docs.example.com、/custom/guide 或 md:guide",
        "hideCcsImportButton": "隐藏 CC Switch 导入按钮",
        "hideCcsImportButtonHint": "启用后将在 API Keys 页面隐藏\"CC Switch 配置\"按钮"
      },
      "payment": {
        "minAmount": "在线充值最低金额",
        "balanceRechargeMultiplier": "余额官方等价汇率",
        "balanceRechargeMultiplierHint": "用于把账户余额展示为官方人民币等价金额，不影响实际充值到账；充值与卡密面值均按账户余额一比一到账。",
        "balanceRechargePreview": "预览：1 账户余额 ≈ 官方 ¥{cny}",
        "addCustomMethod": "添加自定义方式",
        "minAmountHint": "低于该金额时不能创建在线充值订单，可引导用户使用卡密充值。",
        "balanceRechargeChangeHint": "修改后只影响前端等价展示，不会改动用户余额或历史订单。",
        "balanceRechargeUnlockThreshold": "累计净充值解锁阈值",
        "balanceRechargeUnlockThresholdHint": "禁用余额充值时，累计净充值达到该金额的用户仍可看到原生充值；0 表示不启用解锁。",
        "providerGmpay": "GM Pay / Epusdt",
        "field_token": "代币",
        "field_network": "网络",
        "field_gmpayApiBaseHint": "你的 Epusdt/GM Pay 实例基础地址，例如 https://pay.example.com。",
        "field_gmpayCurrencyHint": "提交给 GM Pay 的法币计价币种，收银台会按配置换算为对应加密货币。",
        "field_gmpayNetworkHint": "GM Pay 使用的链/网络标识，例如 tron、ethereum、bsc、polygon、solana。"
      },
      "openaiFastPolicy": {
        "userIds": "指定用户 ID",
        "userIdsHint": "留空表示对全部 Sub2API 用户生效。指定后仅匹配这些用户的 API Key 请求，且优先于全局规则。",
        "userIdPlaceholder": "例如: 1001",
        "addUserId": "添加用户 ID",
        "removeUserId": "移除用户 ID"
      }
    }
  },
  "purchase": {
    "title": "余额充值",
    "description": "充值入口、支付状态和到账结果一览。",
    "notEnabledDesc": "充值入口暂未开放。",
    "notConfiguredTitle": "充值入口暂未开放",
    "notConfiguredDesc": "充值入口暂时不可用。"
  },
  "customPage": {
    "notConfiguredTitle": "页面暂不可用",
    "notConfiguredDesc": "该页面暂时不可访问。",
    "defaultTitle": "内容",
    "gateway": {
      "kicker": "Developer Docs",
      "description": "平台文档、接入说明和外部工具统一在当前账户上下文中展示。",
      "markdownMode": "Markdown 文档",
      "embedMode": "外部嵌入",
      "tocAvailable": "目录可用",
      "secureContext": "账户上下文",
      "loginScoped": "登录态访问"
    },
    "toc": "目录",
    "openToc": "打开目录",
    "collapseToc": "收起目录",
    "loading": "正在读取内容",
    "unavailableTitle": "内容暂不可用",
    "unavailableDesc": "当前内容暂时无法显示。",
    "markdownNotFoundDesc": "这份内容暂时无法显示。",
    "markdownLoadFailedTitle": "内容读取失败",
    "markdownLoadFailedDesc": "稍后可重新打开。",
    "copyFailed": "复制失败"
  },
  "announcements": {
    "title": "站内信",
    "description": "系统通知、服务更新和专属消息。",
    "gateway": {
      "kicker": "MESSAGE CENTER"
    },
    "all": "全部消息",
    "searchPlaceholder": "搜索标题或内容",
    "messageCount": "共 {count} 条",
    "unreadCount": "{count} 条未读",
    "systemSender": "系统消息",
    "selectMessage": "选择一条消息查看详情",
    "unreadOnly": "未读消息",
    "emptyDescription": "新的系统通知和专属消息会出现在这里。",
    "emptySearch": "没有找到匹配的消息",
    "loadFailed": "站内信加载失败",
    "readStatus": "此消息已读",
    "markReadHint": "打开后将标记为已读",
    "viewAll": "进入站内信",
    "total": "条消息",
    "empty": "暂无站内信",
    "emptyUnread": "暂无未读消息",
    "allMarkedAsRead": "所有消息已标记为已读"
  },
  "userSubscriptions": {
    "description": "当前套餐、用量窗口和到期时间一览，订阅状态随订单同步。",
    "noActiveSubscriptionsDesc": "订阅生效、到期和用量窗口会显示在这里。",
    "noExpiration": "长期有效",
    "unlimitedDesc": "该订阅未设置周期用量上限",
    "trust": {
      "status": "订阅状态",
      "usageWindow": "用量窗口",
      "privacy": "订单记录",
      "resetRecord": "重置时间"
    },
    "reset": "重置",
    "resetTitle": "确认提前重置",
    "resetConfirm": "提前刷新当日额度会扣除 {cost} 订阅时间。\n当前剩余：{before}\n重置后剩余：{after}\n确定继续？",
    "resetSuccess": "额度已重置，当前约剩余 {days} 天",
    "resetFailed": "重置失败",
    "resetError": {
      "timeInsufficient": "剩余时间不足 1 天，无法重置",
      "notOwned": "无权操作此订阅",
      "inactive": "订阅已过期或暂停",
      "notFound": "订阅不存在"
    },
    "autoResetLabel": "自动重置",
    "autoResetHint": "当日额度用完时自动刷新，并扣除相应订阅时长",
    "autoResetEnabled": "已开启自动重置",
    "autoResetDisabled": "已关闭自动重置",
    "autoResetFailed": "切换自动重置失败",
    "durationLessThanMinute": "少于 1 分钟",
    "durationDays": "{n} 天",
    "durationHours": "{n} 小时",
    "durationMinutes": "{n} 分",
    "remainingPrefix": "剩余"
  },
  "onboarding": {
    "restartTour": "配置向导",
    "dontShowAgainTitle": "关闭账户概览提示",
    "confirmDontShow": "确定不再显示账户概览提示吗？\n\n您可以随时在右上角头像菜单中重新开启。",
    "confirmExit": "确定要退出账户概览提示吗？您可以随时在右上角菜单重新开始。",
    "interactiveHint": "按 Enter 继续",
    "admin": {
      "welcome": {
        "title": "配置向导",
        "description": "<p>配置路径由分组、上游账号、API 密钥三部分组成。</p><ul><li><b>分组</b>定义服务范围、费率和可见性。</li><li><b>账号池</b>承载上游模型服务。</li><li><b>API 密钥</b>用于分发和统计调用。</li></ul>",
        "nextBtn": "开始配置"
      },
      "groupManage": {
        "title": "第一步：分组管理",
        "description": "<p>分组决定用户可见的服务范围、计费倍率和专属访问权限。</p><ul><li>一个分组可以绑定多个上游账号。</li><li>公开分组面向所有用户，专属分组只面向指定用户。</li><li>费率在分组层统一生效。</li></ul>"
      },
      "createGroup": {
        "title": "创建分组",
        "description": "<p>新分组会作为后续账号绑定和密钥授权的基础。</p><p>保存前确认平台类型，创建后平台类型不可修改。</p>"
      },
      "groupName": {
        "title": "1. 分组名称",
        "description": "<p>名称用于后台识别和用户侧展示。</p><p>建议使用清晰、稳定的套餐或服务名称。</p>",
        "nextBtn": "继续"
      },
      "groupPlatform": {
        "title": "2. 平台类型",
        "description": "<p>平台类型决定该分组可绑定的上游账号类型。</p><ul><li>Anthropic 对应 Claude 系列。</li><li>OpenAI 对应 GPT 系列。</li><li>Google 对应 Gemini 系列。</li></ul>",
        "nextBtn": "继续"
      },
      "groupMultiplier": {
        "title": "3. 费率倍数",
        "description": "<p>费率倍数控制该分组的实际扣费。</p><ul><li>1.0 表示按原始用量计费。</li><li>大于 1.0 表示加价倍率。</li><li>小于 1.0 表示补贴倍率。</li></ul>",
        "nextBtn": "继续"
      },
      "groupExclusive": {
        "title": "4. 专属分组",
        "description": "<p>专属分组只对被授权的用户可见。</p><p>适用于高优先级套餐、企业客户或内部服务。</p>",
        "nextBtn": "继续"
      },
      "groupSubmit": {
        "title": "保存分组",
        "description": "<p>保存后，分组会进入账号绑定和 API 密钥授权流程。</p><p>平台类型保存后不可修改，名称、费率和可见性可后续调整。</p>"
      },
      "accountManage": {
        "title": "第二步：添加账号",
        "description": "<p>账号池承载上游模型服务，并通过分组对外提供。</p><ul><li>同一分组可以绑定多个账号。</li><li>系统会按优先级和可用性调度。</li><li>授权方式支持 OAuth 和 Session Key。</li></ul>"
      },
      "createAccount": {
        "title": "添加账号",
        "description": "<p>添加可用的上游账号后，分组才能实际承载模型调用。</p><p>OAuth 授权支持自动刷新，适合长期运行。</p>"
      },
      "accountName": {
        "title": "1. 账号名称",
        "description": "<p>名称只用于后台识别，不会影响调度。</p><p>可按平台、用途或优先级命名。</p>",
        "nextBtn": "继续"
      },
      "accountPlatform": {
        "title": "2. 平台类型",
        "description": "<p>账号平台必须与目标分组的平台一致。</p><p>不一致的账号不会被该分组调度。</p>",
        "nextBtn": "继续"
      },
      "accountType": {
        "title": "3. 授权方式",
        "description": "<p>OAuth 适合长期服务，支持授权刷新。</p><p>Session Key 适合不支持 OAuth 的平台，需要按上游状态维护。</p>",
        "nextBtn": "继续"
      },
      "accountPriority": {
        "title": "4. 调度优先级",
        "description": "<p>数字越小，调度优先级越高。</p><p>相同优先级会按可用性和调度策略分配。</p>",
        "nextBtn": "继续"
      },
      "accountGroups": {
        "title": "5. 绑定分组",
        "description": "<p>账号必须绑定到至少一个分组才会参与调用。</p><p>一个账号可以服务多个同平台分组。</p>",
        "nextBtn": "继续"
      },
      "accountSubmit": {
        "title": "保存账号",
        "description": "<p>保存后，OAuth 账号会进入上游授权流程。</p><p>授权成功返回后，账号即可按绑定分组参与调度。</p>"
      },
      "keyManage": {
        "title": "第三步：生成 API 密钥",
        "description": "<p>API 密钥承载用户调用、额度控制和用量统计。</p><ul><li>每个密钥绑定一个分组。</li><li>可设置配额、有效期和访问范围。</li><li>用量按密钥独立统计。</li></ul>"
      },
      "createKey": {
        "title": "生成密钥",
        "description": "<p>密钥生成后只展示一次，后台不再回显完整内容。</p><p>请确认所属分组和额度设置符合预期。</p>"
      },
      "keyName": {
        "title": "1. 密钥名称",
        "description": "<p>名称用于后台检索和用量归因。</p><p>可按用户、业务或接入场景命名。</p>",
        "nextBtn": "继续"
      },
      "keyGroup": {
        "title": "2. 绑定分组",
        "description": "<p>分组决定密钥可用的账号池、模型范围和计费倍率。</p><p>专属分组只对授权用户开放。</p>",
        "nextBtn": "继续"
      },
      "keySubmit": {
        "title": "生成并保存",
        "description": "<p>生成后会显示完整 API Key。</p><p>关闭弹窗后无法再次查看完整密钥，只能重新生成。</p>"
      }
    }
  },
  "payment": {
    "quickAmounts": "快捷到账余额",
    "customAmount": "自定义支付金额",
    "methods": {
      "gmpay": "GM Pay",
      "usdt": "USDT",
      "redeem_code": "卡密兑换",
      "admin_balance": "管理员调整",
      "affiliate_rebate": "邀请返佣"
    },
    "qr": {
      "scanToPay": "扫码支付",
      "scanAlipayHint": "支付宝扫码后，订单状态会自动同步。",
      "scanWxpayHint": "微信扫码后，订单状态会自动同步。",
      "payInNewWindow": "支付窗口已打开",
      "payInNewWindowHint": "支付页面已在新窗口打开，订单状态会在当前页面同步。",
      "expiredDesc": "该订单已超时，未产生扣款确认。",
      "cancelledDesc": "本次支付已结束，订单记录会保留。",
      "waitingPayment": "等待支付确认",
      "copyPayUrl": "复制支付链接",
      "copyQrUrl": "复制二维码链接",
      "openPayUrl": "打开支付链接",
      "mobileFallbackHint": "移动端扫码受限时，支付链接和二维码可在其他设备继续使用。",
      "assuranceTitle": "订单明细",
      "officialAssurance": "订单金额、支付方式和到账状态会保留。",
      "orderRecordAssurance": "订单号、支付状态和到账状态同步显示。",
      "billingProtectionAssurance": "支付结果和账户到账状态会同步更新。",
      "privacyAssurance": "订单记录会保留在账户内。"
    },
    "orders": {
      "orderId": "订单号",
      "baseAmount": "基础金额",
      "creditedBalance": "到账余额"
    },
    "result": {
      "processingHint": "支付结果仍在确认中，页面会自动刷新并保留订单记录。",
      "failed": "支付未完成",
      "backToRecharge": "余额充值",
      "viewOrders": "订单",
      "successHint": "到账和订阅状态已同步。",
      "failedHint": "未确认扣款不会计入到账；订单记录已保留。",
      "done": "完成",
      "orderSnapshot": "订单明细",
      "returnSnapshot": "支付返回",
      "assuranceTitle": "订单明细",
      "officialModels": "订单和到账状态已记录",
      "privacy": "订单记录会保留在账户内",
      "refundProtection": "可查看订单状态"
    },
    "groupFallback": "服务档位 #{id}",
    "confirmCancel": "取消后订单状态会同步更新。",
    "amountTooLow": "最低金额 {min}",
    "amountTooHigh": "最高金额 {max}",
    "rechargeRatePreview": "等价展示：1 账户余额 ≈ 官方 ¥{cny}，实际到账按充值金额一比一计算",
    "refundReason": "退款说明",
    "refundReasonPlaceholder": "金额、订单或到账信息",
    "stripeLoadFailed": "支付组件暂未完成加载，订单记录仍会保留。",
    "stripeMissingParams": "支付信息未就绪",
    "stripeNotConfigured": "该支付方式暂未开放",
    "airwallexLoadFailed": "Airwallex 支付组件暂未完成加载，订单记录仍会保留。",
    "airwallexMissingParams": "支付信息未就绪",
    "errors": {
      "tooManyPending": "待支付订单过多（最多 {max} 个），现有订单完成或取消后可继续下单",
      "cancelRateLimited": "取消操作较密集，稍后会恢复。",
      "wechatH5NotAuthorized": "当前微信 H5 支付暂不可用，微信内页面可继续完成支付。",
      "wechatPaymentMpNotConfigured": "当前微信内支付方式暂未开放。",
      "wechatJsapiUnavailable": "当前环境未能拉起微信支付，微信内页面可继续完成支付。",
      "wechatJsapiFailed": "微信支付未完成，扫码支付仍可使用。",
      "wechatUnavailable": "当前微信支付暂不可用。",
      "wechatOpenInWeChatHint": "当前页面链接可在微信内打开，也可使用电脑端微信扫码支付。",
      "wechatScanOnDesktopHint": "电脑端支持微信扫一扫，移动端支持微信内页面支付。",
      "wechatSwitchBrowserHint": "电脑端微信扫码和外部浏览器支付均可使用。",
      "mobilePaymentFallbackToQr": "移动支付暂不可用，已自动切换为扫码支付。",
      "alipayDesktopQrHint": "电脑端支付宝扫码单暂未生成，刷新后可重新获取。",
      "alipayMobileOpenHint": "当前页面可从系统浏览器重新发起支付宝支付。",
      "PAYMENT_DISABLED": "支付服务暂未开放",
      "USER_INACTIVE": "账户当前未启用",
      "BALANCE_PAYMENT_DISABLED": "余额充值暂未开放",
      "INVALID_INPUT": "输入信息不完整",
      "PLAN_NOT_AVAILABLE": "套餐暂未开放",
      "GROUP_NOT_FOUND": "订阅套餐暂未开放",
      "GROUP_TYPE_MISMATCH": "订阅套餐状态不可用",
      "TOO_MANY_PENDING": "待支付订单过多（最多 {max} 个），现有订单完成或取消后可继续下单",
      "PAYMENT_GATEWAY_ERROR": "支付方式当前未开放",
      "NO_AVAILABLE_INSTANCE": "暂无可用的支付方式",
      "PAYMENT_PROVIDER_MISCONFIGURED": "支付方式暂不可用",
      "WXPAY_CONFIG_MISSING_KEY": "支付信息未就绪",
      "WXPAY_CONFIG_INVALID_KEY_LENGTH": "支付信息未就绪",
      "WXPAY_CONFIG_INVALID_KEY": "支付信息未就绪",
      "PENDING_ORDERS": "该支付方式有未完成订单，订单完成后可继续操作",
      "PAYMENT_PROVIDER_CONFLICT": "该支付方式当前不可切换。",
      "CANCEL_RATE_LIMITED": "取消操作较密集，稍后会恢复",
      "CONFLICT": "订单状态已更新，刷新后可查看最新状态",
      "createOrderHint": "支付方式、金额和订单状态会保留在订单中。"
    },
    "stripePay": "确认支付",
    "stripeSuccessProcessing": "支付成功，订单状态正在同步。",
    "stripePopup": {
      "timeout": "等待支付凭证超时"
    },
    "subscribeNow": "选择套餐",
    "planCard": {
      "rate": "计费倍率",
      "dailyLimit": "每日额度",
      "weeklyLimit": "每周额度",
      "monthlyLimit": "每月额度",
      "quota": "周期额度",
      "models": "可用模型"
    },
    "admin": {
      "recordSource": "来源",
      "recordNotes": "备注",
      "sources": {
        "payment_order": "线上订单",
        "redeem_code": "卡密兑换",
        "admin_balance": "管理员调整",
        "affiliate_rebate": "邀请返佣"
      }
    },
    "rechargeTitle": "余额充值",
    "gateway": {
      "kicker": "Wallet",
      "description": "充值入口、支付方式、服务费、余额到账和套餐订阅集中展示，支付完成后订单和余额记录会同步更新。",
      "rechargeDescription": "选择充值方式，完成后余额自动更新。",
      "balance": "当前余额",
      "balanceDisabled": "已关闭",
      "balanceMeta": "可用于 API 调用和订阅扣费",
      "methods": "支付方式",
      "methodsMeta": "当前可用渠道",
      "plans": "订阅套餐",
      "plansMeta": "可购买套餐",
      "noPlansMeta": "暂无套餐",
      "feeRate": "服务费率",
      "feeRateMeta": "按当前配置计算",
      "noFee": "无"
    },
    "quickAmountPayDescription": "支付 {amount}",
    "quickAmountBelowLimit": "低于最低金额",
    "quickAmountBelowRechargeMin": "小额可走卡密",
    "quickAmountAboveLimit": "高于最高金额",
    "quickAmountUnavailable": "当前不可用",
    "customBalanceCredit": "自定义到账余额",
    "methodNoFee": "无额外手续费",
    "cardCodePurchase": {
      "title": "使用卡密充值",
      "description": "前往卡密购买页面获取卡密，购买后可在当前页面完成兑换。",
      "action": "前往购买卡密",
      "redeemTitle": "兑换卡密",
      "redeemDescription": "输入已购买的卡密，兑换成功后余额或权益会立即刷新。",
      "redeemLabel": "卡密",
      "redeemPlaceholder": "输入卡密",
      "redeemAction": "确认兑换"
    },
    "rechargeMethod": {
      "title": "充值方式",
      "unlockHint": "在线充值暂未开放，可继续使用卡密充值。",
      "unlockedHint": "当前账户可使用在线充值，也可以继续使用卡密充值。",
      "cardCodeTitle": "卡密充值",
      "cardCodeDescription": "跳转到卡密购买页面，购买后回到卡密兑换页完成充值。",
      "nativeTitle": "在线充值",
      "nativeDescription": "使用站内支付接口创建订单并自动入账。",
      "nativeLockedDescription": "当前在线充值通道暂未开放，可继续使用卡密充值。",
      "nativeUnavailableTitle": "当前不支持在线充值",
      "feeRate": "手续费 {rate}%",
      "noFee": "无手续费"
    },
    "confirmPayment": "确认支付",
    "orderPreview": "本次支付",
    "selectedPlan": "已选套餐",
    "changePlan": "更换套餐",
    "noPlansDesc": "可购买套餐开放后会显示在这里。",
    "trust": {
      "recharge": "到账记录",
      "privacy": "订单状态",
      "subscription": "订阅同步",
      "stability": "余额变化",
      "orders": "订单记录",
      "caption": "支付金额、到账余额、订阅状态和订单进度会保留在账户内。",
      "rechargeCaption": "支付金额、到账余额和订单进度会保留在账户内。"
    },
    "support": {
      "title": "订单记录",
      "detail": "订单号和到账状态会保留在账户内。",
      "action": "提交工单"
    },
    "amountTooLowCardCode": "在线充值最低 {min}，小于指定金额可使用卡密充值。",
    "airwallexRedirecting": "Airwallex 安全收银台正在打开，订单状态会同步到账户。",
    "stripeRedirecting": "Stripe 安全支付页正在打开，订单状态会同步到账户。",
    "stripeSecureCaption": "支付由 Stripe 安全处理，平台仅保留订单号、金额和状态用于核对。",
    "businessCategory": "业务分类",
    "businessCategories": {
      "recharge": "真实充值",
      "manual_collection": "人工收款",
      "manual_refund": "人工退款",
      "gift_compensation": "赠送补偿",
      "gift_reversal": "赠送扣减",
      "system_service_fee": "系统服务费",
      "affiliate_reward": "邀请返佣",
      "uncategorized": "未分类"
    }
  },
  "invoice": {
    "errors": {
      "INVOICE_NOT_FOUND": "开票申请不存在",
      "INVOICE_INVALID_STATUS": "当前开票申请状态不允许此操作，请刷新后重试",
      "INVOICE_AMOUNT_TOO_SMALL": "开票金额至少 {min_amount} 元",
      "INVOICE_AMOUNT_UNAVAILABLE": "开票金额超过当前可申请额度，可申请额度为 {available_amount} 元",
      "INVOICE_BALANCE_INSUFFICIENT": "账户余额不足，当前余额 {current_balance} 元，预计服务费 {tax_fee} 元",
      "INVOICE_TYPE_INVALID": "发票类型不正确",
      "INVOICE_TITLE_REQUIRED": "请填写发票抬头",
      "INVOICE_ITEM_REQUIRED": "请填写开票项目",
      "INVOICE_RECEIVER_EMAIL_REQUIRED": "请填写接收邮箱",
      "INVOICE_RECEIVER_EMAIL_INVALID": "接收邮箱格式不正确",
      "INVOICE_TAX_ID_REQUIRED": "企业发票需要填写税号",
      "INVOICE_REJECT_REASON_REQUIRED": "驳回时需要填写原因",
      "USER_NOT_FOUND": "用户不存在或已被禁用"
    },
    "page": {
      "title": "发票",
      "description": "订单和开票申请。",
      "notice": "最低开票金额 100 元，税点/服务费按申请金额的 3% 收取，无固定基础费。",
      "gateway": {
        "kicker": "Invoice Center",
        "description": ""
      },
      "trust": {
        "eligibleOrders": "",
        "amountReserved": "",
        "requestTrace": "",
        "privacy": ""
      },
      "stats": {
        "totalOrders": "线上订单",
        "availableOnPage": "本页可申请",
        "availableAmount": "可申请金额",
        "lockedAmount": "已占用金额",
        "unavailableOnPage": "本页暂不可用"
      },
      "orders": {
        "title": "订单",
        "description": "",
        "keywordLabel": "订单号 / 交易单号",
        "keywordPlaceholder": "搜索订单号",
        "statusLabel": "支付状态",
        "invoiceabilityLabel": "可申请状态",
        "startDate": "开始时间",
        "endDate": "结束时间",
        "emptyTitle": "暂无符合条件的订单",
        "emptyDescription": "没有线上订单不影响按上方可申请金额提交。",
        "columns": {
          "order": "订单",
          "orderId": "线上订单号",
          "tradeNo": "交易单号",
          "amount": "可申请金额",
          "fee": "手续费",
          "method": "支付方式",
          "status": "支付状态",
          "paidAt": "支付时间",
          "invoiceability": "可申请状态",
          "reason": "原因"
        },
        "invoiceability": {
          "available": "可申请",
          "unavailable": "暂不可用"
        },
        "refundApplied": "已扣退款 {amount}"
      },
      "form": {
        "title": "发票信息",
        "selectedSummary": "线上订单已选 {count} 笔 · {amount}",
        "saveTemplate": "保存开票信息",
        "limitHint": "最低开票金额 {min}，当前可申请 {available}。",
        "taxHint": "税点/服务费按 {rate}% 计算，无固定基础费；预计服务费 {tax}。",
        "balanceInsufficientHint": "当前余额 {balance}，不足以支付本次预计服务费 {tax}。",
        "templateLabel": "选择模板",
        "noTemplate": "不使用模板",
        "noTemplates": "暂无模板",
        "defaultSuffix": "（默认）",
        "updateTemplate": "更新开票信息",
        "setDefault": "设为默认",
        "deleteTemplate": "删除",
        "typeLabel": "发票类型",
        "amountLabel": "开票金额",
        "amountPlaceholder": "100.00",
        "titleLabel": "发票抬头",
        "titlePlaceholder": "公司全称或个人姓名",
        "taxIdLabel": "税号",
        "taxIdPlaceholder": "纳税人识别号",
        "itemNameLabel": "开票项目",
        "itemNamePlaceholder": "信息技术服务费",
        "receiverEmailLabel": "接收邮箱",
        "receiverEmailPlaceholder": "接收电子发票的邮箱",
        "noteLabel": "备注",
        "notePlaceholder": "可选",
        "submitting": "处理中",
        "submit": "申请开票",
        "minimumNotMet": "当前可申请金额未达到起开金额。",
        "defaultItemName": "信息技术服务费"
      },
      "records": {
        "title": "申请记录",
        "description": "",
        "emptyTitle": "暂无申请记录",
        "emptyDescription": "提交后的开票申请会显示在这里。",
        "amountByTotal": "按金额",
        "cancel": "取消",
        "columns": {
          "title": "发票抬头",
          "type": "发票类型",
          "amount": "开票金额",
          "taxFee": "服务费",
          "orderCount": "申请方式",
          "status": "状态",
          "number": "发票号码",
          "submittedAt": "提交时间",
          "note": "备注",
          "actions": "操作"
        }
      },
      "dialog": {
        "createTitle": "保存开票模板",
        "updateTitle": "更新开票模板",
        "templateName": "模板名称",
        "templateNamePlaceholder": "默认模板",
        "defaultTemplateName": "默认模板",
        "defaultTemplate": "设为默认模板"
      },
      "types": {
        "company_vat_general": "普通发票",
        "company_vat_special": "专用发票",
        "personal": "个人发票"
      },
      "invoiceability": {
        "all": "全部线上订单",
        "available": "可申请",
        "unavailable": "暂不可申请"
      },
      "reasons": {
        "notBalance": "非余额充值订单",
        "notCompleted": "订单未完成",
        "fullyRefunded": "已全额退款",
        "zeroAmount": "订单金额为 0",
        "notInvoiceableSource": "该余额来源不可开票"
      },
      "status": {
        "pending": "待确认",
        "approved": "已确认",
        "rejected": "已驳回",
        "completed": "已完成",
        "cancelled": "已取消"
      },
      "messages": {
        "loadSummaryFailed": "加载开票额度失败",
        "loadInvoicesFailed": "加载开票申请失败",
        "loadOrdersFailed": "加载订单失败",
        "loadTemplatesFailed": "加载开票模板失败",
        "submitSuccess": "开票申请已提交",
        "submitFailed": "提交申请失败",
        "cancelSuccess": "已取消开票申请",
        "cancelFailed": "取消失败",
        "templateUpdated": "开票模板已更新",
        "templateSaved": "开票模板已保存",
        "saveTemplateFailed": "保存开票模板失败",
        "defaultTemplateUpdated": "默认模板已更新",
        "defaultTemplateFailed": "设置默认模板失败",
        "deleteTemplateConfirm": "删除这个开票模板？",
        "templateDeleted": "开票模板已删除",
        "deleteTemplateFailed": "删除开票模板失败"
      }
    }
  },
  "tickets": {
    "title": "工单",
    "description": "用量、订单、账户和服务问题的沟通记录。",
    "assurance": "",
    "gateway": {
      "kicker": "Support Desk",
      "panelTitle": "",
      "totalTickets": "全部工单",
      "totalTicketsMeta": "当前筛选结果",
      "activeTickets": "处理中",
      "unreadMessages": "未读消息",
      "needsAttention": "需要查看"
    },
    "searchPlaceholder": "搜索工单标题或编号...",
    "createTicket": "创建工单",
    "submitTicket": "提交工单",
    "viewDetail": "查看",
    "detailTitle": "工单详情",
    "empty": "暂无工单",
    "emptyDescription": "工单沟通会显示在这里。",
    "unreadCount": "{count} 条未读",
    "noUnread": "无未读",
    "lastMessageAt": "最后消息",
    "attachments": {
      "title": "附件",
      "add": "添加附件",
      "namePlaceholder": "名称",
      "urlPlaceholder": "https://... 或选择图片",
      "chooseImage": "选择图片",
      "imageSelected": "已选择图片",
      "invalidImage": "请选择 PNG、JPG、WebP 或 GIF 图片",
      "imageTooLarge": "图片不能超过 {size}MB",
      "readFailed": "读取所选图片失败"
    },
    "replyAttachments": {
      "title": "附件",
      "add": "添加附件",
      "namePlaceholder": "名称",
      "urlPlaceholder": "https://... 或选择图片",
      "chooseImage": "选择图片",
      "imageSelected": "已选择图片",
      "invalidImage": "请选择 PNG、JPG、WebP 或 GIF 图片",
      "imageTooLarge": "图片不能超过 {size}MB",
      "readFailed": "读取所选图片失败"
    },
    "trust": {
      "privacy": "",
      "billing": "",
      "traceable": ""
    },
    "context": {
      "general": "关联上下文",
      "usage": "用量记录",
      "order": "订单",
      "api_key": "访问凭证",
      "invoice": "发票",
      "request": "请求",
      "request_id": "请求 ID"
    },
    "contextFields": {
      "request_id": "请求 ID",
      "model": "模型",
      "api_key_id": "API Key",
      "group_id": "分组 ID",
      "group_name": "分组",
      "inbound_endpoint": "入口端点",
      "upstream_endpoint": "上游端点",
      "actual_cost": "实际扣费",
      "total_cost": "账面费用",
      "duration_ms": "总耗时",
      "first_token_ms": "首字耗时",
      "status_code": "服务状态码",
      "category": "问题分类",
      "platform": "模型服务",
      "error_message": "问题摘要",
      "api_key_name": "凭证名称",
      "recommended_action": "建议动作",
      "created_at": "请求时间",
      "recent_order_ids": "关联订单",
      "recent_orders": "订单摘要",
      "itemCount": "{count} 项"
    },
    "closeTicket": "关闭工单",
    "reopenTicket": "重新打开",
    "closedReplyHint": "该工单已关闭，重新打开后可继续沟通。",
    "reply": "回复",
    "replyPlaceholder": "问题、截图链接或新的排查信息...",
    "sendReply": "发送回复",
    "created": "工单已创建",
    "replySent": "回复已发送",
    "closed": "工单已关闭",
    "reopened": "工单已重新打开",
    "failedToLoad": "加载工单失败",
    "failedToCreate": "创建工单失败",
    "failedToLoadDetail": "加载工单详情失败",
    "failedToReply": "发送回复失败",
    "failedToUpdate": "更新工单失败",
    "columns": {
      "subject": "主题",
      "status": "状态",
      "category": "分类",
      "priority": "优先级",
      "lastMessageAt": "最后消息",
      "unread": "未读"
    },
    "form": {
      "template": "问题类型",
      "superAdminHint": "该类型会进入专项支持队列，并发送邮件提醒。",
      "subject": "主题",
      "category": "分类",
      "priority": "优先级",
      "body": "问题描述",
      "bodyPlaceholder": "问题、请求 ID、时间范围或相关截图链接",
      "privacyNote": "提交内容和附件只用于本次问题处理。涉及用量或扣费时，关联明细用于定位时间、模型与金额。",
      "bodyMinLength": "至少填写 {count} 个字。",
      "imagePlaceholder": "也可以粘贴支付宝/微信截图或问题截图的图片链接",
      "chooseImage": "选择图片",
      "imageSelected": "已选择图片",
      "viewImage": "查看图片",
      "amountPlaceholder": "请输入未到账金额",
      "orderAmount": "到账金额 {amount}，支付金额 {pay}",
      "noRecentOrders": "最近充值记录为空；订单号和付款时间可写入描述。",
      "contextType": "关联类型",
      "contextTypePlaceholder": "usage / order / api_key",
      "contextId": "关联 ID",
      "contextIdPlaceholder": "关联记录 ID",
      "usageBodyPrefill": "请描述这次请求遇到的问题。系统已关联用量记录 #{usageId}、请求 ID {requestId}、模型 {model}，客服可据此排查扣费、延迟或失败原因。",
      "requestBodyPrefill": "请描述这条问题记录的影响。系统已关联请求 {requestId}、模型 {model}、服务状态码 {status}、分类 {category}。建议先执行：{action}",
      "requestAdviceTitle": "建议先检查",
      "requestAdvice": {
        "auth": "请先确认使用的 API Key、分组权限和客户端 Base URL 是否匹配；如果凭证已删除或重置，请切换到新的访问凭证。",
        "rate_limit": "这通常是限流或并发过高。可以稍后重试、降低并发，或切换到可用容量更高的分组。",
        "quota": "请先检查账户余额、分组额度或订阅状态；如果余额已充值但仍失败，请在工单里附上充值记录。",
        "invalid_request": "请先检查模型名、endpoint、请求参数和客户端版本；如果同样请求在其他分组可用，请说明可用分组。",
        "service_unavailable": "这通常是模型服务暂时不可用或过载。可以稍后重试，或切换到其他可用分组。",
        "upstream": "模型服务返回了异常。请保留请求时间、模型、分组和客户端提示，客服会结合关联记录排查。",
        "internal": "平台侧出现异常。请提交工单，客服会按关联请求继续排查。",
        "cyber": "请求可能触发了安全策略。请检查提示词、附件和调用用途，并在工单里说明业务场景。",
        "default": "请补充客户端提示、请求时间、模型和分组；客服会结合关联记录继续排查。"
      },
      "attachmentHint": {
        "request": "如客户端有截图、请求片段或终端报错，可作为附件补充；不要上传完整密钥。",
        "order": "支付或到账问题建议附支付截图、订单号或付款时间。",
        "invoice": "发票问题建议附开票信息截图、订单号或驳回原因截图。"
      }
    },
    "filters": {
      "allStatus": "全部状态",
      "allPriority": "全部优先级",
      "allCategory": "全部分类",
      "allUnread": "全部",
      "onlyUnread": "仅未读"
    },
    "status": {
      "open": "待处理",
      "pending": "已回复",
      "resolved": "已解决",
      "closed": "已关闭"
    },
    "priority": {
      "low": "低",
      "normal": "普通",
      "high": "高",
      "urgent": "紧急"
    },
    "category": {
      "general": "通用",
      "billing": "计费",
      "usage": "用量",
      "technical": "技术",
      "account": "账户"
    },
    "sender": {
      "user": "我",
      "admin": "客服",
      "system": "平台"
    },
    "errors": {
      "requiredFields": "主题和问题描述不能为空",
      "TICKET_INPUT_REQUIRED": "工单参数不能为空",
      "TICKET_SUBJECT_INVALID": "工单标题不能为空，且不能超过 200 个字符",
      "TICKET_BODY_REQUIRED": "消息内容不能为空",
      "TICKET_STATUS_INVALID": "工单状态不正确",
      "TICKET_PRIORITY_INVALID": "优先级不正确",
      "TICKET_CATEGORY_INVALID": "工单分类不正确",
      "TICKET_TEMPLATE_INVALID": "工单类型不正确",
      "TICKET_TEMPLATE_FIELD_INVALID": "工单类型必填信息未完整",
      "bodyTooShort": "问题描述至少需要 {count} 个字",
      "templateFieldRequired": "请填写「{field}」",
      "imageURLRequired": "「{field}」请选择图片，或填写有效的 http(s) 图片链接",
      "imageFileRequired": "请选择图片文件",
      "imageTooLarge": "图片不能超过 {size}MB",
      "imageReadFailed": "读取图片失败，请重新选择",
      "TICKET_ATTACHMENT_INVALID": "附件名称和链接必须有效，图片最大 2MB，最多 5 个",
      "TICKET_CLOSED": "工单已关闭，重新打开后可继续沟通",
      "TICKET_NOT_FOUND": "工单不存在"
    }
  },
  "settlementCurrency": {
    "label": "结算币种",
    "cny": "人民币",
    "usd": "美元",
    "baseCredit": "账户余额"
  },
  "platformUsage": {
    "today": "今日",
    "total": "累计",
    "breakdown": "按模型服务拆分",
    "other": "其他"
  },
  "userOrders": {
    "description": "订单列表。",
    "gateway": {
      "kicker": "Billing Records",
      "description": ""
    },
    "newOrder": "余额充值",
    "statusFilter": "订单状态",
    "cancelDescription": "取消后本次支付结束，订单记录仍会保留。",
    "refundNote": "退款说明",
    "refundNotePlaceholder": "金额、订单或到账情况",
    "trust": {
      "amount": "",
      "status": "",
      "privacy": "",
      "support": ""
    },
    "summary": {
      "label": "订单摘要",
      "total": "全部订单",
      "totalHint": "跨页统计。",
      "currentPage": "当前页",
      "currentPageHint": "本页订单金额和状态一览。",
      "completedOnPage": "本页已完成",
      "completedOnPageHint": "已完成订单可与到账结果对应。"
    }
  }
} as const

export default custom
