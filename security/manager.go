package security

import (
	"context"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
)

type TokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

func NewManager(iss, aud string) (*Manager, error) {
	provider, err := oidc.NewProvider(context.TODO(), iss)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: aud})

	return &Manager{verifier}, nil
}

type Manager struct {
	TokenVerifier
}

func (a *Manager) AuthorizeMW(h http.Handler) http.Handler {
	return authorize(h.ServeHTTP, a)
}

func (a *Manager) Authorize(h http.HandlerFunc) http.HandlerFunc {
	return authorize(h, a)
}

type ctxKey struct{}

func authorize(h http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		authH := r.Header.Get("Authorization")
		if authH == "" {
			http.Error(rw, "Not authorized", http.StatusForbidden)
			return
		}

		token, err := verifier.Verify(r.Context(), authH[7:])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusForbidden)
			return
		}

		if h != nil {
			ctx := context.WithValue(r.Context(), ctxKey{}, token)
			h(rw, r.WithContext(ctx))
		}
	}
}

func Token(r *http.Request) *oidc.IDToken {
	t := r.Context().Value(ctxKey{})
	if t == nil {
		return nil
	}
	return t.(*oidc.IDToken)
}

type claims struct {
	Roles []string `json:"roles"`
	OID   string   `json:"oid"`
	TID   string   `json:"tid"`
}

func OnlyInRole(roles ...string) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			var c claims
			t := Token(r)
			if t == nil {
				http.Error(rw, "Unauthorized", http.StatusForbidden)
				return
			}

			if err := t.Claims(&c); err != nil {
				http.Error(rw, err.Error(), http.StatusForbidden)
				return
			}

			for _, assignedRole := range c.Roles {
				for _, wantedRole := range roles {
					if assignedRole == wantedRole {
						log.Printf("role(s) %v found in %v", wantedRole, c.Roles)
						if h != nil {
							h.ServeHTTP(rw, r)
						}
						return
					}
				}
			}
			http.Error(rw, "Unauthorized", http.StatusForbidden)
		})
	}
}
