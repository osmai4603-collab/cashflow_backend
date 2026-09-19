package domain

import (
	"errors"
	"fmt"
	"time"
)

// أخطاء النطاق النقية (Sentinel Domain Errors)
var (
	ErrInsufficientBalance = errors.New("insufficient balance for transaction")
	ErrInvalidAmount       = errors.New("transaction amount must be positive")
	ErrWalletNotFound      = errors.New("wallet not found")
	ErrSameAccountTransfer = errors.New("cannot transfer funds to the same wallet")
)

// Wallet يمثل كيان النطاق النقي (Pure Domain Entity)
// لا يحتوي على أي وسوم خاصة بقواعد البيانات أو أطر عمل الويب
type Wallet struct {
	ID        string
	OwnerID   string
	Balance   int64 // يتم تخزين المبالغ بالوحدات الصغرى (Cents) لمنع مشاكل الفاصلة العائمة
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewWallet منشئ كيان المحفظة مع التحقق من صحة القواعد
func NewWallet(id, ownerID, currency string) (*Wallet, error) {
	if id == "" || ownerID == "" {
		return nil, errors.New("wallet id and owner id cannot be empty")
	}
	if currency == "" {
		currency = "USD"
	}

	now := time.Now().UTC()
	return &Wallet{
		ID:        id,
		OwnerID:   ownerID,
		Balance:   0,
		Currency:  currency,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// Debit خصم مبلغ من رصيد المحفظة مع التحقق من كفاية الرصيد
func (w *Wallet) Debit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if w.Balance < amount {
		return fmt.Errorf("%w: available %d, required %d", ErrInsufficientBalance, w.Balance, amount)
	}

	w.Balance -= amount
	w.UpdatedAt = time.Now().UTC()
	return nil
}

// Credit إضافة مبلغ إلى رصيد المحفظة
func (w *Wallet) Credit(amount int64) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}

	w.Balance += amount
	w.UpdatedAt = time.Now().UTC()
	return nil
}
