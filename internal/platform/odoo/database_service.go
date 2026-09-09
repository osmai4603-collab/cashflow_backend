package odoo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type DatabaseService struct {
	baseURL      string
	masterPassword string
	client       *http.Client
}

type CreateDatabaseInput struct {
	MasterPassword string
	Name           string
	Language       string
	Password       string
	Login          string
	Phone          string
	CountryCode    string
	Demo           bool
}

func NewDatabaseService(baseURL, masterPassword string, timeout time.Duration) *DatabaseService {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &DatabaseService{
		baseURL:        strings.TrimRight(baseURL, "/"),
		masterPassword: masterPassword,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (s *DatabaseService) List(ctx context.Context) ([]string, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "call",
		"params":  map[string]any{},
		"id":      1,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode database list request: %w", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.endpoint("/web/database/list"),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("build database list request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Odoo database list: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("Odoo database list returned status %d", response.StatusCode)
	}

	var result struct {
		Result []string `json:"result"`
		Error  any      `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode Odoo database list: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("Odoo database list returned an error")
	}
	return result.Result, nil
}

func (s *DatabaseService) Create(ctx context.Context, input CreateDatabaseInput) error {
	masterPassword := input.MasterPassword
	if masterPassword == "" {
		masterPassword = s.masterPassword
	}
	form := url.Values{
		"master_pwd":   {masterPassword},
		"name":         {input.Name},
		"lang":         {input.Language},
		"password":     {input.Password},
		"login":        {input.Login},
		"phone":        {input.Phone},
		"country_code": {input.CountryCode},
	}
	if input.Demo {
		form.Set("demo", "1")
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.endpoint("/web/database/create"),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("build database create request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := s.client.Do(request)
	if err != nil {
		return fmt.Errorf("request Odoo database create: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices && response.StatusCode < 400 {
		return nil
	}

	_, _ = io.Copy(io.Discard, response.Body)
	return fmt.Errorf("Odoo database create returned status %d", response.StatusCode)
}

func (s *DatabaseService) endpoint(path string) string {
	return s.baseURL + path
}