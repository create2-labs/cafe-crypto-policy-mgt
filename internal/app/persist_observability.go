package app

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/metrics"
)

const (
	persistUserConstraintsIncompatibleEvent = "cpm.persist.user_constraints_incompatible"
	adrSignalNoProviderAfterUserConstraints = "runtime.no_provider_after_user_constraints"
)

type persistMetricsRecorder interface {
	IncPersistUserConstraintsIncompatible()
}

type prometheusPersistMetrics struct{}

func (prometheusPersistMetrics) IncPersistUserConstraintsIncompatible() {
	metrics.IncPersistUserConstraintsIncompatible()
}

type persistLogger interface {
	Println(v ...any)
}

type persistObservability struct {
	logger  persistLogger
	metrics persistMetricsRecorder
}

var persistObs = persistObservability{
	logger:  log.Default(),
	metrics: prometheusPersistMetrics{},
}

func recordPersistUserConstraintsIncompatible(r *http.Request, draftID, cryptoPolicyID, findingCode string) {
	persistObs.recordUserConstraintsIncompatible(r, draftID, cryptoPolicyID, findingCode)
}

func (o persistObservability) recordUserConstraintsIncompatible(
	r *http.Request,
	draftID, cryptoPolicyID, findingCode string,
) {
	o.metrics.IncPersistUserConstraintsIncompatible()
	if o.logger == nil {
		return
	}
	if findingCode == "" {
		findingCode = "unknown"
	}
	fields := []string{
		"event=" + strconv.Quote(persistUserConstraintsIncompatibleEvent),
		"adr_signal=" + strconv.Quote(adrSignalNoProviderAfterUserConstraints),
		"finding_code=" + strconv.Quote(findingCode),
	}
	if draftID = strings.TrimSpace(draftID); draftID != "" {
		fields = append(fields, "draft_id="+strconv.Quote(draftID))
	}
	if cryptoPolicyID = strings.TrimSpace(cryptoPolicyID); cryptoPolicyID != "" {
		fields = append(fields, "crypto_policy_id="+strconv.Quote(cryptoPolicyID))
	}
	if r != nil {
		if rid := strings.TrimSpace(r.Header.Get("X-Request-Id")); rid != "" {
			fields = append(fields, "request_id="+strconv.Quote(rid))
		}
	}
	o.logger.Println(strings.Join(fields, " "))
}
