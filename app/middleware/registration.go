package middleware

import "net/http"

// registrationMiddleware holds all middleware functions to be applied to the server.
// Note that the order of the slice is the order in which the middleware will be applied.
var registrationMiddleware = []func(http.Handler) http.Handler{
	LoggingMiddleware,
	CSRFMiddleware,
}

func GetAllRegisteredMiddleware() []func(http.Handler) http.Handler {
	return registrationMiddleware
}
