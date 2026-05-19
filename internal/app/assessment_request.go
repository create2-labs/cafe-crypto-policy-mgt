package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/create2-labs/cafe-contracts/cafenatsv01"
	walletv01 "github.com/create2-labs/cafe-contracts/observation/wallet/v01"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/api"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
)

func registerPoliciesAssessmentRequestRoute(mux *http.ServeMux, authCfg authConfig) {
	mux.HandleFunc("POST "+cpmroutes.PoliciesAssessmentRequest, func(w http.ResponseWriter, r *http.Request) {
		handlePoliciesAssessmentRequest(w, r, authCfg)
	})
}

func handlePoliciesAssessmentRequest(w http.ResponseWriter, r *http.Request, authCfg authConfig) {
	obs := authCfg.Observability
	if obs == nil {
		obs = newAuthObservability()
	}
	requestID := obs.ensureRequestID(w, r)
	principal, ok := principalFromContext(r.Context())
	if !ok {
		obs.recordDecision(r, requestID, authCategoryOwner, "denied", authCodePrincipalRequired, "principal_missing", "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, "authentication required", map[string]any{"reason": "principal_missing"})
		return
	}

	if authCfg.AssessmentNATSPublish == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "assessment_transport_unavailable",
			"message": "policy assessment request publishing is not configured",
		})
		return
	}
	if strings.TrimSpace(authCfg.DiscoveryHTTPBaseURL) == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "discovery_upstream_unavailable",
			"message": "Discovery HTTP base URL is not configured",
		})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_request", "message": "could not read body"})
		return
	}
	defer func() { _ = r.Body.Close() }()

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}
	if _, has := raw["policy_context"]; has {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "policy_context_forbidden",
			"message": "policy_context must not be present on this route; use Discovery wallet scan detail only",
		})
		return
	}
	for k := range raw {
		switch k {
		case "scan_id", "selection_request", "client_request_id":
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown_field", "message": "unknown field " + k})
			return
		}
	}

	scanRaw, ok := raw["scan_id"]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scan_id_required", "message": "scan_id is required"})
		return
	}
	var scanID string
	if err := json.Unmarshal(scanRaw, &scanID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scan_id_invalid", "message": err.Error()})
		return
	}
	normScanID, err := NormalizeDiscoveryScanID(scanID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "scan_id_invalid", "message": err.Error()})
		return
	}

	selRaw, ok := raw["selection_request"]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "selection_request_required", "message": "selection_request is required"})
		return
	}
	decSel := json.NewDecoder(bytes.NewReader(selRaw))
	decSel.DisallowUnknownFields()
	var sel policy.PolicySelectionRequest
	if err := decSel.Decode(&sel); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "selection_request_invalid", "message": err.Error()})
		return
	}
	if decSel.More() {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "selection_request_invalid", "message": "multiple JSON values"})
		return
	}
	sel.Normalize()
	if err := sel.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "selection_request_invalid", "message": err.Error()})
		return
	}

	var clientRequestID string
	if v, ok := raw["client_request_id"]; ok {
		_ = json.Unmarshal(v, &clientRequestID)
	}

	if scanAuthErr, status := authorizeScanReadForAssessment(r.Context(), principal, normScanID, authCfg, requestID); scanAuthErr.Code != "" {
		writeJSON(w, status, map[string]any{"error": scanAuthErr.Code, "message": scanAuthErr.Message, "details": scanAuthErr.Details})
		return
	}

	detailJSON, st, err := fetchDiscoveryWalletScanDetail(r.Context(), authCfg, r.Header.Get("Authorization"), requestID, normScanID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "discovery_unavailable", "message": err.Error()})
		return
	}
	switch st {
	case http.StatusNotFound:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "scan not found"})
		return
	case http.StatusUnauthorized, http.StatusForbidden:
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "discovery_unavailable", "message": "Discovery rejected the session for scan detail"})
		return
	default:
		if st >= 500 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "discovery_unavailable", "message": fmt.Sprintf("Discovery returned %d", st)})
			return
		}
		if st != http.StatusOK {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "discovery_unavailable", "message": fmt.Sprintf("unexpected Discovery status %d", st)})
			return
		}
	}

	pl, err := api.ObservationPayloadFromDiscoveryWalletScanDetail(detailJSON)
	if err != nil {
		switch {
		case errors.Is(err, api.ErrWalletScanDetailTLS), errors.Is(err, api.ErrWalletScanDetailNoResult):
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "scan not found"})
			return
		default:
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "wallet_scan_detail_invalid", "message": err.Error()})
			return
		}
	}

	walletAddr, werr := targetAddressFromWalletScanDetailJSON(detailJSON)
	if werr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "wallet_scan_detail_invalid", "message": werr.Error()})
		return
	}
	walletSubjectID := normalizeWalletSubjectForAssessment(walletAddr)

	obsEventID, err := newAssessmentEventID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal_error", "message": "could not allocate event id"})
		return
	}
	now := time.Now().UTC()
	observation := walletv01.Event{
		EventID:       obsEventID + "_obs",
		EventType:     walletv01.EventTypeWalletObserved,
		EventVersion:  walletv01.EventVersion,
		OccurredAt:    now,
		CorrelationID: normScanID,
		CausationID:   "cpm_post_policies_assessment_request",
		Producer:      walletv01.ProducerCafeDiscovery,
		Subject: walletv01.Subject{
			Type: string(walletv01.SubjectTypeWallet),
			ID:   walletSubjectID,
		},
		Payload: pl,
	}
	if err := observation.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "observation_invalid", "message": err.Error()})
		return
	}

	cmdEventID, err := newAssessmentEventID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal_error", "message": "could not allocate event id"})
		return
	}
	selWire := policySelectionRequestToWire(sel)
	selWire.Normalize()
	if err := selWire.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "selection_request_invalid", "message": err.Error()})
		return
	}

	cmd := cafenatsv01.PolicyAssessmentRequested{
		EnvelopeV01: cafenatsv01.EnvelopeV01{
			EventID:       cmdEventID,
			EventType:     cafenatsv01.EventTypePolicyAssessmentRequested,
			EventVersion:  cafenatsv01.EventVersionV01,
			OccurredAt:    now,
			CorrelationID: normScanID,
			CausationID:   "cpm_post_policies_assessment_request",
			Producer:      cafenatsv01.ProducerCafeCryptoBackend,
		},
		Subject: cafenatsv01.SubjectRef{
			Type: cafenatsv01.SubjectTypeWallet,
			ID:   walletSubjectID,
		},
		Payload: cafenatsv01.PolicyAssessmentRequestedPayload{
			Observation:      observation,
			SelectionRequest: selWire,
			ClientRequestID:  strings.TrimSpace(clientRequestID),
		},
	}
	if err := cmd.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "assessment_command_invalid", "message": err.Error()})
		return
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "internal_error", "message": err.Error()})
		return
	}
	if err := authCfg.AssessmentNATSPublish(r.Context(), cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "publish_failed", "message": err.Error()})
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"event_id":         cmd.EventID,
		"correlation_id":   cmd.CorrelationID,
		"client_request_id": strings.TrimSpace(clientRequestID),
	})
}

