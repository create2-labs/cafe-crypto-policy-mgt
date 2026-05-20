package contract

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestScripts_noLegacyWalletPolicyContextsRoute(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	script := filepath.Join(repoRoot, "scripts", "ci", "assert-scripts-use-discovery-wallet-scans-v1-only.sh")
	cmd := exec.Command("bash", script)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("assert script failed: %v\n%s", err, out)
	}
	if len(out) == 0 {
		t.Fatal("expected script output")
	}
}
