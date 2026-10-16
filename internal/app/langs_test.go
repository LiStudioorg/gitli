// SPDX-License-Identifier: AGPL-3.0
package app

import "testing"

func TestParseLanguageRefreshPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    int64
		wantErr bool
	}{
		{name: "valid", payload: "42", want: 42},
		{name: "invalid", payload: "abc", wantErr: true},
		{name: "empty", payload: "", wantErr: true},
		{name: "zero", payload: "0", wantErr: true},
		{name: "negative", payload: "-3", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLanguageRefreshPayload(tt.payload)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.payload)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %d, want %d", got, tt.want)
			}
		})
	}
}
