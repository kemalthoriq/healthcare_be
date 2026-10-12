package clientip

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ipKey struct{}

func Middleware(trustedHops int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := Resolve(r, trustedHops)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ipKey{}, ip)))
		})
	}
}

func FromContext(ctx context.Context) string {
	ip, _ := ctx.Value(ipKey{}).(string)
	return ip
}

func Resolve(r *http.Request, trustedHops int) string {
	remote := remoteHost(r.RemoteAddr)
	if trustedHops <= 0 {
		return remote
	}

	var chain []string
	for _, h := range r.Header.Values("X-Forwarded-For") {
		for part := range strings.SplitSeq(h, ",") {
			chain = append(chain, strings.TrimSpace(part))
		}
	}
	chain = append(chain, remote)

	i := max(len(chain)-1-trustedHops, 0)
	addr, err := netip.ParseAddr(chain[i])
	if err != nil {
		return remote
	}
	return addr.Unmap().String()
}

func remoteHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}
