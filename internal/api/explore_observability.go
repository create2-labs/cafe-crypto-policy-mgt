package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/policy"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/metrics"
	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/persistence"
)

const (
	exploreNoDeployableEventName = "cpm.explore.no_deployable_candidate"
	rejectionCodeChainScope      = "incompatible.chain_scope"
	exploreBindingDiscovery      = "discovery"
	exploreBindingUnknown        = "unknown"
	exploreLabelUnknown          = "unknown"
)

type exploreMetricsRecorder interface {
	IncExploreNoDeployableCandidate(rejectionCode, walletType, binding, missingChainCount string)
}

type prometheusExploreMetrics struct{}

func (prometheusExploreMetrics) IncExploreNoDeployableCandidate(rejectionCode, walletType, binding, missingChainCount string) {
	metrics.IncExploreNoDeployableCandidate(rejectionCode, walletType, binding, missingChainCount)
}

type exploreLogger interface {
	Println(v ...any)
}

type exploreObservability struct {
	logger  exploreLogger
	metrics exploreMetricsRecorder
}

var exploreObs = exploreObservability{
	logger:  log.Default(),
	metrics: prometheusExploreMetrics{},
}

func recordExploreNoDeployableCandidate(
	r *http.Request,
	req *decisionExploreRequest,
	decision policy.PolicyDecision,
	instanceScopes map[string][]int64,
) {
	exploreObs.recordNoDeployableCandidate(r, req, decision, instanceScopes)
}

func (o exploreObservability) recordNoDeployableCandidate(
	r *http.Request,
	req *decisionExploreRequest,
	decision policy.PolicyDecision,
	instanceScopes map[string][]int64,
) {
	if len(decision.RankedCandidates) > 0 || len(decision.RejectedCandidates) == 0 {
		return
	}

	targetChainIDs := decision.RequestSummary.TargetChainIDs
	dominantCode := dominantExploreRejectionCode(decision.RejectedCandidates)
	missingCountLabel := bucketMissingChainCount(targetChainIDs, decision.RejectedCandidates, instanceScopes, dominantCode)
	walletType := exploreWalletTypeLabel(req)
	binding := exploreBindingLabel(req)

	o.metrics.IncExploreNoDeployableCandidate(dominantCode, walletType, binding, missingCountLabel)
	if logger := o.logger; logger != nil {
		o.logNoDeployableCandidate(logger, r, req, decision, instanceScopes, dominantCode, missingCountLabel, walletType, binding)
	}
}

func dominantExploreRejectionCode(rejected []policy.RejectedPolicy) string {
	codes := collectBlockingRejectionCodes(rejected)
	if len(codes) == 0 {
		return exploreLabelUnknown
	}
	for _, code := range codes {
		if code == rejectionCodeChainScope {
			return rejectionCodeChainScope
		}
	}
	sort.Strings(codes)
	return codes[0]
}

func collectBlockingRejectionCodes(rejected []policy.RejectedPolicy) []string {
	seen := make(map[string]struct{})
	var codes []string
	for _, candidate := range rejected {
		for _, finding := range candidate.RejectionReasons {
			if finding.Code == "" {
				continue
			}
			if finding.Severity != "" && finding.Severity != policy.AssessmentFindingSeverityBlocking {
				continue
			}
			if _, ok := seen[finding.Code]; ok {
				continue
			}
			seen[finding.Code] = struct{}{}
			codes = append(codes, finding.Code)
		}
	}
	return codes
}

func collectAllRejectionCodes(rejected []policy.RejectedPolicy) []string {
	seen := make(map[string]struct{})
	var codes []string
	for _, candidate := range rejected {
		for _, finding := range candidate.RejectionReasons {
			if finding.Code == "" {
				continue
			}
			if _, ok := seen[finding.Code]; ok {
				continue
			}
			seen[finding.Code] = struct{}{}
			codes = append(codes, finding.Code)
		}
	}
	sort.Strings(codes)
	return codes
}

func bucketMissingChainCount(
	targetChainIDs []int64,
	rejected []policy.RejectedPolicy,
	instanceScopes map[string][]int64,
	dominantCode string,
) string {
	if dominantCode != rejectionCodeChainScope {
		return exploreLabelUnknown
	}
	minMissing, ok := minMissingChainCountForChainScopeRejections(targetChainIDs, rejected, instanceScopes)
	if !ok {
		return exploreLabelUnknown
	}
	return bucketMissingChainCountValue(minMissing)
}

