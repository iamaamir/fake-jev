// Package v1 implements the frozen jev/v1 compatibility boundary.
package v1

import "strings"

const (
	ModelsPath    = "/v1/models"
	SystemOnePath = "/v1/systemone"
)

// Route identifies a recognized jev/v1 data-plane operation.
type Route uint8

const (
	UnknownRoute Route = iota
	ModelsRoute
	SystemOneRoute
)

// Operation is the engine-facing name of a recognized route.
func (r Route) Operation() string {
	switch r {
	case ModelsRoute:
		return "models"
	case SystemOneRoute:
		return "systemone"
	default:
		return ""
	}
}

// RecognizeRoute recognizes only the exact method/path pairs in §39.8. The
// query string is deliberately ignored, while trailing slashes are not.
func RecognizeRoute(method, requestTarget string) Route {
	path := requestTarget
	if query := strings.IndexByte(path, '?'); query >= 0 {
		path = path[:query]
	}
	switch {
	case method == "GET" && path == ModelsPath:
		return ModelsRoute
	case method == "POST" && path == SystemOnePath:
		return SystemOneRoute
	default:
		return UnknownRoute
	}
}

// MatchRoute is the boolean form of RecognizeRoute.
func MatchRoute(method, requestTarget string) (Route, bool) {
	route := RecognizeRoute(method, requestTarget)
	return route, route != UnknownRoute
}

// RouteFor is an alias useful to callers that treat routing as a lookup.
func RouteFor(method, requestTarget string) Route {
	return RecognizeRoute(method, requestTarget)
}

// UnknownRouteResponse is the deterministic data-plane 404 body for an
// unrecognized route. Routing callers may serialize it as JSON.
type UnknownRouteResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func NewUnknownRouteResponse() UnknownRouteResponse {
	return UnknownRouteResponse{
		Error:   "fake_jev_unknown_route",
		Message: "No active compatibility profile handles this route.",
	}
}
