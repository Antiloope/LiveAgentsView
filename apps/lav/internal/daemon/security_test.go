package daemon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testConfig() secureConfig { return newSecureConfig("8420") }

func newRequest(method, path string) *http.Request {
	r := httptest.NewRequest(method, "http://127.0.0.1:8420"+path, nil)
	r.Host = "127.0.0.1:8420"
	return r
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
}

// A route registered on the mux after secure() is wired must still be
// rejected, proving the middleware guards the whole mux rather than
// individual handlers that remember to call it.
func TestSecure_ProtectsRoutesAddedLater(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/throwaway-test-route", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	wrapped := secure(testConfig(), mux)

	req := newRequest(http.MethodPost, "/api/throwaway-test-route")
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("throwaway route with no client header: got %d, want 403", rec.Code)
	}
}

func TestSecure_RejectsCrossOrigin(t *testing.T) {
	req := newRequest(http.MethodGet, "/api/sessions")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", rec.Code)
	}
}

func TestSecure_AllowsOwnOrigins(t *testing.T) {
	for _, origin := range []string{"http://127.0.0.1:8420", "http://localhost:8420", "http://[::1]:8420"} {
		req := newRequest(http.MethodGet, "/api/sessions")
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Origin=%q: got %d, want 200", origin, rec.Code)
		}
	}
}

func TestSecure_RejectsCrossSiteSecFetchSite(t *testing.T) {
	for _, v := range []string{"cross-site", "same-site"} {
		req := newRequest(http.MethodGet, "/api/sessions")
		req.Header.Set("Sec-Fetch-Site", v)
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("Sec-Fetch-Site=%q: got %d, want 403", v, rec.Code)
		}
	}
}

func TestSecure_AllowsSameOriginOrAbsentSecFetchSite(t *testing.T) {
	for _, v := range []string{"same-origin", "none", ""} {
		req := newRequest(http.MethodGet, "/api/sessions")
		if v != "" {
			req.Header.Set("Sec-Fetch-Site", v)
		}
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Sec-Fetch-Site=%q: got %d, want 200", v, rec.Code)
		}
	}
}

// The DNS-rebinding case: a hostile page's own domain resolves to
// 127.0.0.1, so the browser believes it is same-origin and sends a Host
// the attacker's DNS chose rather than one the daemon recognizes.
func TestSecure_RejectsDNSRebindingHost(t *testing.T) {
	req := newRequest(http.MethodGet, "/api/sessions")
	req.Host = "evil.example"
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("Host: evil.example: got %d, want 403", rec.Code)
	}
}

func TestSecure_AllowsLoopbackHosts(t *testing.T) {
	for _, host := range []string{"127.0.0.1:8420", "localhost:8420", "[::1]:8420"} {
		req := newRequest(http.MethodGet, "/api/sessions")
		req.Host = host
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("Host=%q: got %d, want 200", host, rec.Code)
		}
	}
}

// The exact bypass the spec measured live: a text/plain body is a CORS
// simple request, so it reaches the daemon with no preflight. The handler
// must never run — the body is never decoded.
func TestSecure_RejectsNonJSONContentTypeWithoutDecoding(t *testing.T) {
	decoded := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8420/api/piloted/sessions", strings.NewReader(`{"provider":"cursor"}`))
	req.Host = "127.0.0.1:8420"
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set(clientHeader, "1")
	rec := httptest.NewRecorder()
	secure(testConfig(), handler).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d, want 415", rec.Code)
	}
	if decoded {
		t.Fatal("handler ran for a text/plain body — it must be rejected before decoding")
	}
}

func TestSecure_RejectsBodyWithNoContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8420/api/piloted/sessions", strings.NewReader(`{}`))
	req.Host = "127.0.0.1:8420"
	req.Header.Set(clientHeader, "1")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("got %d, want 415", rec.Code)
	}
}

func TestSecure_AllowsJSONContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8420/api/piloted/sessions", strings.NewReader(`{}`))
	req.Host = "127.0.0.1:8420"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(clientHeader, "1")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

func TestSecure_RequiresClientHeaderOnMutatingMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		req := newRequest(method, "/api/piloted/sessions/abc/cancel")
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s without client header: got %d, want 403", method, rec.Code)
		}
	}
}

// GET requests, including the two SSE streams, never carry the client
// header — EventSource cannot set one — so they must not require it.
func TestSecure_GetNeverNeedsClientHeader(t *testing.T) {
	for _, path := range []string{"/healthz", "/api/sessions", "/api/events/stream", "/api/piloted/sessions/abc/stream"} {
		req := newRequest(http.MethodGet, path)
		rec := httptest.NewRecorder()
		secure(testConfig(), okHandler()).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s: got %d, want 200", path, rec.Code)
		}
	}
}

func TestSecure_LegitimateMutatingRequestPasses(t *testing.T) {
	req := newRequest(http.MethodPost, "/api/piloted/sessions/abc/cancel")
	req.Header.Set(clientHeader, "1")
	req.Header.Set("Origin", "http://localhost:8420")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
}

// A rejection must be a diagnosable 403/415 with a reason, not a silent
// drop that leaves a misconfigured legitimate client guessing.
func TestSecure_RejectionBodyExplainsWhy(t *testing.T) {
	req := newRequest(http.MethodGet, "/api/sessions")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "evil.example") {
		t.Fatalf("rejection body doesn't explain why: %q", rec.Body.String())
	}
}

func TestSecure_NeverAddsCORSHeaders(t *testing.T) {
	req := newRequest(http.MethodOptions, "/api/sessions")
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	secure(testConfig(), okHandler()).ServeHTTP(rec, req)
	if v := rec.Header().Get("Access-Control-Allow-Origin"); v != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin: %q", v)
	}
}
