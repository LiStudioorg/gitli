// SPDX-License-Identifier: AGPL-3.0
package i18n

import (
	"net/http"
	"testing"
)

func TestFromRequest(t *testing.T) {
	cases := map[string]Lang{
		"zh-CN,zh;q=0.9,en;q=0.8": ZH,
		"en-US,en;q=0.9":          EN,
		"en-GB":                   EN,
		"":                        ZH,
		"fr-FR":                   ZH,
	}
	for header, want := range cases {
		r, _ := http.NewRequest("GET", "/", nil)
		if header != "" {
			r.Header.Set("Accept-Language", header)
		}
		if got := FromRequest(r); got != want {
			t.Fatalf("%q -> %s, want %s", header, got, want)
		}
	}
}

func TestT(t *testing.T) {
	if T(EN, "forbidden") != "insufficient permissions" {
		t.Fatal("english message mismatch")
	}
	if T(ZH, "forbidden") != "权限不足" {
		t.Fatal("chinese message mismatch")
	}
	if T(EN, "unknown.key") != "unknown.key" {
		t.Fatal("missing key should return key")
	}
}