func bucketMissingChainCountValue(count int) string {
	switch {
	case count <= 0:
		return "0"
	case count == 1:
		return "1"
	case count == 2:
		return "2"
	case count == 3:
		return "3"
	default:
		return "4_plus"
	}
}

func minMissingChainCountForChainScopeRejections(
	targetChainIDs []int64,
	rejected []policy.RejectedPolicy,
	instanceScopes map[string][]int64,
) (int, bool) {
	if len(targetChainIDs) == 0 || len(instanceScopes) == 0 {
		return 0, false
	}

	found := false
	minMissing := 0
	for _, candidate := range rejected {
		if !rejectedCandidateHasCode(candidate, rejectionCodeChainScope) {
			continue
		}
		scope, ok := instanceScopes[candidate.CryptoPolicyInstanceID]
		if !ok {
			continue
		}
		missing := countMissingTargetChains(targetChainIDs, scope)
		if !found || missing < minMissing {
			minMissing = missing
			found = true
		}
	}
	return minMissing, found
}

func rejectedCandidateHasCode(candidate policy.RejectedPolicy, code string) bool {
	for _, finding := range candidate.RejectionReasons {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func countMissingTargetChains(targetChainIDs, scopeChainIDs []int64) int {
	scopeSet := make(map[int64]struct{}, len(scopeChainIDs))
	for _, id := range scopeChainIDs {
		if id > 0 {
			scopeSet[id] = struct{}{}
		}
	}
	missing := 0
	seen := make(map[int64]struct{}, len(targetChainIDs))
	for _, id := range targetChainIDs {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := scopeSet[id]; !ok {
			missing++
		}
	}
	return missing
}

func missingTargetChains(targetChainIDs, scopeChainIDs []int64) []int64 {
	scopeSet := make(map[int64]struct{}, len(scopeChainIDs))
	for _, id := range scopeChainIDs {
		if id > 0 {
			scopeSet[id] = struct{}{}
		}
	}
	var missing []int64
	seen := make(map[int64]struct{}, len(targetChainIDs))
	for _, id := range targetChainIDs {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := scopeSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	sort.Slice(missing, func(i, j int) bool { return missing[i] < missing[j] })
	return missing
}

func exploreBindingLabel(req *decisionExploreRequest) string {
	if req == nil {
		return exploreBindingUnknown
	}
	if strings.TrimSpace(req.ScanID) != "" {
		return exploreBindingDiscovery
	}
	if req.PolicyContext != nil {
		if strings.TrimSpace(req.PolicyContext.ScanID) != "" {
			return exploreBindingDiscovery
		}
		if strings.TrimSpace(req.PolicyContext.Status) != "" {
			return exploreBindingDiscovery
		}
	}
	return exploreBindingUnknown
}

func exploreWalletTypeLabel(req *decisionExploreRequest) string {
	if req == nil || req.PolicyContext == nil {
		return exploreLabelUnknown
	}
	kind := normalizeWireAccountKind(req.PolicyContext.WalletType)
	if strings.TrimSpace(kind) == "" {
		return exploreLabelUnknown
	}
	return kind
}

func exploreWalletAddressHash(pc *walletPolicyContextWire) string {
	if pc == nil {
		return ""
	}
	raw := strings.TrimSpace(pc.TargetAddress)
	if raw == "" {
		raw = strings.TrimSpace(pc.WalletAddress)
	}
	if raw == "" {
		return ""
	}
	norm, err := persistence.NormalizeWalletTargetAddress(raw)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])[:16]
}

func (o exploreObservability) logNoDeployableCandidate(
	logger exploreLogger,
	r *http.Request,
	req *decisionExploreRequest,
	decision policy.PolicyDecision,
	instanceScopes map[string][]int64,
	dominantCode string,
	missingCountLabel string,
	walletType string,
	binding string,
) {
	targetChainIDs := decision.RequestSummary.TargetChainIDs
	observedChainIDs := decision.ObservedWalletSummary.ChainIDs
	missingChainIDs, candidateChainIDs := closestChainScopeMissingChains(targetChainIDs, decision.RejectedCandidates, instanceScopes)

	fields := []string{
		"event=" + strconv.Quote(exploreNoDeployableEventName),
		"dominant_rejection_code=" + strconv.Quote(dominantCode),
		"wallet_type=" + strconv.Quote(walletType),
		"binding=" + strconv.Quote(binding),
		"missing_chain_count_bucket=" + strconv.Quote(missingCountLabel),
		"rejected_candidates_count=" + strconv.Itoa(len(decision.RejectedCandidates)),
	}
	if req != nil && req.PolicyContext != nil {
		if hash := exploreWalletAddressHash(req.PolicyContext); hash != "" {
			fields = append(fields, "wallet_address_hash="+strconv.Quote(hash))
		}
		if scanID := strings.TrimSpace(req.ScanID); scanID != "" {
			fields = append(fields, "scan_id="+strconv.Quote(scanID))
		} else if scanID := strings.TrimSpace(req.PolicyContext.ScanID); scanID != "" {
			fields = append(fields, "scan_id="+strconv.Quote(scanID))
		}
	}
	if len(targetChainIDs) > 0 {
		fields = append(fields, "requested_chain_ids="+strconv.Quote(formatInt64Slice(targetChainIDs)))
	}
	if len(observedChainIDs) > 0 {
		fields = append(fields, "observed_chain_ids="+strconv.Quote(formatInt64Slice(observedChainIDs)))
	}
	if len(candidateChainIDs) > 0 {
		fields = append(fields, "candidate_chain_ids="+strconv.Quote(formatInt64Slice(candidateChainIDs)))
	}
	if len(missingChainIDs) > 0 {
		fields = append(fields, "missing_chain_ids="+strconv.Quote(formatInt64Slice(missingChainIDs)))
	}
	if codes := collectAllRejectionCodes(decision.RejectedCandidates); len(codes) > 0 {
		fields = append(fields, "rejection_codes="+strconv.Quote(strings.Join(codes, ",")))
	}
	for _, rejected := range decision.RejectedCandidates {
		if rejected.CryptoPolicyInstanceID != "" {
			fields = append(fields, "candidate_instance_id="+strconv.Quote(rejected.CryptoPolicyInstanceID))
		}
		if rejected.TemplateID != "" {
			fields = append(fields, "candidate_template_id="+strconv.Quote(rejected.TemplateID))
		}
	}
	if r != nil {
		if requestID, ok := sanitizeExploreRequestID(r.Header.Get("X-Request-Id")); ok {
			fields = append(fields, "request_id="+strconv.Quote(requestID))
		}
	}
	logger.Println(strings.Join(fields, " "))
}

func closestChainScopeMissingChains(
	targetChainIDs []int64,
	rejected []policy.RejectedPolicy,
	instanceScopes map[string][]int64,
) (missingChainIDs []int64, candidateChainIDs []int64) {
	if len(targetChainIDs) == 0 {
		return nil, nil
	}
	minMissing, ok := minMissingChainCountForChainScopeRejections(targetChainIDs, rejected, instanceScopes)
	if !ok {
		return nil, nil
	}
	for _, candidate := range rejected {
		if !rejectedCandidateHasCode(candidate, rejectionCodeChainScope) {
			continue
		}
		scope, scopeOK := instanceScopes[candidate.CryptoPolicyInstanceID]
		if !scopeOK {
			continue
		}
		missing := countMissingTargetChains(targetChainIDs, scope)
		if missing != minMissing {
			continue
		}
		return missingTargetChains(targetChainIDs, scope), append([]int64(nil), scope...)
	}
	return nil, nil
}

func formatInt64Slice(ids []int64) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

func sanitizeExploreRequestID(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > 128 {
		return "", false
	}
	for _, ch := range value {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == ':' || ch == '-' {
			continue
		}
		return "", false
	}
	return value, true
}

func instanceScopeByID(instances []*policy.CryptoPolicyInstance) map[string][]int64 {
	scopes := make(map[string][]int64, len(instances))
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		scopes[inst.ID] = append([]int64(nil), inst.Scope.ChainIDs...)
	}
	return scopes
}

// setExploreObservabilityForTest swaps the package-level explore observability sink (tests only).
func setExploreObservabilityForTest(obs exploreObservability) func() {
	prev := exploreObs
	exploreObs = obs
	return func() { exploreObs = prev }
}

type testExploreMetrics struct {
	increments []testExploreMetricIncrement
}

type testExploreMetricIncrement struct {
	RejectionCode      string
	WalletType         string
	Binding            string
	MissingChainCount  string
}

func (m *testExploreMetrics) IncExploreNoDeployableCandidate(rejectionCode, walletType, binding, missingChainCount string) {
	m.increments = append(m.increments, testExploreMetricIncrement{
		RejectionCode:     rejectionCode,
		WalletType:        walletType,
		Binding:           binding,
		MissingChainCount: missingChainCount,
	})
}

type testExploreLogger struct {
	lines []string
}

func (l *testExploreLogger) Println(v ...any) {
	if len(v) == 0 {
		return
	}
	l.lines = append(l.lines, fmt.Sprint(v...))
}
