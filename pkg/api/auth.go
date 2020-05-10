package api

import (
	"context"
	"github.com/virtual-vgo/vvgo/pkg/login"
	"github.com/virtual-vgo/vvgo/pkg/tracing"
	"net/http"
	"strings"
)

// Authenticate http requests using the sessions api
// If the request has a valid session or token with the required role, it is allowed access.
type RBACMux struct {
	Basic    map[[2]string][]login.Role
	Bearer   map[string][]login.Role
	Sessions *login.Store
	*http.ServeMux
}

// HandleFunc registers the handler function for the given pattern.
func (auth *RBACMux) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request), role login.Role) {
	auth.Handle(pattern, http.HandlerFunc(handler), role)
}
func (auth *RBACMux) Handle(pattern string, handler http.Handler, role login.Role) {
	// anonymous access goes directly to the mux
	if role == login.RoleAnonymous {
		auth.ServeMux.Handle(pattern, handler)
		return
	}

	auth.ServeMux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracing.StartSpan(r.Context(), "rbac_mux")
		defer span.Send()

		var identity login.Identity

		// check for basic auth
		switch {
		case auth.readBasicAuth(r, &identity):
		case auth.readBearer(r, &identity):
		case auth.readSession(ctx, r, &identity):
		default:
			identity = login.Anonymous()
		}

		if identity.HasRole(role) {
			handler.ServeHTTP(w, r.Clone(context.WithValue(ctx, "vvgo_identity", &identity)))
		} else {
			unauthorized(w)
		}
	})
}

func (auth *RBACMux) readBasicAuth(r *http.Request, dest *login.Identity) bool {
	user, pass, _ := r.BasicAuth()
	gotRoles, ok := auth.Basic[[2]string{user, pass}]
	if !ok {
		return false
	}

	*dest = login.Identity{
		Kind:  login.KindBasic,
		Roles: gotRoles,
	}
	return true
}

func (auth *RBACMux) readBearer(r *http.Request, dest *login.Identity) bool {
	bearer := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(bearer, "Bearer ") {
		return false
	}
	bearer = bearer[len("Bearer "):]

	gotRoles, ok := auth.Bearer[bearer]
	if !ok {
		return false
	}

	*dest = login.Identity{
		Kind:  login.KindBearer,
		Roles: gotRoles,
	}
	return true
}

func (auth *RBACMux) readSession(ctx context.Context, r *http.Request, dest *login.Identity) bool {
	return auth.Sessions.ReadSessionFromRequest(ctx, r, dest) != nil
}
