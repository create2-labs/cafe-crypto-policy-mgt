package cphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/authz"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

// Config wires the cafe-persistence internal CP HTTP client (openapi/internal/cp/v1).
type Config struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// Client implements persistence.PolicyStore over HTTP (PERS-D5a).
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

var _ persistence.PolicyStore = (*Client)(nil)

// NewClient returns a PolicyStore backed by cafe-persistence internal/cp/v1.
func NewClient(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		token:      strings.TrimSpace(cfg.Token),
		httpClient: hc,
	}
}

func (c *Client) SavePolicy(principal authz.Principal, id string, scanID string, payload map[string]any) (persistence.PolicyRecord, error) {
	_ = principal
	_ = id
	_ = scanID
	_ = payload
	return persistence.PolicyRecord{}, persistence.ErrUnsupportedStoreOperation
}

func (c *Client) CreatePolicy(principal authz.Principal, in persistence.CreatePolicyInput) (persistence.CreatePolicyResult, error) {
	if err := principal.Validate(); err != nil {
		return persistence.CreatePolicyResult{}, persistence.ErrPrincipalRequired
	}
	verifiedAt := in.WalletControlVerifiedAt.UTC()
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}
	body := map[string]any{
		"scan_id":                    strings.TrimSpace(in.ScanID),
		"wallet_address":             strings.TrimSpace(in.WalletAddress),
		"chain_id":                   in.ChainID,
		"payload":                    in.Payload,
		"payload_sha256":             strings.TrimSpace(in.PayloadSHA256),
		"signed_message_hash":        strings.TrimSpace(in.SignedMessageHash),
		"wallet_control_method":      strings.TrimSpace(in.WalletControlMethod),
		"wallet_control_verified_at": verifiedAt.Format(time.RFC3339Nano),
	}
	if method := strings.TrimSpace(in.WalletControlMethod); method == "" {
		body["wallet_control_method"] = "eoa_signature"
	}
	if in.ChallengeIssuedAt != nil && !in.ChallengeIssuedAt.IsZero() {
		body["challenge_issued_at"] = in.ChallengeIssuedAt.UTC().Format(time.RFC3339Nano)
	}
	if in.ChallengeExpiresAt != nil && !in.ChallengeExpiresAt.IsZero() {
		body["challenge_expires_at"] = in.ChallengeExpiresAt.UTC().Format(time.RFC3339Nano)
	}
	var wire createPolicyResponseWire
	if err := c.doJSON(context.Background(), http.MethodPost, c.baseURL+V1Base+Policies, principal, body, false, &wire); err != nil {
		return persistence.CreatePolicyResult{}, err
	}
	persistedAt, err := parseTime(wire.PersistedAt)
	if err != nil {
		return persistence.CreatePolicyResult{}, err
	}
	return persistence.CreatePolicyResult{
		PolicyID:      strings.TrimSpace(wire.PolicyID),
		ScanID:        strings.TrimSpace(wire.ScanID),
		WalletAddress: strings.TrimSpace(wire.WalletAddress),
		ChainID:       wire.ChainID,
		PayloadSHA256: strings.TrimSpace(wire.PayloadSHA256),
		PersistedAt:   persistedAt,
	}, nil
}

