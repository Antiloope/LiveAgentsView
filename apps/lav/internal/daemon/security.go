package daemon

import (
	"log"
	"mime"
	"net/http"
)

// clientHeader is required on every state-changing request. It exists only
// to force a CORS preflight: a cross-origin fetch cannot set a custom
// header without one, and the daemon answers no preflight affirmatively
// (see secure below), so a hostile page can never send it.
const clientHeader = "X-LAV-Client"

// secureConfig is the daemon's own loopback identity — the Host and Origin
// values a legitimate request can carry — computed once from the port it is
// actually listening on, since LAV_PORT can change it.
type secureConfig struct {
	hosts   map[string]bool
	origins map[string]bool
}

func newSecureConfig(port string) secureConfig {
	hosts := map[string]bool{
		"127.0.0.1:" + port: true,
		"localhost:" + port: true,
		"[::1]:" + port:     true,
	}
	origins := make(map[string]bool, len(hosts))
	for host := range hosts {
		origins["http://"+host] = true
	}
	return secureConfig{hosts: hosts, origins: origins}
}

// secure wraps next so that only the daemon's own frontend and its own CLI
// can reach any route — no website should be able to make this daemon
// launch a process. Applied once around the whole mux rather than per
// handler, so a route registered later is covered without anyone
// remembering to guard it. Checks run cheapest and most decisive first;
// none of them is expensive enough to choose further between.
func secure(cfg secureConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && !cfg.origins[origin] {
			reject(w, r, http.StatusForbidden, "cross-origin request, Origin: "+origin)
			return
		}
		if sfs := r.Header.Get("Sec-Fetch-Site"); sfs != "" && sfs != "same-origin" && sfs != "none" {
			reject(w, r, http.StatusForbidden, "cross-site request, Sec-Fetch-Site: "+sfs)
			return
		}
		if !cfg.hosts[r.Host] {
			reject(w, r, http.StatusForbidden, "unrecognized Host: "+r.Host)
			return
		}
		// r.ContentLength is 0 only when the request definitely carries no
		// body; -1 (length unknown, e.g. chunked) is treated as "has one" so
		// it cannot be used to smuggle a body past this check.
		if r.ContentLength != 0 {
			ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if ct != "application/json" {
				reject(w, r, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if r.Header.Get(clientHeader) == "" {
				reject(w, r, http.StatusForbidden, "missing "+clientHeader+" header")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func reject(w http.ResponseWriter, r *http.Request, status int, reason string) {
	log.Printf("lav: rejected %s %s (%s): %s", r.Method, r.URL.Path, r.RemoteAddr, reason)
	http.Error(w, reason, status)
}
