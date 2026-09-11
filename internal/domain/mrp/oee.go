package mrp

import (
	"time"

	platformerrors "cashflow_backend/internal/platform/errors"
)

type LossType string

const (
	LossTypeProductive   LossType = "productive"
	LossTypePerformance  LossType = "performance"
	LossTypeAvailability LossType = "availability"
	LossTypeQuality      LossType = "quality"
)

type WorkcenterProductivity struct {
	ID           int64      `json:"id"`
	WorkcenterID int64      `json:"workcenter_id"`
	WorkorderID  *int64     `json:"workorder_id,omitempty"`
	LossID       int64      `json:"loss_id"`
	LossType     LossType   `json:"loss_type"`
	DateStart    time.Time  `json:"date_start"`
	DateEnd      *time.Time `json:"date_end,omitempty"`
	Duration     float64    `json:"duration"`
	Description  string     `json:"description,omitempty"`
	CompanyID    int64      `json:"company_id"`
}

type ProductivityLoss struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	LossType  LossType `json:"loss_type"`
	CompanyID int64    `json:"company_id"`
	Active    bool     `json:"active"`
}

type OEEMetrics struct {
	WorkcenterID int64     `json:"workcenter_id"`
	DateFrom     time.Time `json:"date_from"`
	DateTo       time.Time `json:"date_to"`
	Availability float64   `json:"availability"`
	Performance  float64   `json:"performance"`
	Quality      float64   `json:"quality"`
	OEE          float64   `json:"oee"`
}

func CalculateOEE(workcenterID int64, from, to time.Time, plannedMinutes, idealMinutes, produced, rejected float64, logs []WorkcenterProductivity) (OEEMetrics, error) {
	if workcenterID <= 0 || !to.After(from) || plannedMinutes <= 0 || idealMinutes < 0 || produced < 0 || rejected < 0 || rejected > produced {
		return OEEMetrics{}, platformerrors.Validation("invalid OEE inputs", nil)
	}
	productiveMinutes := 0.0
	for _, log := range logs {
		if log.WorkcenterID == workcenterID && log.LossType == LossTypeProductive && log.Duration > 0 {
			productiveMinutes += log.Duration
		}
	}
	availability := productiveMinutes / plannedMinutes * 100
	if availability > 100 {
		availability = 100
	}
	performance := 0.0
	if productiveMinutes > 0 {
		performance = idealMinutes * produced / productiveMinutes * 100
	}
	if performance > 100 {
		performance = 100
	}
	quality := 100.0
	if produced > 0 {
		quality = (produced - rejected) / produced * 100
	}
	return OEEMetrics{WorkcenterID: workcenterID, DateFrom: from, DateTo: to, Availability: availability, Performance: performance, Quality: quality, OEE: availability * performance * quality / 10000}, nil
}
