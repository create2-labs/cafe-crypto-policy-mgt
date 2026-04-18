package vocabulary

import "strings"

// AlgorithmID identifies the observed signing / verification algorithm using exported strings.
// Well-known values are constants; hybrid profiles use the "hybrid_*" prefix pattern.
type AlgorithmID string

const (
	AlgorithmSecp256k1ECRecover AlgorithmID = "secp256k1_ecrecover"
	AlgorithmMLDSA44            AlgorithmID = "mldsa44"
	AlgorithmMLDSA65            AlgorithmID = "mldsa65"
	AlgorithmFalcon512          AlgorithmID = "falcon512"
)

const hybridAlgorithmPrefix = "hybrid_"

// IsValidAlgorithmID reports whether id is an allowed exported algorithm identifier:
// a well-known constant or any non-empty string with prefix "hybrid_".
func IsValidAlgorithmID(id string) bool {
	if id == "" {
		return false
	}
	switch AlgorithmID(id) {
	case AlgorithmSecp256k1ECRecover, AlgorithmMLDSA44, AlgorithmMLDSA65, AlgorithmFalcon512:
		return true
	default:
		return strings.HasPrefix(id, hybridAlgorithmPrefix) && len(id) > len(hybridAlgorithmPrefix)
	}
}
