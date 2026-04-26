package recharge

import "time"

// RedeemCode 对应 redeem_codes 表。
type RedeemCode struct {
	ID        uint64     `db:"id" json:"id"`
	Code      string     `db:"code" json:"code"`
	Credits   int64      `db:"credits" json:"credits"`
	BatchNo   string     `db:"batch_no" json:"batch_no"`
	Remark    string     `db:"remark" json:"remark"`
	CreatedBy uint64     `db:"created_by" json:"created_by"`
	UsedBy    *uint64    `db:"used_by" json:"used_by,omitempty"`
	UsedAt    *time.Time `db:"used_at" json:"used_at,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"-"`
}

func (r *RedeemCode) IsUsed() bool { return r != nil && r.UsedBy != nil }

type RedeemCodeListFilter struct {
	Status  string
	BatchNo string
	Code    string
}