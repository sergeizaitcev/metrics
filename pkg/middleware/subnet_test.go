package middleware_test

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/require"

	"github.com/sergeizaitcev/metrics/pkg/middleware"
)

func TestSubnet(t *testing.T) {
	testCases := []struct {
		name   string
		ip     net.IP
		subnet *net.IPNet
		want   int
	}{
		{
			name: "ipv4",
			ip:   net.ParseIP("127.0.0.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			want: http.StatusOK,
		},
		{
			name: "ipv6",
			ip:   net.ParseIP("::ffff:127.0.0.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("::ffff:127.0.0.1/124")
				return subnet
			}(),
			want: http.StatusOK,
		},
		{
			name: "empty",
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			want: http.StatusForbidden,
		},
		{
			name: "no contains",
			ip:   net.ParseIP("127.0.1.1"),
			subnet: func() *net.IPNet {
				_, subnet, _ := net.ParseCIDR("127.0.0.1/24")
				return subnet
			}(),
			want: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			next := func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
				w.WriteHeader(http.StatusOK)
			}

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.Header.Set(middleware.IPHeader, tc.ip.String())

			subnet := middleware.Use(next, middleware.Subnet(tc.subnet))
			subnet(rec, req, httprouter.Params{})

			require.Equal(t, tc.want, rec.Code)
		})
	}
}
