package app

import "testing"

func TestPolicyPayloadRequiresEOAWalletProof(t *testing.T) {
	cases := []struct {
		name    string
		payload map[string]any
		want    bool
	}{
		{
			name: "legacy selected_wallet_policy_context",
			payload: map[string]any{
				"selected_wallet_policy_context": map[string]any{
					"wallet_address": "0xabc",
				},
			},
			want: true,
		},
		{
			name: "policy_context eoa",
			payload: map[string]any{
				"policy_context": map[string]any{
					"wallet_type": "eoa",
				},
			},
			want: true,
		},
		{
			name:    "fixture payload without wallet",
			payload: map[string]any{"mode": "strict"},
			want:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := policyPayloadRequiresEOAWalletProof(tc.payload); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