func (c *Client) ListPersistedPoliciesForScan(principal authz.Principal, scanID string) ([]persistence.PolicyRecord, error) {
	if err := principal.Validate(); err != nil {
		return nil, persistence.ErrPrincipalRequired
	}
	scanID = strings.TrimSpace(scanID)
	if scanID == "" {
		return nil, errors.New("scan_id is required")
	}
	u := c.baseURL + V1Base + Policies + "?scan_id=" + url.QueryEscape(scanID)
	var wire listPoliciesWire
	if err := c.doJSON(context.Background(), http.MethodGet, u, principal, nil, true, &wire); err != nil {
		return nil, err
	}
	out := make([]persistence.PolicyRecord, 0, len(wire.Items))
	for _, item := range wire.Items {
		rec, err := item.toRecord()
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, nil
}

func (c *Client) DeletePolicy(principal authz.Principal, id string) error {
	if err := principal.Validate(); err != nil {
		return persistence.ErrPrincipalRequired
	}
	return c.doNoContent(context.Background(), http.MethodDelete, c.policyURL(strings.TrimSpace(id)), principal)
}

func (c *Client) GetPolicy(principal authz.Principal, id string) (persistence.PolicyRecord, error) {
	if err := principal.Validate(); err != nil {
		return persistence.PolicyRecord{}, persistence.ErrPrincipalRequired
	}
	var wire policyRowWire
	if err := c.doJSON(context.Background(), http.MethodGet, c.policyURL(strings.TrimSpace(id)), principal, nil, true, &wire); err != nil {
		return persistence.PolicyRecord{}, err
	}
	return wire.toRecord()
}

func (c *Client) CountActiveWalletCPMContext(principal authz.Principal, normalizedTargetAddress string) (persistence.WalletTargetContextCounts, error) {
	if err := principal.Validate(); err != nil {
		return persistence.WalletTargetContextCounts{}, persistence.ErrPrincipalRequired
	}
	wallet, err := persistence.NormalizeWalletTargetAddress(normalizedTargetAddress)
	if err != nil {
		return persistence.WalletTargetContextCounts{}, err
	}
	u := c.baseURL + V1Base + ReferenceWallet + "?wallet_address=" + url.QueryEscape(wallet)
	var wire walletReferenceWire
	if err := c.doJSON(context.Background(), http.MethodGet, u, principal, nil, true, &wire); err != nil {
		return persistence.WalletTargetContextCounts{}, err
	}
	return persistence.WalletTargetContextCounts{
		Exists:      wire.Exists,
		PolicyCount: int(wire.PolicyCount),
	}, nil
}

func (c *Client) policyURL(policyID string) string {
	rel := strings.ReplaceAll(PolicyByID, "{policy_id}", policyID)
	return c.baseURL + V1Base + rel
}

func (c *Client) doNoContent(ctx context.Context, method, url string, principal authz.Principal) error {
	return c.doJSON(ctx, method, url, principal, nil, true, nil)
}

func (c *Client) doJSON(ctx context.Context, method, url string, principal authz.Principal, body any, idempotent bool, dst any) error {
	var lastErr error
	attempts := 1
	if idempotent {
		attempts = 2
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(50 * time.Millisecond)
		}
		lastErr = c.doJSONOnce(ctx, method, url, principal, body, dst)
		if lastErr == nil {
			return nil
		}
		if !idempotent || !errors.Is(lastErr, persistence.ErrPersistenceUnavailable) {
			return lastErr
		}
	}
	return lastErr
}

func (c *Client) doJSONOnce(ctx context.Context, method, url string, principal authz.Principal, body any, dst any) error {
	var bodyReader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set(headerAuthorization, "Bearer "+c.token)
	req.Header.Set(headerUserID, principal.UserID)
	if tenant := strings.TrimSpace(principal.TenantID); tenant != "" {
		req.Header.Set(headerTenantID, tenant)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return persistence.ErrPersistenceUnavailable
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return persistence.ErrPersistenceUnavailable
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if dst == nil {
			return nil
		}
		if len(raw) == 0 {
			return nil
		}
		if err := json.Unmarshal(raw, dst); err != nil {
			return persistence.ErrPersistenceUnavailable
		}
		return nil
	}
	return mapHTTPError(resp.StatusCode, raw)
}

func mapHTTPError(status int, _ []byte) error {
	switch status {
	case http.StatusNotFound:
		return persistence.ErrPolicyNotFound
	case http.StatusForbidden:
		return persistence.ErrForbidden
	case http.StatusConflict:
		return persistence.ErrPolicyAlreadyExists
	default:
		return persistence.ErrPersistenceUnavailable
	}
}

type policyRowWire struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	TenantID      string         `json:"tenant_id"`
	ScanID        string         `json:"scan_id"`
	Payload       map[string]any `json:"payload"`
	PayloadSHA256 string         `json:"payload_sha256"`
	WalletAddress string         `json:"wallet_address"`
	ChainID       int64          `json:"chain_id"`
	CreatedAt     string         `json:"created_at"`
	UpdatedAt     string         `json:"updated_at"`
	PersistedAt   string         `json:"persisted_at"`
}

func (w policyRowWire) toRecord() (persistence.PolicyRecord, error) {
	createdAt, err := parseTime(w.CreatedAt)
	if err != nil {
		return persistence.PolicyRecord{}, err
	}
	updatedAt, err := parseTime(w.UpdatedAt)
	if err != nil {
		return persistence.PolicyRecord{}, err
	}
	payload := w.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return persistence.PolicyRecord{
		ID:            strings.TrimSpace(w.ID),
		OwnerUserID:   strings.TrimSpace(w.UserID),
		TenantID:      strings.TrimSpace(w.TenantID),
		ScanID:        strings.TrimSpace(w.ScanID),
		Payload:       payload,
		PayloadSHA256: strings.TrimSpace(w.PayloadSHA256),
		WalletAddress: strings.TrimSpace(w.WalletAddress),
		ChainID:       w.ChainID,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}, nil
}

type createPolicyResponseWire struct {
	PolicyID      string `json:"policy_id"`
	ScanID        string `json:"scan_id"`
	WalletAddress string `json:"wallet_address"`
	ChainID       int64  `json:"chain_id"`
	PayloadSHA256 string `json:"payload_sha256"`
	PersistedAt   string `json:"persisted_at"`
}

type listPoliciesWire struct {
	Items []policyRowWire `json:"items"`
}

type walletReferenceWire struct {
	Exists      bool  `json:"exists"`
	PolicyCount int64 `json:"policy_count"`
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}
