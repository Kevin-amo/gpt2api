package recharge

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/432539/gpt2api/internal/middleware"
	"github.com/432539/gpt2api/pkg/resp"
)

// GET /api/recharge/redeem-status
func (h *Handler) RedeemStatus(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		resp.Unauthorized(c, "unauthorized")
		return
	}
	row, err := h.svc.GetMyRedeemStatus(c.Request.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.OK(c, gin.H{"redeemed": false})
			return
		}
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"redeemed": true, "record": row})
}

// POST /api/recharge/redeem
func (h *Handler) Redeem(c *gin.Context) {
	uid := middleware.UserID(c)
	if uid == 0 {
		resp.Unauthorized(c, "unauthorized")
		return
	}
	var req struct {
		Code string `json:"code" binding:"required,max=64"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	row, err := h.svc.RedeemCode(c.Request.Context(), uid, req.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrRedeemCodeInvalid):
			resp.BadRequest(c, "兑换码不存在或已失效")
		case errors.Is(err, ErrRedeemCodeUsed):
			resp.Conflict(c, "兑换码已被使用")
		case errors.Is(err, ErrRedeemLimitReached):
			resp.Conflict(c, "每个用户仅可兑换一次")
		default:
			resp.Internal(c, err.Error())
		}
		return
	}
	resp.OK(c, row)
}

// GET /api/admin/recharge/codes
func (h *AdminHandler) ListRedeemCodes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	rows, total, err := h.svc.AdminListRedeemCodes(c.Request.Context(), RedeemCodeListFilter{
		Status:  c.Query("status"),
		BatchNo: c.Query("batch_no"),
		Code:    c.Query("code"),
	}, offset, limit)
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": rows, "total": total, "limit": limit, "offset": offset})
}

// POST /api/admin/recharge/codes/batch
func (h *AdminHandler) GenerateRedeemCodes(c *gin.Context) {
	actorID := middleware.UserID(c)
	var req struct {
		Count   int    `json:"count" binding:"required,min=1,max=1000"`
		Credits int64  `json:"credits" binding:"required,min=1"`
		Prefix  string `json:"prefix" binding:"max=8"`
		Remark  string `json:"remark" binding:"max=255"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	items, batchNo, err := h.svc.AdminGenerateRedeemCodes(c.Request.Context(), RedeemBatchCreateInput{
		Count:     req.Count,
		Credits:   req.Credits,
		Prefix:    req.Prefix,
		Remark:    req.Remark,
		CreatedBy: actorID,
	})
	if err != nil {
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"items": items, "batch_no": batchNo, "total": len(items)})
}

// DELETE /api/admin/recharge/codes/:id
func (h *AdminHandler) DeleteRedeemCode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.svc.AdminDeleteRedeemCode(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			resp.NotFound(c, "兑换码不存在")
			return
		}
		if errors.Is(err, ErrRedeemCodeNotDeletable) {
			resp.Conflict(c, "已使用的兑换码不可删除")
			return
		}
		resp.Internal(c, err.Error())
		return
	}
	resp.OK(c, gin.H{"ok": true})
}