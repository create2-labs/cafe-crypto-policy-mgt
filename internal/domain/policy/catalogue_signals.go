package policy

import (
	"log"
	"strings"

	"github.com/create2-labs/cafe-crypto-policy-mgt/internal/domain/provider"
)

// catalogueSignalsLogger is the minimal logger for catalogue startup signals (CPM-P11a).
type catalogueSignalsLogger interface {
	Printf(format string, v ...any)
}

// CheckPostureOrphanage reports ADR §7.2.1 posture-only orphanage for each CP:
// empty allowed_providers, or no allowed provider profile with
// resulting_posture == required_posture. Intentionally ignores chain status
// (planned-only) and wallet type — those remain explore couche A rejections.
// Returns the number of orphan CPs logged.
func CheckPostureOrphanage(cps []*CryptoPolicy, reg *provider.Registry, logger catalogueSignalsLogger) int {
	if logger == nil {
		logger = log.Default()
	}
	n := 0
	for _, cp := range cps {
		if cp == nil {
			continue
		}
		if hasPostureCompatibleProvider(cp, reg) {
			continue
		}
		n++
		logger.Printf(
			"WARN catalogue: posture orphanage crypto_policy_id=%s required_posture=%s allowed_providers=%v",
			cp.ID, cp.RequiredPosture, cp.AllowedProviders,
		)
	}
	return n
}

func hasPostureCompatibleProvider(cp *CryptoPolicy, reg *provider.Registry) bool {
	if cp == nil {
		return false
	}
	if len(cp.AllowedProviders) == 0 {
		return false
	}
	required := strings.TrimSpace(string(cp.RequiredPosture))
	if required == "" || reg == nil {
		return false
	}
	for _, providerID := range cp.AllowedProviders {
		providerID = strings.TrimSpace(providerID)
		if providerID == "" {
			continue
		}
		for _, resolved := range reg.ProfilesForProvider(providerID) {
			if resolved == nil {
				continue
			}
			if strings.TrimSpace(resolved.Profile.ResultingPosture) == required {
				return true
			}
		}
	}
	return false
}
