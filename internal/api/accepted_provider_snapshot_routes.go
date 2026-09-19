package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/cpmroutes"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
)

// registerAcceptedProviderSnapshotRoute registers POST /accepted-provider-snapshots
// (CFB-P5 Option B): read-only construction of accepted_provider_snapshot from the
// provider registry so clients need no FE provider fixture.
func registerAcceptedProviderSnapshotRoute(mux *http.ServeMux, store *ReadStore) {
	mux.HandleFunc("POST "+cpmroutes.AcceptedProviderSnapshots, func(w http.ResponseWriter, r *http.Request) {
		req, err := decodeAcceptedProviderSnapshotRequest(r)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		cp, ok := store.cryptoPolicyByID[req.CryptoPolicyID]
		if !ok {
			respondJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown crypto_policy_id"})
			return
		}

		result, err := policy.BuildAcceptedProviderSnapshot(policy.BuildAcceptedProviderSnapshotInput{
			CryptoPolicy:       cp,
			SolutionProfileRef: req.SolutionProfileRef,
			ChainID:            req.ChainID,
			AcceptedFindings:   req.AcceptedFindings,
			Registry:           store.providers,
		})
		if err != nil {
			status, msg := mapSnapshotAssistError(err)
			respondJSON(w, status, map[string]any{"error": msg})
			return
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"crypto_policy_id":             cp.ID,
			"required_posture":             cp.RequiredPosture,
			"solution_profile_ref":         result.SolutionProfileRef,
			"accepted_provider_snapshot":   result.AcceptedProviderSnapshot,
		})
	})
}

type acceptedProviderSnapshotRequest struct {
	CryptoPolicyID     string                   `json:"crypto_policy_id"`
	SolutionProfileRef policy.SolutionProfileRef `json:"solution_profile_ref"`
	ChainID            int64                    `json:"-"`
	AcceptedFindings   []string                 `json:"accepted_findings"`
}

func decodeAcceptedProviderSnapshotRequest(r *http.Request) (*acceptedProviderSnapshotRequest, error) {
	if r == nil {
		return nil, errors.New("request is nil")
	}
	defer func() { _ = r.Body.Close() }()
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, fmt.Errorf("invalid json body: %w", err)
	}
	for key := range raw {
		switch key {
		case "crypto_policy_id", "solution_profile_ref", "chain_id", "accepted_findings":
		default:
			return nil, fmt.Errorf("unknown field %s", key)
		}
	}

	cpRaw, ok := raw["crypto_policy_id"]
	if !ok {
		return nil, errors.New("crypto_policy_id is required")
	}
	var cryptoPolicyID string
	if err := json.Unmarshal(cpRaw, &cryptoPolicyID); err != nil {
		return nil, fmt.Errorf("crypto_policy_id: %w", err)
	}
	cryptoPolicyID = strings.TrimSpace(cryptoPolicyID)
	if cryptoPolicyID == "" {
		return nil, errors.New("crypto_policy_id is required")
	}

	refRaw, ok := raw["solution_profile_ref"]
	if !ok || len(bytesTrimSpaceJSON(refRaw)) == 0 {
		return nil, errors.New("solution_profile_ref is required")
	}
	var ref policy.SolutionProfileRef
	if err := json.Unmarshal(refRaw, &ref); err != nil {
		return nil, fmt.Errorf("solution_profile_ref: %w", err)
	}

	chainRaw, ok := raw["chain_id"]
	if !ok || len(bytesTrimSpaceJSON(chainRaw)) == 0 {
		return nil, errors.New("chain_id is required")
	}
	var chainID policy.FlexibleChainID
	if err := json.Unmarshal(chainRaw, &chainID); err != nil {
		return nil, fmt.Errorf("chain_id: %w", err)
	}
	if chainID.Int64() < 1 {
		return nil, errors.New("chain_id is required")
	}

	var findings []string
	if v, ok := raw["accepted_findings"]; ok && len(bytesTrimSpaceJSON(v)) > 0 && string(v) != "null" {
		if err := json.Unmarshal(v, &findings); err != nil {
			return nil, fmt.Errorf("accepted_findings: %w", err)
		}
	}

	return &acceptedProviderSnapshotRequest{
		CryptoPolicyID:     cryptoPolicyID,
		SolutionProfileRef: ref,
		ChainID:            chainID.Int64(),
		AcceptedFindings:   findings,
	}, nil
}

func mapSnapshotAssistError(err error) (int, string) {
	switch {
	case errors.Is(err, policy.ErrSnapshotAssistChainNotSupported),
		errors.Is(err, policy.ErrSnapshotAssistProviderNotAllowed),
		errors.Is(err, policy.ErrSnapshotAssistProfileNotFound):
		return http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, policy.ErrSnapshotAssistFindingsUnknown),
		errors.Is(err, policy.ErrSnapshotAssistChainRequired),
		errors.Is(err, policy.ErrSnapshotAssistCryptoPolicyRequired),
		errors.Is(err, policy.ErrCryptoPolicyPayloadInvalid):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusBadRequest, err.Error()
	}
}
