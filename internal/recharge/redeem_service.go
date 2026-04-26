package recharge

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"

	"github.com/432539/gpt2api/internal/billing"
)

var (
	ErrRedeemCodeUsed         = errors.New("recharge: redeem code already used")
	ErrRedeemLimitReached     = errors.New("recharge: user already redeemed")
	ErrRedeemCodeDeleted      = errors.New("recharge: redeem code deleted")
	ErrRedeemCodeInvalid      = errors.New("recharge: redeem code invalid")
	ErrRedeemCodeNotDeletable = errors.New("recharge: used redeem code cannot be deleted")
)

type RedeemBatchCreateInput struct {
	Count     int
	Credits   int64
	Prefix    string
	Remark    string
	CreatedBy uint64
}

func (s *Service) AdminGenerateRedeemCodes(ctx context.Context, in RedeemBatchCreateInput) ([]RedeemCode, string, error) {
	if in.Count <= 0 {
		return nil, "", errors.New("count must be positive")
	}
	if in.Count > 1000 {
		return nil, "", errors.New("count too large")
	}
	if in.Credits <= 0 {
		return nil, "", errors.New("credits must be positive")
	}
	batchNo := genRedeemBatchNo()
	prefix := sanitizeRedeemPrefix(in.Prefix)
	items := make([]RedeemCode, 0, in.Count)
	for len(items) < in.Count {
		code := genRedeemCode(prefix)
		row := RedeemCode{
			Code:      code,
			Credits:   in.Credits,
			BatchNo:   batchNo,
			Remark:    strings.TrimSpace(in.Remark),
			CreatedBy: in.CreatedBy,
		}
		res, err := s.dao.DB().ExecContext(ctx,
			`INSERT INTO redeem_codes (code, credits, batch_no, remark, created_by) VALUES (?, ?, ?, ?, ?)`,
			row.Code, row.Credits, row.BatchNo, row.Remark, row.CreatedBy)
		if err != nil {
			if isDuplicateKey(err) {
				continue
			}
			return nil, "", err
		}
		id, _ := res.LastInsertId()
		row.ID = uint64(id)
		items = append(items, row)
	}
	return items, batchNo, nil
}

func (s *Service) AdminListRedeemCodes(ctx context.Context, f RedeemCodeListFilter, offset, limit int) ([]RedeemCode, int64, error) {
	return s.dao.ListRedeemCodes(ctx, f, offset, limit)
}

func (s *Service) AdminDeleteRedeemCode(ctx context.Context, id uint64) error {
	row, err := s.dao.GetRedeemCodeByID(ctx, id)
	if err != nil {
		return err
	}
	if row.IsUsed() {
		return ErrRedeemCodeNotDeletable
	}
	return s.dao.SoftDeleteRedeemCode(ctx, id)
}

func (s *Service) GetMyRedeemStatus(ctx context.Context, userID uint64) (*RedeemCode, error) {
	return s.dao.GetUserRedeemRecord(ctx, userID)
}

func (s *Service) RedeemCode(ctx context.Context, userID uint64, rawCode string) (*RedeemCode, error) {
	code := normalizeRedeemCode(rawCode)
	if code == "" {
		return nil, ErrRedeemCodeInvalid
	}
	tx, err := s.dao.DB().BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	var row RedeemCode
	err = tx.GetContext(ctx, &row,
		`SELECT * FROM redeem_codes WHERE code = ? AND deleted_at IS NULL FOR UPDATE`, code)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return nil, ErrRedeemCodeInvalid
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if row.IsUsed() {
		_ = tx.Rollback()
		return nil, ErrRedeemCodeUsed
	}
	var existed uint64
	err = tx.GetContext(ctx, &existed,
		`SELECT id FROM redeem_codes WHERE used_by = ? AND deleted_at IS NULL LIMIT 1`, userID)
	if err == nil {
		_ = tx.Rollback()
		return nil, ErrRedeemLimitReached
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE redeem_codes SET used_by = ?, used_at = NOW() WHERE id = ? AND used_by IS NULL`,
		userID, row.ID); err != nil {
		if isDuplicateKey(err) {
			_ = tx.Rollback()
			return nil, ErrRedeemLimitReached
		}
		_ = tx.Rollback()
		return nil, err
	}
	if _, err = tx.ExecContext(ctx,
		`UPDATE users SET credit_balance = credit_balance + ?, version = version + 1 WHERE id = ? AND deleted_at IS NULL`,
		row.Credits, userID); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	var balanceAfter int64
	if err = tx.QueryRowxContext(ctx,
		`SELECT credit_balance FROM users WHERE id = ?`, userID).Scan(&balanceAfter); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	refID := fmt.Sprintf("redeem:%s", row.Code)
	remark := fmt.Sprintf("兑换码兑换:%s", row.Code)
	if strings.TrimSpace(row.Remark) != "" {
		remark = fmt.Sprintf("兑换码兑换:%s (%s)", row.Code, strings.TrimSpace(row.Remark))
	}
	if _, err = tx.ExecContext(ctx,
		`INSERT INTO credit_transactions (user_id, key_id, type, amount, balance_after, ref_id, remark)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, 0, billing.KindRedeem, row.Credits, balanceAfter, refID, remark); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		if isDuplicateKey(err) {
			return nil, ErrRedeemLimitReached
		}
		return nil, err
	}
	now := time.Now()
	row.UsedBy = &userID
	row.UsedAt = &now
	return &row, nil
}

func normalizeRedeemCode(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func sanitizeRedeemPrefix(s string) string {
	s = normalizeRedeemCode(s)
	s = strings.Trim(s, "-")
	if s == "" {
		return "RC"
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

func genRedeemBatchNo() string {
	return time.Now().UTC().Format("20060102150405")
}

func genRedeemCode(prefix string) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 12)
	raw := make([]byte, len(buf))
	_, _ = rand.Read(raw)
	for i := range buf {
		buf[i] = alphabet[int(raw[i])%len(alphabet)]
	}
	body := string(buf)
	return fmt.Sprintf("%s-%s-%s-%s", prefix, body[:4], body[4:8], body[8:12])
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}