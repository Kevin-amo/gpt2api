package server

import (
	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/account"
	"github.com/432539/gpt2api/internal/apikey"
	"github.com/432539/gpt2api/internal/audit"
	"github.com/432539/gpt2api/internal/auth"
	"github.com/432539/gpt2api/internal/backup"
	"github.com/432539/gpt2api/internal/channel"
	"github.com/432539/gpt2api/internal/config"
	"github.com/432539/gpt2api/internal/gateway"
	"github.com/432539/gpt2api/internal/image"
	"github.com/432539/gpt2api/internal/middleware"
	"github.com/432539/gpt2api/internal/model"
	"github.com/432539/gpt2api/internal/proxy"
	"github.com/432539/gpt2api/internal/rbac"
	"github.com/432539/gpt2api/internal/recharge"
	"github.com/432539/gpt2api/internal/settings"
	"github.com/432539/gpt2api/internal/usage"
	"github.com/432539/gpt2api/internal/user"
	pkgjwt "github.com/432539/gpt2api/pkg/jwt"
	"github.com/432539/gpt2api/pkg/resp"
)

// Deps 汇集路由依赖。
type Deps struct {
	Config *config.Config
	JWT    *pkgjwt.Manager

	AuthH *auth.Handler
	UserH *user.Handler

	KeySvc     *apikey.Service
	KeyH       *apikey.Handler
	ProxyH     *proxy.Handler
	AccountH   *account.Handler
	ChannelH   *channel.Handler

	GatewayH *gateway.Handler
	ImagesH  *gateway.ImagesHandler

	BackupH      *backup.Handler
	AuditH       *audit.Handler
	AuditDAO     *audit.DAO
	AdminUserH   *user.AdminHandler
	AdminGroupH  *user.AdminGroupHandler

	AdminModelH *model.AdminHandler
	AdminKeyH   *apikey.AdminHandler
	AdminUsageH *usage.AdminHandler

	// 生成面板:当前用户视角的 usage / image 只读接口
	MeUsageH *usage.MeHandler
	MeImageH *image.MeHandler

	AdminImageH *image.AdminHandler

	RechargeH      *recharge.Handler
	AdminRechargeH *recharge.AdminHandler

	SettingsH *settings.Handler
}

