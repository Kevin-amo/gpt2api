package recharge

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func (d *DAO) GetRedeemCodeByID(ctx context.Context, id uint64) (*RedeemCode, error) {
	var row RedeemCode
	err := d.db.GetContext(ctx, &row,
		`SELECT * FROM redeem_codes WHERE id = ? AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (d *DAO) GetRedeemCodeByCode(ctx context.Context, code string) (*RedeemCode, error) {
	var row RedeemCode
	err := d.db.GetContext(ctx, &row,
		`SELECT * FROM redeem_codes WHERE code = ? AND deleted_at IS NULL`, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (d *DAO) GetUserRedeemRecord(ctx context.Context, userID uint64) (*RedeemCode, error) {
	var row RedeemCode
	err := d.db.GetContext(ctx, &row,
		`SELECT * FROM redeem_codes WHERE used_by = ? AND deleted_at IS NULL ORDER BY id DESC LIMIT 1`, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &row, err
}

func (d *DAO) ListRedeemCodes(ctx context.Context, f RedeemCodeListFilter, offset, limit int) ([]RedeemCode, int64, error) {
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	where := []string{"deleted_at IS NULL"}
	args := []any{}
	if f.Status == "used" {
		where = append(where, "used_by IS NOT NULL")
	} else if f.Status == "unused" {
		where = append(where, "used_by IS NULL")
	}
	if s := strings.TrimSpace(f.BatchNo); s != "" {
		where = append(where, "batch_no = ?")
		args = append(args, s)
	}
	if s := strings.TrimSpace(f.Code); s != "" {
		where = append(where, "code LIKE ?")
		args = append(args, "%"+s+"%")
	}
	ws := strings.Join(where, " AND ")
	rows := make([]RedeemCode, 0, limit)
	if err := d.db.SelectContext(ctx, &rows,
		fmt.Sprintf(`SELECT * FROM redeem_codes WHERE %s ORDER BY id DESC LIMIT ? OFFSET ?`, ws),
		append(args, limit, offset)...); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := d.db.GetContext(ctx, &total,
		fmt.Sprintf(`SELECT COUNT(*) FROM redeem_codes WHERE %s`, ws), args...); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (d *DAO) SoftDeleteRedeemCode(ctx context.Context, id uint64) error {
	_, err := d.db.ExecContext(ctx,
		`UPDATE redeem_codes SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL`, id)
	return err
}
