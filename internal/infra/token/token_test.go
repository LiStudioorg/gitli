// SPDX-License-Identifier: AGPL-3.0
package token

import "testing"

func TestNewAndVerify(t *testing.T) {
	tok, err := New("glpat")
	if err != nil {
		t.Fatal(err)
	}
	if tok.Plain == "" || tok.Hash == "" {
		t.Fatal("empty token fields")
	}
	if !Verify(tok.Plain, tok.Hash) {
		t.Fatal("verify should succeed")
	}
	if Verify("other", tok.Hash) {
		t.Fatal("verify should fail for wrong token")
	}
}

func TestUniqueness(t *testing.T) {
	a, _ := New("")
	b, _ := New("")
	if a.Plain == b.Plain {
		t.Fatal("tokens should be unique")
	}
}
