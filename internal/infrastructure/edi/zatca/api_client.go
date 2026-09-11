package zatca

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ZATCASandboxURL    = "https://gw-fatoora.zatca.gov.sa/e-invoicing/developer-portal"
	ZATCAProductionURL = "https://gw-fatoora.zatca.gov.sa/e-invoicing/core"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	authToken  string
}

func NewClient(baseURL, authToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: baseURL, authToken: authToken, httpClient: httpClient}
}

type CSIDResponse struct {
	RequestID           string `json:"requestID"`
	BinarySecurityToken string `json:"binarySecurityToken"`
	Secret              string `json:"secret"`
}
type ValidationMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ValidationResults struct {
	Status   string              `json:"status"`
	Warnings []ValidationMessage `json:"warningMessages"`
	Errors   []ValidationMessage `json:"errorMessages"`
	Info     []ValidationMessage `json:"infoMessages"`
}
type ComplianceResult struct {
	ReportingStatus   string            `json:"reportingStatus"`
	ValidationResults ValidationResults `json:"validationResults"`
}
type ClearanceRequest struct {
	InvoiceXML  string `json:"invoice"`
	InvoiceHash string `json:"invoiceHash"`
	UUID        string `json:"uuid"`
}
type ClearanceResponse struct {
	Status            string            `json:"reportingStatus"`
	ClearedInvoice    string            `json:"clearedInvoice,omitempty"`
	ValidationResults ValidationResults `json:"validationResults"`
}
type ReportingRequest = ClearanceRequest
type ReportingResponse = ClearanceResponse

func (c *Client) ComplianceCSID(ctx context.Context, csr, otp string) (*CSIDResponse, error) {
	var result CSIDResponse
	err := c.post(ctx, "/compliance", map[string]string{"csr": csr, "otp": otp}, &result)
	return &result, err
}
func (c *Client) ComplianceCheck(ctx context.Context, invoice []byte, invoiceHash, uuid string) (*ComplianceResult, error) {
	var result ComplianceResult
	err := c.post(ctx, "/compliance/invoice", map[string]any{"invoiceHash": invoiceHash, "invoice": base64.StdEncoding.EncodeToString(invoice), "uuid": uuid}, &result)
	return &result, err
}
func (c *Client) ProductionCSID(ctx context.Context, complianceCSID, requestID string) (*CSIDResponse, error) {
	var result CSIDResponse
	err := c.post(ctx, "/production/csids", map[string]string{"complianceCSID": complianceCSID, "requestID": requestID}, &result)
	return &result, err
}
func (c *Client) ClearInvoice(ctx context.Context, request *ClearanceRequest) (*ClearanceResponse, error) {
	var result ClearanceResponse
	err := c.post(ctx, "/invoices/clearance/single", request, &result)
	return &result, err
}
func (c *Client) ReportInvoice(ctx context.Context, request *ReportingRequest) (*ReportingResponse, error) {
	var result ReportingResponse
	err := c.post(ctx, "/invoices/reporting/single", request, &result)
	return &result, err
}

func (c *Client) post(ctx context.Context, path string, body any, result any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Basic "+c.authToken)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("ZATCA API returned status %d: %s", response.StatusCode, string(data))
	}
	return json.Unmarshal(data, result)
}
