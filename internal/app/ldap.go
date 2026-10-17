// SPDX-License-Identifier: AGPL-3.0
package app

import (
	"context"

	"github.com/LiStudioorg/gitli/internal/infra/ldapauth"
	"github.com/LiStudioorg/gitli/internal/service/auth"
)

// ldapAdapter adapts ldapauth.Service to auth.LDAPAuthenticator.
type ldapAdapter struct {
	svc *ldapauth.Service
}

// newLDAPAdapter returns nil when LDAP is disabled.
func newLDAPAdapter(svc *ldapauth.Service) auth.LDAPAuthenticator {
	if svc == nil || !svc.Enabled() {
		return nil
	}
	return &ldapAdapter{svc: svc}
}

func (a *ldapAdapter) Authenticate(ctx context.Context, username, password string) (string, string, string, error) {
	u, err := a.svc.Authenticate(ctx, username, password)
	if err != nil {
		return "", "", "", err
	}
	return u.Username, u.Email, u.FullName, nil
}
