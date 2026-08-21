package app

import (
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
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/walletobserved"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
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
		obs.recordDecision(r, requestID, authCategoryOwner, authOutcomeDenied, authCodePrincipalRequired, authReasonPrincipalMissing, "", "")
		obs.writeAuthError(w, r, http.StatusUnauthorized, authCodePrincipalRequired, errMsgAuthenticationRequired, reasonDetails(authReasonPrincipalMissing))
		return
	}
	if !assessmentRequestPreconditions(w, authCfg) {
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("invalid_request", "could not read body"))
		return
	}
	defer func() { _ = r.Body.Close() }()

	parsed, ok := parseAssessmentRequestBody(w, body)
	if !ok {
		return
	}

	if scanAuthErr, status := authorizeScanReadForAssessment(r.Context(), principal, parsed.normScanID, authCfg, requestID); scanAuthErr.Code != "" {
		writeJSON(w, status, apiErrorWithDetails(scanAuthErr.Code, scanAuthErr.Message, scanAuthErr.Details))
		return
	}

	source, ok := fetchWalletScanAssessmentSource(w, r.Context(), authCfg, r.Header.Get("Authorization"), requestID, parsed.normScanID)
	if !ok {
		return
	}

	publishPolicyAssessmentRequest(w, r.Context(), authCfg, parsed, source)
}

type parsedAssessmentRequest struct {
	normScanID      string
	cryptoPolicyID  string
	clientRequestID string
}

type walletScanAssessmentSource struct {
	payload         walletobserved.Payload
	walletSubjectID string
}

func assessmentRequestPreconditions(w http.ResponseWriter, authCfg authConfig) bool {
	if authCfg.AssessmentNATSPublish == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(
			"assessment_transport_unavailable",
			"policy assessment request publishing is not configured",
		))
		return false
	}
	if strings.TrimSpace(authCfg.DiscoveryHTTPBaseURL) == "" {
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(
			"discovery_upstream_unavailable",
			"Discovery HTTP base URL is not configured",
		))
		return false
	}
	return true
}

func parseAssessmentRequestBody(w http.ResponseWriter, body []byte) (parsedAssessmentRequest, bool) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("invalid_json", err.Error()))
		return parsedAssessmentRequest{}, false
	}
	if _, has := raw["policy_context"]; has {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			jsonKeyError:   "policy_context_forbidden",
			jsonKeyMessage: "policy_context must not be present on this route; use Discovery wallet scan detail only",
		})
		return parsedAssessmentRequest{}, false
	}
	for k := range raw {
		if _, forbidden := assessmentHTTPForbiddenKeys[k]; forbidden {
			writeJSON(w, http.StatusBadRequest, apiErrorJSON("legacy_field_rejected", "legacy field rejected: "+k))
			return parsedAssessmentRequest{}, false
		}
	}
	for k := range raw {
		switch k {
		case jsonFieldScanID, jsonFieldCryptoPolicyID, "client_request_id":
		default:
			writeJSON(w, http.StatusBadRequest, apiErrorJSON("unknown_field", "unknown field "+k))
			return parsedAssessmentRequest{}, false
		}
	}

	scanRaw, ok := raw[jsonFieldScanID]
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("scan_id_required", "scan_id is required"))
		return parsedAssessmentRequest{}, false
	}
	var scanID string
	if err := json.Unmarshal(scanRaw, &scanID); err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("scan_id_invalid", err.Error()))
		return parsedAssessmentRequest{}, false
	}
	normScanID, err := NormalizeDiscoveryScanID(scanID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("scan_id_invalid", err.Error()))
		return parsedAssessmentRequest{}, false
	}

	cpRaw, ok := raw[jsonFieldCryptoPolicyID]
	if !ok {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("crypto_policy_id_required", "crypto_policy_id is required"))
		return parsedAssessmentRequest{}, false
	}
	var cryptoPolicyID string
	if err := json.Unmarshal(cpRaw, &cryptoPolicyID); err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("crypto_policy_id_invalid", err.Error()))
		return parsedAssessmentRequest{}, false
	}
	cryptoPolicyID = strings.TrimSpace(cryptoPolicyID)
	if cryptoPolicyID == "" {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("crypto_policy_id_required", "crypto_policy_id is required"))
		return parsedAssessmentRequest{}, false
	}

	var clientRequestID string
	if v, ok := raw["client_request_id"]; ok {
		_ = json.Unmarshal(v, &clientRequestID)
	}
	return parsedAssessmentRequest{
		normScanID:      normScanID,
		cryptoPolicyID:  cryptoPolicyID,
		clientRequestID: clientRequestID,
	}, true
}

