package loyaltyhttp

import (
	"time"

	"cashflow_backend/internal/domain/loyalty"
	loyaltyusecase "cashflow_backend/internal/usecase/loyalty"
)

// ─────────────────────────────────────────────────────────────────────────────
// Programs
// ─────────────────────────────────────────────────────────────────────────────

type CreateProgramRequest struct {
	Name            string                  `json:"name"`
	Active          *bool                   `json:"active"`
	Sequence        *int                    `json:"sequence"`
	CompanyID       *int64                  `json:"company_id"`
	Currency        string                  `json:"currency"`
	PricelistIDs    []int64                 `json:"pricelist_ids"`
	ProgramType     loyalty.ProgramType     `json:"program_type"`
	DateFrom        *time.Time              `json:"date_from"`
	DateTo          *time.Time              `json:"date_to"`
	LimitUsage      bool                    `json:"limit_usage"`
	MaxUsage        int                     `json:"max_usage"`
	AppliesOn       loyalty.AppliesOn       `json:"applies_on"`
	Trigger         loyalty.Trigger         `json:"trigger"`
	PortalVisible   bool                    `json:"portal_visible"`
	PortalPointName string                  `json:"portal_point_name"`
	Rules           []loyalty.LoyaltyRule   `json:"rules"`
	Rewards         []loyalty.LoyaltyReward `json:"rewards"`
	Mails           []loyalty.LoyaltyMail   `json:"mails"`
}

func (req *CreateProgramRequest) ToInput() loyaltyusecase.CreateProgramInput {
	return loyaltyusecase.CreateProgramInput{
		Name:            req.Name,
		Sequence:        derefInt(req.Sequence),
		CompanyID:       req.CompanyID,
		Currency:        req.Currency,
		PricelistIDs:    req.PricelistIDs,
		ProgramType:     req.ProgramType,
		DateFrom:        req.DateFrom,
		DateTo:          req.DateTo,
		LimitUsage:      req.LimitUsage,
		MaxUsage:        req.MaxUsage,
		AppliesOn:       req.AppliesOn,
		Trigger:         req.Trigger,
		PortalVisible:   req.PortalVisible,
		PortalPointName: req.PortalPointName,
		Rules:           req.Rules,
		Rewards:         req.Rewards,
		Mails:           req.Mails,
	}
}

type UpdateProgramRequest struct {
	Name            *string                `json:"name"`
	Active          *bool                  `json:"active"`
	Sequence        *int                   `json:"sequence"`
	Currency        *string                `json:"currency"`
	PricelistIDs    *[]int64               `json:"pricelist_ids"`
	DateFrom        *time.Time             `json:"date_from"`
	DateTo          *time.Time             `json:"date_to"`
	LimitUsage      *bool                  `json:"limit_usage"`
	MaxUsage        *int                   `json:"max_usage"`
	AppliesOn       *loyalty.AppliesOn     `json:"applies_on"`
	Trigger         *loyalty.Trigger       `json:"trigger"`
	PortalVisible   *bool                  `json:"portal_visible"`
	PortalPointName *string                `json:"portal_point_name"`
	Rules           []loyalty.LoyaltyRule  `json:"rules"`
	Rewards         []loyalty.LoyaltyReward `json:"rewards"`
	Mails           []loyalty.LoyaltyMail  `json:"mails"`
}

func (req *UpdateProgramRequest) ToInput() loyaltyusecase.UpdateProgramInput {
	in := loyaltyusecase.UpdateProgramInput{
		Name:            req.Name,
		Active:          req.Active,
		Sequence:        req.Sequence,
		Currency:        req.Currency,
		DateFrom:        req.DateFrom,
		DateTo:          req.DateTo,
		LimitUsage:      req.LimitUsage,
		MaxUsage:        req.MaxUsage,
		AppliesOn:       req.AppliesOn,
		Trigger:         req.Trigger,
		PortalVisible:   req.PortalVisible,
		PortalPointName: req.PortalPointName,
		Rules:           req.Rules,
		Rewards:         req.Rewards,
		Mails:           req.Mails,
	}
	if req.PricelistIDs != nil {
		in.PricelistIDs = *req.PricelistIDs
	}
	return in
}

type SetProgramTypeRequest struct {
	ProgramType loyalty.ProgramType `json:"program_type"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Cards
// ─────────────────────────────────────────────────────────────────────────────

type GenerateCardsRequest struct {
	ProgramID      int64      `json:"program_id"`
	PartnerID      *int64     `json:"partner_id"`
	ExpirationDate *time.Time `json:"expiration_date"`
	Count          int        `json:"count"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Order operations
// ─────────────────────────────────────────────────────────────────────────────

type ClaimCouponRequest struct {
	Code string `json:"code"`
}

type ApplyCodeRequest struct {
	RuleID int64 `json:"rule_id"`
}

type RedeemCouponRequest struct {
	Code     string  `json:"code"`
	RewardID int64   `json:"reward_id"`
	Points   float64 `json:"points"`
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}