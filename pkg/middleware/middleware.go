package middleware

import (
	"github.com/julienschmidt/httprouter"
)

// Middleware определяет функцию-обёртку для функции-обработчика.
type Middleware = func(httprouter.Handle) httprouter.Handle

// Use оборачивает функцию-обработчика функциями-обёртками.
func Use(handler httprouter.Handle, middlewares ...Middleware) httprouter.Handle {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}
