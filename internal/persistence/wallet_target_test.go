package persistence

import "testing"

func TestNormalizeWalletTargetAddress_canonical(t *testing.T) {
	const want = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	cases := []string{
		"0x742d35cc6634c0532925a3b844bc454e4438f44e",
		"0X742d35Cc6634C0532925a3b844Bc454e4438f44e",
		"742d35cc6634c0532925a3b844bc454e4438f44e",
	}
	for _, in := range cases {
		got, err := NormalizeWalletTargetAddress(in)
		if err != nil {
			t.Fatalf("NormalizeWalletTargetAddress(%q): %v", in, err)
		}
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestWalletSubjectIDFromAddress_andNormalizeWalletSubjectID(t *testing.T) {
	const addr = "0x742d35cc6634c0532925a3b844bc454e4438f44e"
	const want = WalletSubjectPrefix + addr

	subject, err := WalletSubjectIDFromAddress("0X742d35Cc6634C0532925a3b844Bc454e4438f44e")
	if err != nil {
		t.Fatalf("WalletSubjectIDFromAddress: %v", err)
	}
	if subject != want {
		t.Fatalf("WalletSubjectIDFromAddress = %q, want %q", subject, want)
	}

	cases := map[string]string{
		"wallet:0x742d35Cc6634C0532925a3b844Bc454e4438f44e": want,
		"wallet:" + addr: want,
		addr: want,
		"tls:cluster-1": "tls:cluster-1",
		"wallet:not-hex": "wallet:not-hex",
	}
	for in, expected := range cases {
		if got := NormalizeWalletSubjectID(in); got != expected {
			t.Fatalf("NormalizeWalletSubjectID(%q) = %q, want %q", in, got, expected)
		}
	}
}
