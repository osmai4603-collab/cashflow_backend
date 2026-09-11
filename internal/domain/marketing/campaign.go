package marketing

import (
	"errors"
	"time"
)

type CampaignState string

const (
	CampaignStateDraft     CampaignState = "draft"
	CampaignStateScheduled CampaignState = "scheduled"
	CampaignStateRunning   CampaignState = "running"
	CampaignStateCompleted CampaignState = "completed"
	CampaignStateArchived  CampaignState = "archived"
)

type Campaign struct {
	ID             int64         `json:"id"`
	Name           string        `json:"name"`
	UserID         int64         `json:"user_id"`
	UTMSource      string        `json:"utm_source"`
	UTMMedium      string        `json:"utm_medium"`
	UTMCampaign    string        `json:"utm_campaign"`
	TotalSent      int           `json:"total_sent"`
	TotalDelivered int           `json:"total_delivered"`
	TotalOpened    int           `json:"total_opened"`
	TotalClicked   int           `json:"total_clicked"`
	TotalBounced   int           `json:"total_bounced"`
	TotalRevenue   float64       `json:"total_revenue"`
	State          CampaignState `json:"state"`
	CompanyID      int64         `json:"company_id"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (c *Campaign) Validate() error {
	if c.Name == "" {
		return errors.New("campaign name is required")
	}
	if c.CompanyID == 0 {
		return errors.New("company id is required")
	}
	return nil
}
