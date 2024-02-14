package middleware

import (
	"net"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

const IPHeader = "X-Real-IP"

// Subnet проверяет IP-адрес входящего запроса на вхождение в доверенную подсеть.
func Subnet(subnet *net.IPNet) Middleware {
	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
			ip := r.Header.Get(IPHeader)
			if ip == "" || !subnet.Contains(net.ParseIP(ip)) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next(w, r, p)
		}
	}
}