// New 构建 gin.Engine 并挂载所有路由。
func New(d *Deps) *gin.Engine {
	if d.Config.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.Recover(),
		middleware.AccessLog(),
		middleware.CORS(d.Config.Security.CORSOrigins),
	)

	r.GET("/healthz", func(c *gin.Context) { resp.OK(c, gin.H{"status": "ok"}) })
	r.GET("/readyz", func(c *gin.Context) { resp.OK(c, gin.H{"status": "ok"}) })

	// ---- 内部管理 API(JWT) ----
	api := r.Group("/api")
	{
		authGrp := api.Group("/auth")
		{
			authGrp.POST("/register", d.AuthH.Register)
			authGrp.POST("/login", d.AuthH.Login)
			authGrp.POST("/refresh", d.AuthH.Refresh)
		}

		authed := api.Group("", middleware.JWTAuth(d.JWT))
		{
			authed.GET("/me", d.UserH.Me)
			authed.GET("/me/menu", d.UserH.Menu)

			// 用户端 API Key 管理(需 self:key 权限,普通用户/管理员都持有)
			keys := authed.Group("/keys", middleware.RequirePerm(rbac.PermSelfKey))
			{
				keys.POST("", d.KeyH.Create)
				keys.GET("", d.KeyH.List)
				keys.PATCH("/:id", d.KeyH.Update)
				keys.DELETE("/:id", d.KeyH.Delete)
			}

			// 充值(自己的订单、下单、取消)
			if d.RechargeH != nil {
				rg := authed.Group("/recharge", middleware.RequirePerm(rbac.PermSelfRecharge))
				{
					rg.GET("/packages", d.RechargeH.ListPackages)
					rg.POST("/orders", d.RechargeH.CreateOrder)
					rg.GET("/orders", d.RechargeH.ListMyOrders)
					rg.POST("/orders/:id/cancel", d.RechargeH.CancelOrder)
					rg.GET("/redeem-status", d.RechargeH.RedeemStatus)
					rg.POST("/redeem", d.RechargeH.Redeem)
				}
			}

			// 生成面板:当前用户的用量明细(文字 token) + 图片任务历史
			if d.MeUsageH != nil {
				ug := authed.Group("/me/usage", middleware.RequirePerm(rbac.PermSelfUsage))
				{
					ug.GET("/logs", d.MeUsageH.Logs)
					ug.GET("/stats", d.MeUsageH.Stats)
				}
			}
			// 当前用户的积分流水(只读)
			authed.GET("/me/credit-logs",
				middleware.RequirePerm(rbac.PermSelfUsage), d.UserH.CreditLogs)
			if d.MeImageH != nil {
				ig := authed.Group("/me/images", middleware.RequirePerm(rbac.PermSelfImage))
				{
					ig.GET("/tasks", d.MeImageH.List)
					ig.GET("/tasks/:id", d.MeImageH.Get)
				}
			}
			if d.AdminModelH != nil {
				// 普通用户视角的 enabled 模型列表(用于面板下拉)
				authed.GET("/me/models", d.AdminModelH.ListEnabledForMe)
			}

			// ---- 在线体验:复用 /v1 handler,但入口是 JWT 鉴权 + 自动映射内部 key ----
			if d.GatewayH != nil && d.KeySvc != nil {
				pg := authed.Group("/me/playground", gateway.JWTAsPlaygroundKey(d.KeySvc))
				{
					pg.POST("/chat", d.GatewayH.ChatCompletions)
					if d.ImagesH != nil {
						pg.POST("/image", gateway.PlaygroundImagePreflight(), d.ImagesH.ImageGenerations)
						pg.POST("/image-edit", gateway.PlaygroundImagePreflight(), d.ImagesH.ImageEdits)
					}
				}
			}
		}

		// 公开接口(无需 JWT)
		pub := api.Group("/public")
		if d.SettingsH != nil {
			pub.GET("/site-info", d.SettingsH.Public)
		}
		if d.RechargeH != nil {
			pub.POST("/epay/notify", d.RechargeH.EPayNotify)
			pub.GET("/epay/notify", d.RechargeH.EPayNotify)
		}

		// admin 全组强制 RequireAdmin;所有写操作再通过 audit.Middleware 自动落审计。
		adminMW := []gin.HandlerFunc{
			middleware.JWTAuth(d.JWT),
			middleware.RequireAdmin(),
		}
		if d.AuditDAO != nil {
			adminMW = append(adminMW, audit.Middleware(d.AuditDAO))
		}
		admin := api.Group("/admin", adminMW...)
		{
			admin.GET("/ping", func(c *gin.Context) { resp.OK(c, gin.H{"msg": "admin pong"}) })

			// 代理池
			pg := admin.Group("/proxies", middleware.RequirePerm(rbac.PermProxyRead, rbac.PermProxyWrite))
			{
				pg.POST("", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Create)
				pg.POST("/import", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Import)
				pg.POST("/probe-all", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.ProbeAll)
				pg.GET("", d.ProxyH.List)
				pg.GET("/:id", d.ProxyH.Get)
				pg.POST("/:id/probe", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Probe)
				pg.PATCH("/:id", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Update)
				pg.DELETE("/:id", middleware.RequirePerm(rbac.PermProxyWrite), d.ProxyH.Delete)
			}

			// 账号池
			ag := admin.Group("/accounts", middleware.RequirePerm(rbac.PermAccountRead, rbac.PermAccountWrite))
			{
				ag.POST("", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Create)
				ag.POST("/import", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Import)
				ag.POST("/import-tokens", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ImportTokens)
				ag.POST("/refresh-all", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.RefreshAll)
				ag.POST("/probe-quota-all", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ProbeQuotaAll)
				ag.POST("/bulk-delete", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.BulkDelete)
				ag.GET("/auto-refresh", d.AccountH.GetAutoRefresh)
				ag.PUT("/auto-refresh", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.SetAutoRefresh)
				ag.GET("/quota-summary", d.AccountH.QuotaSummary)
				ag.GET("", d.AccountH.List)
				ag.GET("/:id", d.AccountH.Get)
				ag.GET("/:id/secrets", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.GetSecrets)
				ag.PATCH("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Update)
				ag.DELETE("/:id", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Delete)
				ag.POST("/:id/refresh", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.Refresh)
				ag.POST("/:id/probe-quota", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.ProbeQuota)
				ag.POST("/:id/bind-proxy", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.BindProxy)
				ag.DELETE("/:id/bind-proxy", middleware.RequirePerm(rbac.PermAccountWrite), d.AccountH.UnbindProxy)
			}

			// ---- 用户管理 ----
			if d.AdminUserH != nil {
				ug := admin.Group("/users", middleware.RequirePerm(rbac.PermUserRead, rbac.PermUserWrite))
				{
					ug.GET("", d.AdminUserH.List)
					ug.POST("", middleware.RequirePerm(rbac.PermUserWrite), d.AdminUserH.Create)
					ug.GET("/:id", d.AdminUserH.Get)
					ug.PATCH("/:id", middleware.RequirePerm(rbac.PermUserWrite), d.AdminUserH.Update)
					ug.POST("/:id/reset-password",
						middleware.RequirePerm(rbac.PermUserWrite), d.AdminUserH.ResetPassword)
					ug.DELETE("/:id", middleware.RequirePerm(rbac.PermUserWrite), d.AdminUserH.Delete)
					// 积分调账
					ug.POST("/:id/credits/adjust",
						middleware.RequirePerm(rbac.PermUserCredit), d.AdminUserH.Adjust)
					ug.GET("/:id/credit-logs",
						middleware.RequirePerm(rbac.PermUsageReadAll), d.AdminUserH.CreditLogs)
				}

				// ---- 积分管理(全局视图) ----
				cg := admin.Group("/credits", middleware.RequirePerm(rbac.PermUserCredit))
				{
					cg.GET("/summary", d.AdminUserH.CreditsSummary)
					cg.GET("/logs", d.AdminUserH.CreditLogsGlobal)
					cg.POST("/adjust", d.AdminUserH.AdjustByUser)
				}
			}

			// ---- 用户分组 ----
			if d.AdminGroupH != nil {
				gg := admin.Group("/groups", middleware.RequirePerm(rbac.PermGroupWrite))
				{
					gg.GET("", d.AdminGroupH.List)
					gg.POST("", d.AdminGroupH.Create)
					gg.PUT("/:id", d.AdminGroupH.Update)
					gg.DELETE("/:id", d.AdminGroupH.Delete)
				}
			}

			// 审计日志只读
			if d.AuditH != nil {
				auditG := admin.Group("/audit", middleware.RequirePerm(rbac.PermAuditRead))
				auditG.GET("/logs", d.AuditH.List)
			}

			// 生成记录(管理员全局视图)
			if d.AdminImageH != nil {
				admin.GET("/image-tasks", middleware.RequirePerm(rbac.PermUsageReadAll), d.AdminImageH.List)
			}

			// ---- 上游渠道(OpenAI/Gemini 兼容) ----
			if d.ChannelH != nil {
				cg := admin.Group("/channels",
					middleware.RequirePerm(rbac.PermChannelRead, rbac.PermChannelWrite))
				{
					cg.GET("", d.ChannelH.List)
					cg.POST("", middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.Create)
					cg.GET("/:id", d.ChannelH.Get)
					cg.PATCH("/:id", middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.Update)
					cg.DELETE("/:id", middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.Delete)
					cg.POST("/:id/test", middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.Test)
					cg.GET("/:id/mappings", d.ChannelH.ListMappings)
					cg.POST("/:id/mappings",
						middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.CreateMapping)
				}
				// 映射单条操作:另起一组,避免和 /channels/:id 冲突。
				mg := admin.Group("/channel-mappings",
					middleware.RequirePerm(rbac.PermChannelRead, rbac.PermChannelWrite))
				{
					mg.PATCH("/:mid",
						middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.UpdateMapping)
					mg.DELETE("/:mid",
						middleware.RequirePerm(rbac.PermChannelWrite), d.ChannelH.DeleteMapping)
				}
			}

			// ---- 模型配置 ----
			if d.AdminModelH != nil {
				mg := admin.Group("/models",
					middleware.RequirePerm(rbac.PermModelRead, rbac.PermModelWrite))
				{
					mg.GET("", d.AdminModelH.List)
					// 下面的写路径额外加一层 model:write,防读权限账号写入
					mg.POST("",
						middleware.RequirePerm(rbac.PermModelWrite),
						d.AdminModelH.Create)
					mg.PUT("/:id",
						middleware.RequirePerm(rbac.PermModelWrite),
						d.AdminModelH.Update)
					mg.PATCH("/:id/enabled",
						middleware.RequirePerm(rbac.PermModelWrite),
						d.AdminModelH.SetEnabled)
					mg.DELETE("/:id",
						middleware.RequirePerm(rbac.PermModelWrite),
						d.AdminModelH.Delete)
				}
			}

			// ---- 全局 API Keys(跨用户) ----
			if d.AdminKeyH != nil {
				kg := admin.Group("/keys", middleware.RequirePerm(rbac.PermKeyReadAll, rbac.PermKeyWriteAll))
				{
					kg.GET("", d.AdminKeyH.List)
					kg.PATCH("/:id", middleware.RequirePerm(rbac.PermKeyWriteAll), d.AdminKeyH.SetEnabled)
				}
			}

			// ---- 用量聚合 / 日志 ----
			if d.AdminUsageH != nil {
				ug := admin.Group("/usage", middleware.RequirePerm(rbac.PermUsageReadAll, rbac.PermStatsReadAll))
				{
					ug.GET("/stats", d.AdminUsageH.Stats)
					ug.GET("/logs", d.AdminUsageH.Logs)
				}
			}

			// ---- 充值套餐 + 订单 ----
			if d.AdminRechargeH != nil {
				rg := admin.Group("/recharge", middleware.RequirePerm(rbac.PermRechargeManage))
				{
					rg.GET("/packages", d.AdminRechargeH.ListPackages)
					rg.POST("/packages", d.AdminRechargeH.CreatePackage)
					rg.PATCH("/packages/:id", d.AdminRechargeH.UpdatePackage)
					rg.DELETE("/packages/:id", d.AdminRechargeH.DeletePackage)
					rg.GET("/orders", d.AdminRechargeH.ListOrders)
					rg.POST("/orders/:id/force-paid", d.AdminRechargeH.ForcePaid)
					rg.GET("/codes", d.AdminRechargeH.ListRedeemCodes)
					rg.POST("/codes/batch", d.AdminRechargeH.GenerateRedeemCodes)
					rg.DELETE("/codes/:id", d.AdminRechargeH.DeleteRedeemCode)
				}
			}

			// 系统设置(站点 / 注册 / SMTP 测试 等)
			if d.SettingsH != nil {
				sg := admin.Group("/settings", middleware.RequirePerm(rbac.PermSystemSetting))
				{
					sg.GET("", d.SettingsH.List)
					sg.PUT("", d.SettingsH.Update)
					sg.POST("/reload", d.SettingsH.Reload)
					sg.POST("/test-email", d.SettingsH.TestMail)
				}
			}

			// 数据库备份/恢复(超高危,细粒度权限 + handler 内二次密码)
			if d.BackupH != nil {
				bg := admin.Group("/system/backup", middleware.RequirePerm(rbac.PermSystemBackup))
				{
					bg.GET("", d.BackupH.List)
					bg.POST("", d.BackupH.Create)
					bg.GET("/:id/download", d.BackupH.Download)
					bg.DELETE("/:id", d.BackupH.Delete)
					bg.POST("/:id/restore", d.BackupH.Restore)
					bg.POST("/upload", d.BackupH.Upload)
				}
			}
		}
	}

	// ---- OpenAI 兼容网关(API Key) ----
	v1 := r.Group("/v1", apikey.Middleware(d.KeySvc, false))
	{
		v1.GET("/models", d.GatewayH.ListModels)
		v1.POST("/chat/completions", d.GatewayH.ChatCompletions)

		if d.ImagesH != nil {
			v1.POST("/images/generations", d.ImagesH.ImageGenerations)
			v1.POST("/images/edits", d.ImagesH.ImageEdits)
			v1.GET("/images/tasks/:id", d.ImagesH.ImageTask)
		}
	}

	// ---- 图片代理(签名 URL,无需 API Key,对 <img src> 友好)----
	if d.ImagesH != nil {
		r.GET("/p/img/:task_id/:idx", d.ImagesH.ImageProxy)
	}

	// ---- 前端 SPA(可选;找不到 dist 就跳过) ----
	mountSPA(r)

	return r
}
