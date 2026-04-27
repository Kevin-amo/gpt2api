// Package gateway - playground 子模块。
//
// "在线体验" 面板让登录用户用浏览器里的 JWT 直接跑 chat/image。
// 为了零改动复用现有的 /v1/chat/completions 和 /v1/images/generations
// 业务链路(模型白名单 / 倍率 / RPM/TPM / 预扣-结算 / usage_logs / image_tasks),
// 我们做一个小中间件,把 JWT 里的 user_id 映射到一把内部
// "__playground__" APIKey,塞进 apikey.Ctx,之后原封不动调用现有 handler。
//
// 这把 key 永远不暴露给用户、不会出现在 API Keys 列表,也不计入 quota。
// 所有的扣费仍然记在对应 user_id 的积分钱包上,审计可追溯。
package gateway

import (
	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/apikey"
	"github.com/432539/gpt2api/internal/middleware"
	"github.com/432539/gpt2api/pkg/resp"
)

// JWTAsPlaygroundKey 是一个 gin 中间件。
// 必须挂在 middleware.JWTAuth 之后,它会按 JWT 的 user_id 懒加载(没有就创建)
// 一把内部 playground key,塞进 apikey.Ctx,供下游 handler 直接取用。
func JWTAsPlaygroundKey(svc *apikey.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := middleware.UserID(c)
		if uid == 0 {
			resp.Unauthorized(c, "not logged in")
			c.Abort()
			return
		}
		k, err := svc.EnsureInternalKey(c.Request.Context(), uid)
		if err != nil {
			resp.Internal(c, err.Error())
			c.Abort()
			return
		}
		c.Set(apikey.CtxKey, k)
		c.Set(apikey.CtxKeyOwner, k.UserID)
		c.Next()
	}
}

// PlaygroundForceSingleImage 仅对在线体验的文生图入口生效。
// 公开 /v1/images/generations 仍保留多张能力,避免误伤外部 API 客户端。
func PlaygroundForceSingleImage() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("playground.force_single_image", true)
		c.Next()
	}
}

func shouldForceSinglePlaygroundImage(c *gin.Context) bool {
	v, ok := c.Get("playground.force_single_image")
	if !ok {
		return false
	}
	force, _ := v.(bool)
	return force
}

// PlaygroundImagePreflight 原先在图生图未实现时用于拦截带 reference_images 的请求。
// 现在 chatgpt 上游协议已接通(见 upstream/chatgpt/files.go + StreamFConversation 的
// multimodal_text 支持),这里恢复为 pass-through,仅保留名字避免上层引用破坏。
//
// 保留的唯一职责:未登录时快速失败,给错误 JSON,好过透传到下游 handler 再发一遍 401。
func PlaygroundImagePreflight() gin.HandlerFunc {
	return func(c *gin.Context) {
		if middleware.UserID(c) == 0 {
			resp.Unauthorized(c, "not logged in")
			c.Abort()
			return
		}
		c.Next()
	}
}