// assessmentHTTPForbiddenKeys: selection_request and couche-B fields are invalid on this HTTP trigger
// (same family as explore/assessment NATS v0.2; persist user_constraints is CPM-P10).
var assessmentHTTPForbiddenKeys = map[string]struct{}{
	jsonFieldSelectionRequest:     {},
	"allow_new_wallet":            {},
	"address_continuity_required": {},
	"key_rotation_model":          {},
	"target_posture":              {},
	"target_chain_ids":            {},
	"require_multichain":          {},
	"recovery_required":           {},
	"minimum_maturity":            {},
	"allow_research":              {},
	"allowed_provider_modes":      {},
	"preferred_families":          {},
	"preferred_providers":         {},
	"require_bundler_available":   {},
	"require_paymaster_available": {},
	"approval_mode":               {},
}

func fetchWalletScanAssessmentSource(
	w http.ResponseWriter,
	ctx context.Context,
	authCfg authConfig,
	authorization, requestID, normScanID string,
) (walletScanAssessmentSource, bool) {
	detailJSON, st, err := fetchDiscoveryWalletScanDetail(ctx, authCfg, authorization, requestID, normScanID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodeDiscoveryUnavailable, err.Error()))
		return walletScanAssessmentSource{}, false
	}
	if !writeDiscoveryScanDetailStatusOK(w, st) {
		return walletScanAssessmentSource{}, false
	}
	return parseWalletScanAssessmentSource(w, detailJSON)
}

func writeDiscoveryScanDetailStatusOK(w http.ResponseWriter, st int) bool {
	switch st {
	case http.StatusNotFound:
		writeJSON(w, http.StatusNotFound, apiErrorJSON(errCodeNotFound, errMsgScanNotFound))
		return false
	case http.StatusUnauthorized, http.StatusForbidden:
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodeDiscoveryUnavailable, "Discovery rejected the session for scan detail"))
		return false
	case http.StatusOK:
		return true
	default:
		if st >= 500 {
			writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodeDiscoveryUnavailable, fmt.Sprintf("Discovery returned %d", st)))
			return false
		}
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON(errCodeDiscoveryUnavailable, fmt.Sprintf("unexpected Discovery status %d", st)))
		return false
	}
}

func parseWalletScanAssessmentSource(w http.ResponseWriter, detailJSON []byte) (walletScanAssessmentSource, bool) {
	pl, err := api.ObservationPayloadFromDiscoveryWalletScanDetail(detailJSON)
	if err != nil {
		switch {
		case errors.Is(err, api.ErrWalletScanDetailTLS), errors.Is(err, api.ErrWalletScanDetailNoResult):
			writeJSON(w, http.StatusNotFound, apiErrorJSON(errCodeNotFound, errMsgScanNotFound))
		default:
			writeJSON(w, http.StatusBadRequest, apiErrorJSON("wallet_scan_detail_invalid", err.Error()))
		}
		return walletScanAssessmentSource{}, false
	}
	walletAddr, werr := targetAddressFromWalletScanDetailJSON(detailJSON)
	if werr != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("wallet_scan_detail_invalid", werr.Error()))
		return walletScanAssessmentSource{}, false
	}
	subjectID, serr := persistence.WalletSubjectIDFromAddress(walletAddr)
	if serr != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("wallet_scan_detail_invalid", serr.Error()))
		return walletScanAssessmentSource{}, false
	}
	return walletScanAssessmentSource{
		payload:         pl,
		walletSubjectID: subjectID,
	}, true
}

