// Package settings 系统设置(KV)。
//
// 设计要点:
//   - 存储:system_settings(k PRIMARY KEY, v TEXT)
//   - 缓存:启动时从 DB 全量加载到内存 map,Set 时写库 + 原子替换缓存
//   - 读取:全应用都通过 Service.GetXxx 读,纳秒级;写入只有管理员能触发
//   - 热更:无需重启,改完立即对后续请求生效
//   - 白名单:所有允许的 key 都在 Keys 表里声明,未声明的 key 写入会被拒绝,
//     避免前端/客户端乱写键污染表
package settings

import "strings"

// Keys 允许的全部 key + 类型 schema(类型仅用于校验/解析)。
// 默认值仅在 DB 里缺失时使用(migration 已种子化,通常不会用到)。
type KeyDef struct {
	Key      string
	Type     string // "string" | "bool" | "int" | "float" | "email" | "url"
	Category string // "site" | "auth" | "defaults" | "gateway" | "billing" | "mail"
	Default  string
	Label    string
	Desc     string
	Public   bool // true 则 /api/public/site-info 会返回给匿名访问者
}

// ---- key 常量 ----
const (
	// 通用
	SiteName         = "site.name"
	SiteDescription  = "site.description"
	SiteLogoURL      = "site.logo_url"
	SiteFooter       = "site.footer"
	SiteContactEmail = "site.contact_email"
	SiteDocsURL       = "site.docs_url"
	SiteAPIBaseURL    = "site.api_base_url"
	SiteAnnouncementTitle = "site.announcement_title"
	SiteAnnouncementHTML  = "site.announcement_html"
	SiteAnnouncementPopupEnabled = "site.announcement_popup_enabled"
	SiteAnnouncementVersion = "site.announcement_version"
	UIDefaultPageSize = "ui.default_page_size"

	// 安全与认证
	AuthAllowRegister       = "auth.allow_register"
	AuthRequireEmailVerify  = "auth.require_email_verify"
	AuthEmailDomainWhitelist = "auth.email_domain_whitelist"
	AuthPasswordMinLength   = "auth.password_min_length"
	AuthInviteCodeRequired  = "auth.invite_code_required"
	Auth2FAEnabled          = "auth.2fa_enabled"
	AuthJWTAccessTTLSec     = "auth.jwt_access_ttl_sec"
	AuthJWTRefreshTTLSec    = "auth.jwt_refresh_ttl_sec"

	// 用户默认值(旧 auth.* 保留兼容)
	AuthDefaultGroupID       = "auth.default_group_id"
	AuthSignupBonusCredits   = "auth.signup_bonus_credits"
	LimitDefaultRPM          = "limit.default_rpm"
	LimitDefaultTPM          = "limit.default_tpm"
	KeyDefaultDailyQuota     = "key.default_daily_quota_credits"
	KeyMaxPerUser            = "key.max_per_user"

	// 网关与调度
	GatewayUpstreamTimeoutSec = "gateway.upstream_timeout_sec"
	GatewaySSEReadTimeoutSec  = "gateway.sse_read_timeout_sec"
	GatewayCooldown429Sec     = "gateway.cooldown_429_sec"
	GatewayWarnedPauseHours   = "gateway.warned_pause_hours"
	GatewayDailyUsageRatio    = "gateway.daily_usage_ratio"
	GatewayRetryOnFailure     = "gateway.retry_on_failure"
	GatewayRetryMax           = "gateway.retry_max"
	GatewayDispatchQueueWaitSec = "gateway.dispatch_queue_wait_sec"

	// 代理管理(健康探测)
	ProxyProbeEnabled     = "proxy.probe_enabled"
	ProxyProbeIntervalSec = "proxy.probe_interval_sec"
	ProxyProbeTimeoutSec  = "proxy.probe_timeout_sec"
	ProxyProbeTargetURL   = "proxy.probe_target_url"
	ProxyProbeConcurrency = "proxy.probe_concurrency"

	// 账号刷新 & 额度探测
	AccountRefreshEnabled        = "account.refresh_enabled"
	AccountRefreshIntervalSec    = "account.refresh_interval_sec"
	AccountRefreshAheadSec       = "account.refresh_ahead_sec"
	AccountRefreshConcurrency    = "account.refresh_concurrency"
	AccountQuotaProbeEnabled     = "account.quota_probe_enabled"
	AccountQuotaProbeIntervalSec = "account.quota_probe_interval_sec"
	AccountDefaultClientID       = "account.default_client_id"

	// 计费与充值
	BillingCreditPerCNY         = "billing.credit_per_cny"
	BillingNotifyAdminOnAdjust  = "billing.notify_admin_on_adjust"
	RechargeEnabled             = "recharge.enabled"
	RechargeMinCNY              = "recharge.min_cny"
	RechargeMaxCNY              = "recharge.max_cny"
	RechargeDailyLimitCNY       = "recharge.daily_limit_cny"
	RechargeOrderExpireMinutes  = "recharge.order_expire_minutes"

	// 邮件
	MailEnabledDisplay = "mail.enabled_display"
)

