package middleware

import (
	"github.com/julienschmidt/httprouter"
)

// Middleware ...
type Middleware = func(httprouter.Handle) httprouter.Handle

// Use ...
func Use(handler httprouter.Handle, middlewares ...Middleware) httprouter.Handle {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}
