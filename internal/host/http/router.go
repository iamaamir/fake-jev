// Package http provides the HTTP host for fake-jev compatibility profiles.
package http

import (
	"net/http"
	"strings"

	"fake-jev/internal/compat/jev/v1"
	"fake-jev/internal/engine"
)

// Router binds the active compatibility profile to the provider-neutral engine.
// It performs no HTTP response writing; that remains the server's concern.
type Router struct {
	engine  *engine.Engine
	profile *v1.JevV1Profile
}

// NewRouter creates a router for the jev/v1 profile.
func NewRouter(engineState *engine.Engine, profile *v1.JevV1Profile) *Router {
	if profile == nil {
		profile = v1.DefaultProfile()
	}
	return &Router{engine: engineState, profile: profile}
}

// NewDefaultRouter creates a router using the built-in jev/v1 profile.
func NewDefaultRouter(engineState *engine.Engine) *Router {
	return NewRouter(engineState, v1.DefaultProfile())
}

func (r *Router) active() bool { return r != nil && r.engine != nil && r.profile != nil }

func (r *Router) route(request *http.Request) v1.Route {
	if !r.active() || request == nil || request.URL == nil {
		return v1.UnknownRoute
	}
	return v1.RecognizeRoute(request.Method, request.URL.RequestURI())
}

func (r *Router) isControl(request *http.Request) bool {
	return request != nil && request.URL != nil && strings.HasPrefix(request.URL.Path, "/__fake/")
}

func (r *Router) profileID() string {
	if r == nil || r.profile == nil {
		return "jev/v1"
	}
	return r.profile.ID()
}