func publishPolicyAssessmentRequest(
	w http.ResponseWriter,
	ctx context.Context,
	authCfg authConfig,
	parsed parsedAssessmentRequest,
	source walletScanAssessmentSource,
) {
	now := time.Now().UTC()
	observation, ok := buildWalletObservedEvent(w, parsed.normScanID, source, now)
	if !ok {
		return
	}
	cmd, ok := buildPolicyAssessmentCommand(w, parsed, source, observation, now)
	if !ok {
		return
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiErrorJSON(errCodeInternalError, err.Error()))
		return
	}
	if err := authCfg.AssessmentNATSPublish(ctx, cafenatsv01.NATSSubjectPolicyAssessmentRequestedV01, payload); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, apiErrorJSON("publish_failed", err.Error()))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"event_id":          cmd.EventID,
		"correlation_id":    cmd.CorrelationID,
		"client_request_id": strings.TrimSpace(parsed.clientRequestID),
	})
}

func buildWalletObservedEvent(
	w http.ResponseWriter,
	normScanID string,
	source walletScanAssessmentSource,
	now time.Time,
) (walletv01.Event, bool) {
	obsEventID, err := newAssessmentEventID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiErrorJSON(errCodeInternalError, "could not allocate event id"))
		return walletv01.Event{}, false
	}
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
			ID:   source.walletSubjectID,
		},
		Payload: source.payload,
	}
	if err := observation.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("observation_invalid", err.Error()))
		return walletv01.Event{}, false
	}
	return observation, true
}

func buildPolicyAssessmentCommand(
	w http.ResponseWriter,
	parsed parsedAssessmentRequest,
	source walletScanAssessmentSource,
	observation walletv01.Event,
	now time.Time,
) (cafenatsv01.PolicyAssessmentRequested, bool) {
	cmdEventID, err := newAssessmentEventID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiErrorJSON(errCodeInternalError, "could not allocate event id"))
		return cafenatsv01.PolicyAssessmentRequested{}, false
	}
	cmd := cafenatsv01.PolicyAssessmentRequested{
		EnvelopeV01: cafenatsv01.EnvelopeV01{
			EventID:       cmdEventID,
			EventType:     cafenatsv01.EventTypePolicyAssessmentRequested,
			EventVersion:  cafenatsv01.EventVersionV01,
			OccurredAt:    now,
			CorrelationID: parsed.normScanID,
			CausationID:   "cpm_post_policies_assessment_request",
			Producer:      cafenatsv01.ProducerCafeCryptoBackend,
		},
		Subject: cafenatsv01.SubjectRef{
			Type: cafenatsv01.SubjectTypeWallet,
			ID:   source.walletSubjectID,
		},
		Payload: cafenatsv01.PolicyAssessmentRequestedPayload{
			CryptoPolicyID:  parsed.cryptoPolicyID,
			Observation:     observation,
			ClientRequestID: strings.TrimSpace(parsed.clientRequestID),
		},
	}
	if err := cmd.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, apiErrorJSON("assessment_command_invalid", err.Error()))
		return cafenatsv01.PolicyAssessmentRequested{}, false
	}
	return cmd, true
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
		ScanFamily string `json:"scan_family"`
		Result     struct {
			TargetAddress string `json:"target_address"`
		} `json:"result"`
	}
	if err := json.Unmarshal(detail, &wrap); err != nil {
		return "", err
	}
	if strings.EqualFold(strings.TrimSpace(wrap.ScanFamily), "tls") {
		return "", errDiscoveryScanNotWallet
	}
	addr := strings.TrimSpace(wrap.Result.TargetAddress)
	if addr == "" {
		return "", errDiscoveryScanNotWallet
	}
	return addr, nil
}

func newAssessmentEventID() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "evt_pol_req_" + hex.EncodeToString(b[:]), nil
}