// Defs 所有合法 key 的 schema。前端编辑页按 category + order 展示。
var Defs = []KeyDef{
	// ---------- 通用 ----------
	{Key: SiteName, Type: "string", Category: "site", Default: "GPT2API", Label: "站点名称", Desc: "展示在顶栏和登录页大标题", Public: true},
	{Key: SiteDescription, Type: "string", Category: "site", Default: "企业级 OpenAI 兼容网关", Label: "副标题", Desc: "登录页宣传语", Public: true},
	{Key: SiteLogoURL, Type: "url", Category: "site", Default: "", Label: "Logo URL", Desc: "空则使用默认图标", Public: true},
	{Key: SiteFooter, Type: "string", Category: "site", Default: "", Label: "页脚文案", Desc: "版权/备案号等(纯文本)", Public: true},
	{Key: SiteContactEmail, Type: "email", Category: "site", Default: "", Label: "联系邮箱", Desc: "对外展示的客服邮箱", Public: true},
	{Key: SiteDocsURL, Type: "url", Category: "site", Default: "", Label: "文档链接", Desc: "留空则前端隐藏「文档」入口", Public: true},
	{Key: SiteAPIBaseURL, Type: "url", Category: "site", Default: "", Label: "API Base URL", Desc: "展示给用户的 /v1 入口;留空=当前站点地址", Public: true},
	{Key: SiteAnnouncementTitle, Type: "string", Category: "site", Default: "", Label: "公告标题", Desc: "展示在用户总览弹窗和公告按钮中", Public: true},
	{Key: SiteAnnouncementHTML, Type: "html", Category: "site", Default: "", Label: "公告内容 HTML", Desc: "支持富文本编辑和 HTML 渲染", Public: true},
	{Key: SiteAnnouncementPopupEnabled, Type: "bool", Category: "site", Default: "false", Label: "自动弹窗", Desc: "开启后用户进入总览页可自动弹出公告", Public: true},
	{Key: SiteAnnouncementVersion, Type: "int", Category: "site", Default: "0", Label: "公告版本", Desc: "用于前端红点和“今日不再提醒”状态隔离", Public: true},
	{Key: UIDefaultPageSize, Type: "int", Category: "site", Default: "20", Label: "默认每页条数", Desc: "后台表格默认分页(5~100)"},

	// ---------- 安全与认证 ----------
	{Key: AuthAllowRegister, Type: "bool", Category: "auth", Default: "true", Label: "开放注册", Desc: "关闭后仅管理员可创建用户", Public: true},
	{Key: AuthRequireEmailVerify, Type: "bool", Category: "auth", Default: "false", Label: "邮箱验证", Desc: "注册时必须验证邮箱(预留;尚未实装)"},
	{Key: AuthEmailDomainWhitelist, Type: "string", Category: "auth", Default: "", Label: "邮箱域名白名单", Desc: "逗号分隔,如 qq.com,gmail.com;留空=不限"},
	{Key: AuthPasswordMinLength, Type: "int", Category: "auth", Default: "6", Label: "密码最小长度", Desc: "注册/改密时强制校验(6~64)"},
	{Key: AuthInviteCodeRequired, Type: "bool", Category: "auth", Default: "false", Label: "邀请码注册", Desc: "必须邀请码才能注册(预留)"},
	{Key: Auth2FAEnabled, Type: "bool", Category: "auth", Default: "false", Label: "二次验证 2FA", Desc: "允许用户绑定 TOTP(预留)"},
	{Key: AuthJWTAccessTTLSec, Type: "int", Category: "auth", Default: "7200", Label: "Access Token TTL(秒)", Desc: "默认 7200(2 小时);改后新发 token 生效"},
	{Key: AuthJWTRefreshTTLSec, Type: "int", Category: "auth", Default: "604800", Label: "Refresh Token TTL(秒)", Desc: "默认 604800(7 天)"},

	// ---------- 用户默认值 ----------
	{Key: AuthDefaultGroupID, Type: "int", Category: "defaults", Default: "1", Label: "默认分组 ID", Desc: "新用户自动加入的分组(对应 user_groups.id)"},
	{Key: AuthSignupBonusCredits, Type: "int", Category: "defaults", Default: "0", Label: "注册赠送积分", Desc: "单位:厘,10000 = 1 积分"},
	{Key: LimitDefaultRPM, Type: "int", Category: "defaults", Default: "60", Label: "默认 RPM", Desc: "未被 key/group 覆盖时生效"},
	{Key: LimitDefaultTPM, Type: "int", Category: "defaults", Default: "60000", Label: "默认 TPM", Desc: ""},
	{Key: KeyDefaultDailyQuota, Type: "int", Category: "defaults", Default: "0", Label: "API Key 默认日配额", Desc: "单位:厘;0=不限"},
	{Key: KeyMaxPerUser, Type: "int", Category: "defaults", Default: "20", Label: "单用户最多 Key 数", Desc: "0=不限"},

	// ---------- 网关与调度 ----------
	{Key: GatewayUpstreamTimeoutSec, Type: "int", Category: "gateway", Default: "60", Label: "上游请求超时(秒)", Desc: "非流式请求上游响应超时"},
	{Key: GatewaySSEReadTimeoutSec, Type: "int", Category: "gateway", Default: "120", Label: "SSE 读超时(秒)", Desc: "流式响应无数据时的中断阈值"},
	{Key: GatewayCooldown429Sec, Type: "int", Category: "gateway", Default: "300", Label: "429 冷却(秒)", Desc: "账号遇 429 后暂停调度"},
	{Key: GatewayWarnedPauseHours, Type: "int", Category: "gateway", Default: "24", Label: "风险暂停(小时)", Desc: "账号被识别为 warned 时的暂停时长"},
	{Key: GatewayDailyUsageRatio, Type: "float", Category: "gateway", Default: "0.8", Label: "日用比例阈值", Desc: "0.0~1.0;超过后降低调度优先级"},
	{Key: GatewayRetryOnFailure, Type: "bool", Category: "gateway", Default: "true", Label: "失败自动重试", Desc: "遇到可恢复错误时切换账号重试"},
	{Key: GatewayRetryMax, Type: "int", Category: "gateway", Default: "1", Label: "最大重试次数", Desc: "0~3"},
	{Key: GatewayDispatchQueueWaitSec, Type: "int", Category: "gateway", Default: "120", Label: "账号排队等待上限(秒)", Desc: "并发大于账号数时,请求会在队列里等空闲账号;超过此秒数仍拿不到才返回 no_available_account。0=不排队,立即失败"},

	// ---------- 代理管理(健康探测) ----------
	{Key: ProxyProbeEnabled, Type: "bool", Category: "gateway", Default: "true", Label: "代理探测开关", Desc: "开启后后台定时对启用的代理做连通性探测,更新健康分"},
	{Key: ProxyProbeIntervalSec, Type: "int", Category: "gateway", Default: "300", Label: "代理探测间隔(秒)", Desc: "两轮探测之间的间隔,建议 ≥ 60"},
	{Key: ProxyProbeTimeoutSec, Type: "int", Category: "gateway", Default: "10", Label: "代理探测超时(秒)", Desc: "单条代理一次探测的超时时间"},
	{Key: ProxyProbeTargetURL, Type: "url", Category: "gateway", Default: "", Label: "代理探测目标 URL", Desc: "返回 2xx/3xx 视为成功。**留空(推荐)**时走内置候选链(api.ipify.org / cloudflare cdn-cgi/trace / httpbin.org),任意一个通过即判代理可用;填单一 URL 则只用该地址(代理若恰好被该站限制会被误判为失败)"},
	{Key: ProxyProbeConcurrency, Type: "int", Category: "gateway", Default: "8", Label: "代理探测并发", Desc: "同时探测的代理数(1~64)"},

	// ---------- 账号池(AT 刷新 / 额度探测) ----------
	{Key: AccountRefreshEnabled, Type: "bool", Category: "gateway", Default: "true", Label: "账号 AT 自动刷新", Desc: "后台定时扫账号,即将过期的 AT 自动用 RT/ST 刷新"},
	{Key: AccountRefreshIntervalSec, Type: "int", Category: "gateway", Default: "120", Label: "账号刷新扫描间隔(秒)", Desc: "多久扫一次,建议 60~300"},
	{Key: AccountRefreshAheadSec, Type: "int", Category: "gateway", Default: "900", Label: "账号预刷新提前量(秒)", Desc: "距离过期多少秒内就触发刷新,建议 ≥ 300"},
	{Key: AccountRefreshConcurrency, Type: "int", Category: "gateway", Default: "4", Label: "账号刷新并发", Desc: "同时刷新的账号数(1~32)"},
	{Key: AccountQuotaProbeEnabled, Type: "bool", Category: "gateway", Default: "true", Label: "账号额度自动探测", Desc: "后台定期查询账号的图片剩余额度"},
	{Key: AccountQuotaProbeIntervalSec, Type: "int", Category: "gateway", Default: "18000", Label: "额度探测最小间隔(秒)", Desc: "同一账号两次探测的最小间隔,默认 **18000=5 小时**;当账号剩余额度=0 且已过重置时间时,会忽略此间隔立即补探一次,以第一时间反映真实恢复额度"},
	{Key: AccountDefaultClientID, Type: "string", Category: "gateway", Default: "app_EMoamEEZ73f0CkXaXp7hrann", Label: "导入账号默认 client_id", Desc: "JSON 未指定时使用的 OAuth client_id"},

	// ---------- 计费与充值 ----------
	{Key: BillingCreditPerCNY, Type: "int", Category: "billing", Default: "10000", Label: "1 元 = N 积分·厘", Desc: "展示用换算;默认 10000"},
	{Key: BillingNotifyAdminOnAdjust, Type: "bool", Category: "billing", Default: "false", Label: "调账邮件通知", Desc: "管理员调账时邮件通知超管(预留)"},
	{Key: RechargeEnabled, Type: "bool", Category: "billing", Default: "true", Label: "启用充值", Desc: "关闭后前端隐藏充值菜单,下单接口返回 403", Public: true},
	{Key: RechargeMinCNY, Type: "int", Category: "billing", Default: "100", Label: "最低金额(分)", Desc: "100 = 1 元"},
	{Key: RechargeMaxCNY, Type: "int", Category: "billing", Default: "0", Label: "最高金额(分)", Desc: "0=不限"},
	{Key: RechargeDailyLimitCNY, Type: "int", Category: "billing", Default: "0", Label: "单用户每日上限(分)", Desc: "0=不限;按今日已支付订单金额累计"},
	{Key: RechargeOrderExpireMinutes, Type: "int", Category: "billing", Default: "30", Label: "订单有效期(分钟)", Desc: "到期未支付自动取消"},

	// ---------- 邮件 ----------
	{Key: MailEnabledDisplay, Type: "string", Category: "mail", Default: "auto", Label: "邮件开关展示", Desc: "auto/true/false;实际是否发邮件由 SMTP 配置决定"},
}

// DefByKey 快速查一条 schema。
func DefByKey(k string) (KeyDef, bool) {
	for _, d := range Defs {
		if d.Key == k {
			return d, true
		}
	}
	return KeyDef{}, false
}

// IsAllowedKey 白名单判定。
func IsAllowedKey(k string) bool {
	k = strings.TrimSpace(k)
	if k == "" {
		return false
	}
	_, ok := DefByKey(k)
	return ok
}