func fetchDiscoveryWalletScanDetail(ctx context.Context, authCfg authConfig, authorization string, requestID, scanID string) ([]byte, int, error) {
	timeout := authCfg.DiscoveryHTTPTimeoutSec
	if timeout <= 0 {
		timeout = 5
	}
	u := strings.TrimSuffix(strings.TrimSpace(authCfg.DiscoveryHTTPBaseURL), "/") + "/discovery/v1/wallets/scans/" + url.PathEscape(scanID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return nil, 0, err
	}
	if strings.TrimSpace(authorization) != "" {
		req.Header.Set("Authorization", authorization)
	}
	if requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return raw, resp.StatusCode, nil
}

func targetAddressFromWalletScanDetailJSON(detail []byte) (string, error) {
	var wrap struct {
		Result struct {
			TargetAddress string `json:"target_address"`
		} `json:"result"`
	}
	if err := json.Unmarshal(detail, &wrap); err != nil {
		return "", err
	}
	addr := strings.TrimSpace(wrap.Result.TargetAddress)
	if addr == "" {
		return "", fmt.Errorf("target_address is required in wallet scan result")
	}
	return addr, nil
}

func normalizeWalletSubjectForAssessment(address string) string {
	const walletPrefix = "wallet:"
	a := strings.TrimSpace(address)
	la := strings.ToLower(a)
	if strings.HasPrefix(la, walletPrefix) {
		return walletPrefix + strings.ToLower(strings.TrimSpace(a[len(walletPrefix):]))
	}
	if strings.HasPrefix(la, "0x") {
		return walletPrefix + strings.ToLower(a)
	}
	return walletPrefix + strings.ToLower(a)
}

func policySelectionRequestToWire(r policy.PolicySelectionRequest) cafenatsv01.PolicySelectionRequestWire {
	modes := make([]string, 0, len(r.AllowedProviderModes))
	for _, m := range r.AllowedProviderModes {
		modes = append(modes, string(m))
	}
	return cafenatsv01.PolicySelectionRequestWire{
		TargetPosture:             string(r.TargetPosture),
		TargetChainIDs:            append([]int64(nil), r.TargetChainIDs...),
		RequireMultichain:         r.RequireMultichain,
		AllowNewWallet:            r.AllowNewWallet,
		AddressContinuityRequired: r.AddressContinuityRequired,
		KeyRotationRequired:       r.KeyRotationRequired,
		RecoveryRequired:          r.RecoveryRequired,
		MinimumMaturity:           r.MinimumMaturity,
		AllowResearch:             r.AllowResearch,
		AllowedProviderModes:      modes,
		PreferredFamilies:         append([]string(nil), r.PreferredFamilies...),
		PreferredProviders:        append([]string(nil), r.PreferredProviders...),
		RequireBundlerAvailable:   r.RequireBundlerAvailable,
		RequirePaymasterAvailable: r.RequirePaymasterAvailable,
		ApprovalMode:              string(r.ApprovalMode),
	}
}

func newAssessmentEventID() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "evt_pol_req_" + hex.EncodeToString(b[:]), nil
}
