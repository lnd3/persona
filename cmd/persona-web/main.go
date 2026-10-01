// persona-web serves persona's public landing page (site/index.html)
// and reports page views to wisp (github.com/lnd3/wisp), the product
// group's cookieless analytics service — see plan/actions/A010.
//
// It exists for exactly one reason: wisp's hook must run inside the
// code actually handling a visitor's request, and persona previously
// had no such code — persona-caddy served site/index.html directly
// via its own file_server, so no persona process ever saw a request.
// This binary is deliberately as small as the static page it serves;
// persona-caddy still terminates TLS and reverse-proxies to it (see
// deploy/persona-caddy/Caddyfile), nothing else changes about the
// deployment shape.
package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/lnd3/wisp/hook"
)

func main() {
	addr := envOr("PERSONA_WEB_ADDR", ":8080")
	siteDir := envOr("PERSONA_SITE_DIR", "/srv/site")

	clientIP, err := trustedProxyClientIP(os.Getenv("PERSONA_EDGE_SUBNET"))
	if err != nil {
		log.Fatalf("persona-web: %v", err)
	}

	w, err := hook.Start(hook.Config{
		Endpoint:   os.Getenv("WISP_ENDPOINT"), // empty => disabled, no-op hook (dev/local runs)
		ProductKey: "persona",
		Token:      os.Getenv("WISP_TOKEN"),
		ClientIP:   clientIP,
		OnError: func(err error) {
			// Never contains request data (IPs, User-Agents, keys) per
			// hook's own doc comment — safe to log plainly.
			log.Printf("persona-web: wisp hook: %v", err)
		},
	})
	if err != nil {
		log.Fatalf("persona-web: starting wisp hook: %v", err)
	}

	indexPath := filepath.Join(siteDir, "index.html")
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(rw, r)
			return
		}
		rec := &statusRecorder{ResponseWriter: rw, status: http.StatusOK}
		http.ServeFile(rec, r, indexPath)
		// Only a real, successful page view counts — not a 404, not a
		// 304 conditional-GET — per A010's own "after a successful page
		// response" rule. "/" is already a route template (persona has
		// exactly one real page today), never a raw path.
		if rec.status == http.StatusOK {
			w.View(r, "/")
		}
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("persona-web: ListenAndServe: %v", err)
		}
	}()
	log.Printf("persona-web: listening on %s, serving %s", addr, siteDir)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("persona-web: shutdown: %v", err)
	}
	if err := w.Close(ctx); err != nil {
		log.Printf("persona-web: wisp hook close: %v", err)
	}
}

// statusRecorder captures the status code a handler actually sent, so
// the caller can decide whether to count a wisp view — net/http gives
// no other way to observe what http.ServeFile decided (200, 404, 304,
// 412, ...) without wrapping ResponseWriter.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// trustedProxyClientIP returns a hook.Config.ClientIP implementation
// that trusts the X-Real-IP header set by persona-caddy
// (deploy/persona-caddy/Caddyfile's own `header_up X-Real-IP
// {http.request.remote.host}`) only when the request's own RemoteAddr
// falls inside trustedCIDR — persona-web and persona-caddy share the
// same docker network (persona-edge, see deploy/docker-compose.yml),
// so a request's RemoteAddr there is always persona-caddy's own
// container IP; nothing else on that network can forge it. Falls back
// to RemoteAddr itself (hook's own default behavior) for any request
// that didn't arrive via persona-caddy, or if trustedCIDR is empty
// (e.g. local/dev runs with no proxy in front at all).
func trustedProxyClientIP(trustedCIDR string) (func(r *http.Request) string, error) {
	var trustedNet *net.IPNet
	if trustedCIDR != "" {
		_, n, err := net.ParseCIDR(trustedCIDR)
		if err != nil {
			return nil, err
		}
		trustedNet = n
	}
	return func(r *http.Request) string {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		if trustedNet != nil {
			if remote := net.ParseIP(host); remote != nil && trustedNet.Contains(remote) {
				if xri := r.Header.Get("X-Real-IP"); xri != "" {
					return xri
				}
			}
		}
		return host
	}, nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
