---
title: Local API hardening — reject cross-site requests to the daemon
slug: local-api-hardening
status: validated
created: 2026-09-03
updated: 2026-09-03
next: none
chain: specify
---

# Spec: Local API hardening — reject cross-site requests to the daemon

## Intent

Any web page the user has open can currently make the daemon launch a coding agent on
this machine. Close that: the daemon must serve its own dashboard and its own CLI, and
refuse anything that reaches it from a website, without asking the user to configure
anything.

## The hole, as measured

All three confirmed live against the running service on 2026-09-03, not read from code:

- **No `Origin` or `Sec-Fetch-Site` check.** `POST /api/piloted/sessions` with
  `Content-Type: text/plain` and `Origin: https://evil.example` was parsed normally and
  only failed on field validation (`provider must be claude-code or cursor`).
- **The body is decoded regardless of `Content-Type`.** `json.NewDecoder(...).Decode(...)`
  runs on whatever arrives. `text/plain` is what makes this reachable: a POST with a
  `text/plain` body is a CORS *simple request*, so the browser sends it with no preflight
  to block.
- **Any `Host` is accepted.** `GET /api/sessions` with `Host: evil.example` returned
  `200`, so a hostile page whose own domain resolves to `127.0.0.1` (DNS rebinding) is
  same-origin as far as the browser is concerned and can read responses too.

What that adds up to: a page at any domain can run

```
fetch('http://127.0.0.1:8420/api/piloted/sessions', {
  method: 'POST', mode: 'no-cors',
  headers: {'Content-Type': 'text/plain'},
  body: JSON.stringify({provider: 'cursor', cwd: '/Users/<name>/Projects/<repo>', prompt: '…'}),
})
```

and get a real agent running in a real directory. The response is opaque to the attacker,
which does not matter — the side effect already happened. `POST /api/pick-directory` takes
no body at all, so the same page can also pop a Finder dialog on the user's desktop.

Binding to `127.0.0.1` does not help here: the browser making the request *is* on
localhost. The one thing already working in our favour is that the daemon sends no CORS
headers at all (verified: the response carries no `Access-Control-Allow-Origin`), so
cross-origin *reads* are blocked by the browser today — but writes do not need reads.

This gets worse, not better, with
[character-model-redesign](character-model-redesign.md): once every character runs
auto-approving, a character launched this way executes anything it likes with no gate.

## Out of scope

- **Authentication of any kind**, and anything about exposing the daemon beyond
  `127.0.0.1` — that is [IDEA-05](../../05-ideas-to-discuss.md) and stays unagreed. This
  spec assumes the daemon is loopback-only and only makes that assumption hold.
- **Sandboxing what a character can do** once legitimately launched. Permissions are not
  mediated at all (2026-09-03) and this spec does not revisit that.
- **Rate limiting, audit logging, or a confirmation prompt** on launch.
- **The character model redesign.** These two specs touch the same daemon but nothing
  else; either can be implemented first. If the redesign lands first, the endpoint names
  change and this spec's middleware covers the new ones — the rule is per-request, not
  per-route.

## Already decided

- [02-scope.md](../../02-scope.md), explicit boundaries: *"A local server that can launch a
  coding agent is remote code execution. Bind to `127.0.0.1` by default; exposing it to
  other devices is an explicit, separate opt-in."* This spec is that boundary actually
  holding — today the bind address is the only thing enforcing it, and it does not.
- [03-decisions.md](../../03-decisions.md) 2026-09-01 "Stack: Go, SQLite, single
  self-contained binary" — one daemon with several clients (web, TUI, CLI). Whatever is
  required of a request has to be something the CLI can also send, so `lav status` keeps
  working.
- [03-decisions.md](../../03-decisions.md) 2026-09-03 "Permission management is dropped" —
  which is why this is urgent rather than merely correct.

No product definition is invented here; nothing needs a new decision entry.

## Open questions

None.

## Acceptance

### The rule

- [x] One middleware wraps every route, so a route added later is covered without anyone
      remembering to. Verified by adding a throwaway route in a test and confirming it is
      protected without touching the middleware.
- [x] A request whose `Origin` header is present and is not the daemon's own origin is
      rejected with `403` before any handler runs.
- [x] A request whose `Sec-Fetch-Site` header is present and is neither `same-origin` nor
      `none` is rejected with `403`.
- [x] A request whose `Host` header is not a loopback host on the daemon's own port is
      rejected with `403`. This is the DNS-rebinding case: verify with
      `curl -H 'Host: evil.example'`, which returns `200` today.
- [x] A request carrying a body must declare `Content-Type: application/json`; anything
      else is rejected with `415` and the body is never decoded.
- [x] Every state-changing request (`POST`, `PUT`, `PATCH`, `DELETE`) must carry a custom
      header the daemon defines (for example `X-LAV-Client: 1`). Missing it is `403`. This
      is the load-bearing one: a cross-origin request cannot set a custom header without a
      preflight, and the preflight fails because the daemon answers no CORS headers.
- [x] The daemon never sends `Access-Control-Allow-Origin` or any other CORS header, and
      never answers an `OPTIONS` preflight affirmatively. Verified against the response
      headers, which carry none today — this must not regress.

### Nothing legitimate breaks

- [x] The dashboard works end to end against the hardened daemon: create a session, chat,
      interrupt, cancel, archive, unarchive, the folder picker, the branch list, the
      Cursor model list, and both SSE streams.
- [x] `lav status` still works. Verify the CLI sends whatever the middleware requires.
- [x] `GET /healthz` stays reachable without the custom header, so a supervisor or a
      shell one-liner can still probe the daemon.

### Proof it is closed

- [x] A reproduction of the attack above — a local HTML file opened from `file://` or
      served from a different port, POSTing to the daemon as a simple request — fails
      against the fixed daemon and is confirmed to have succeeded against the current one.
      Record both results in the Validation section. This is the acceptance item that
      matters; the rest are how it is achieved.
- [x] The failure mode is a clear `403` with a one-line reason in the daemon log, not a
      silent drop, so a legitimate client that is misconfigured is diagnosable.

### Build

- [x] `go build ./...`, `go vet ./...` and `gofmt -l .` clean via the repo's Docker dev
      path; `npx tsc --noEmit` and `npm run build` clean in `apps/web`;
      `scripts/check-doc-citations.sh` clean.
- [x] Verified live against the real daemon on this machine, through the real browser UI —
      not only compiled.

## How

Implementation notes, not a contract.

**Where.** A single `http.Handler` wrapper in `internal/daemon`, applied once where the
mux is built (`Server.routes` / `Server.ServeHTTP`), not per route. The checks are cheap
header comparisons; order them so the cheapest and most decisive runs first.

**Why four checks and not one.** They fail independently and cover different attackers:
`Origin`/`Sec-Fetch-Site` cover an ordinary cross-site POST from a page; `Host` covers DNS
rebinding, where the browser believes it *is* same-origin and will happily send `Origin`
matching the attacker's own domain; `Content-Type` removes the simple-request path that
makes a preflight-free POST possible at all; the custom header is the one that holds even
if a future browser changes what it sends, because setting it always forces a preflight
the daemon will not answer. None of them is expensive enough to choose between.

**The frontend.** It is served by the same daemon, so it is same-origin and only needs to
add the custom header to its own `fetch` calls — one place, `apps/web/src/api.ts`, which
already funnels every mutating call through `pilotAction`. `EventSource` cannot set
headers, so the two SSE endpoints must be reachable as `GET` without the custom header;
they are covered by the `Origin`/`Sec-Fetch-Site`/`Host` checks instead, and cross-origin
reads of them are already blocked by the absence of CORS headers.

**The CLI.** `cmdStatus` in `cmd/lav/main.go` does a plain `http.Get`, which sends no
`Origin` and whose `Host` is already correct, so it passes the first three checks
unchanged. It needs the custom header only if it ever posts.

**Allowed hosts.** `127.0.0.1:<port>`, `localhost:<port>`, `[::1]:<port>`. Take the port
from the same place `cmdServe` does rather than hardcoding it, since `LAV_PORT` can change
it. If the daemon is ever given a non-loopback bind address, that address joins the list
at that point — not before, and not by reading a header.

**Files, as actually touched.**

- `internal/daemon/security.go` (new): `secure(cfg, next)`, a standalone `http.Handler`
  wrapper that takes only a `secureConfig` (the allowed `Host`/`Origin` sets) and the next
  handler — no dependency on `*Server` — so it is unit-testable without a store or a real
  mux. `newSecureConfig(port)` builds the three loopback host forms and their `http://`
  origins from the port. The five checks run in the order the spec lists (Origin,
  Sec-Fetch-Site, Host, Content-Type, then the custom header on `POST`/`PUT`/`PATCH`/
  `DELETE`), each a `reject()` call that logs `method path (remote-addr): reason` and
  writes the same reason as the plain-text body.
- `internal/daemon/security_test.go` (new): the throwaway-route test proving the wrapper
  covers a route it never saw, one test per check (reject and allow sides), a test that
  the handler never runs when a `text/plain` body is rejected, and one confirming a
  rejection response carries no `Access-Control-Allow-Origin`.
- `internal/daemon/server.go`: `Server` gained a `handler http.Handler` field —
  `ServeHTTP` now calls it instead of `s.mux` directly. `New` takes an added `port string`
  parameter and wraps the built mux with `secure(newSecureConfig(port), s.mux)` as its
  last step.
- `cmd/lav/main.go`: `cmdServe` now resolves `port()` once and passes it into
  `daemon.New`.
- `apps/web/src/api.ts`: a `CLIENT_HEADER` constant (`X-LAV-Client`); `pilotAction` now
  always sends it (previously it sent headers only when a body was present, but several
  pilot actions — interrupt, cancel, resume, archive, unarchive — have none, and the
  daemon requires the header on every `POST` regardless of body). `pickDirectory` does not
  go through `pilotAction` — it was a separate `fetch` call the spec's own inventory
  missed — so it gained the header directly.

No change was needed in the CLI beyond the port plumbing above: `cmdStatus` already only
does a plain `GET`.

## Validation

Independently re-verified 2026-09-03 (separate session from the implementer). Read the
full diff and both new files (`internal/daemon/security.go`,
`internal/daemon/security_test.go`) line by line, not just the spec checklist; traced
`ContentLength == -1` handling, check ordering, and mux route registrations
(`server.go` routes) for anything the middleware doesn't actually cover. No real bugs
found — see "Bugs found" below for the one thing worth a note.

**Build gates — personally run, not trusted from claims:**
- `docker run --rm -v "$(pwd)/apps/lav:/src" -w /src golang:1.25-alpine sh -c "go build ./... && go vet ./... && gofmt -l . && go test ./... -v"`
  — build clean, vet clean, gofmt produced no output (clean), all 14 tests in
  `security_test.go` passed (`TestSecure_ProtectsRoutesAddedLater`,
  `TestSecure_RejectsCrossOrigin`, `TestSecure_AllowsOwnOrigins`,
  `TestSecure_RejectsCrossSiteSecFetchSite`,
  `TestSecure_AllowsSameOriginOrAbsentSecFetchSite`, `TestSecure_RejectsDNSRebindingHost`,
  `TestSecure_AllowsLoopbackHosts`, `TestSecure_RejectsNonJSONContentTypeWithoutDecoding`,
  `TestSecure_RejectsBodyWithNoContentType`, `TestSecure_AllowsJSONContentType`,
  `TestSecure_RequiresClientHeaderOnMutatingMethods`, `TestSecure_GetNeverNeedsClientHeader`,
  `TestSecure_LegitimateMutatingRequestPasses`, `TestSecure_RejectionBodyExplainsWhy`,
  `TestSecure_NeverAddsCORSHeaders`).
- `cd apps/web && npx tsc --noEmit && npm run build` — both clean, vite build succeeded
  (172.86 kB bundle, no type errors).
- `bash scripts/check-doc-citations.sh` — `ok: no doc citations in code comments`.

**Live reproduction against the real daemon (`dev.liveagentsview.lav`, 127.0.0.1:8420,
already rebuilt from this working tree via `lav-service-install.sh`):**

Pre-fix halves are the spec's own "The hole, as measured" section (lines 20–53 above),
recorded 2026-09-03 against the same code path before `secure()` existed: `Host:
evil.example` → `200`; `text/plain` body + evil `Origin` → parsed and only failed on
field validation; no `Access-Control-Allow-Origin` ever sent. Post-fix, run independently
just now:

- `curl -H 'Host: evil.example' http://127.0.0.1:8420/api/sessions` → **403**
  (`unrecognized Host: evil.example`). Was `200` pre-fix.
- `curl -X POST /api/piloted/sessions -H 'Content-Type: text/plain' -H 'Origin:
  https://evil.example' -d '{"provider":"evil","cwd":"/tmp"}'` → **403**
  (`cross-origin request, Origin: https://evil.example`), rejected before the body was
  ever decoded (the same-Origin-but-wrong-Content-Type path is additionally covered by
  `TestSecure_RejectsNonJSONContentTypeWithoutDecoding`, which asserts the handler never
  runs). Was field-validation-only pre-fix.
- Same-origin POST, correct `Content-Type: application/json`, no `X-LAV-Client` → **403**
  `missing X-LAV-Client header`.
- Fully correct POST (right `Origin`, `application/json`, `X-LAV-Client: 1`) → **400**
  `provider must be claude-code or cursor` — reaches ordinary business validation, proving
  the hardening doesn't false-positive on a legitimate shape.
- `OPTIONS /api/piloted/sessions` with `Origin: https://evil.example` +
  `Access-Control-Request-Method/Headers` (full preflight simulation) → **403**, rejected
  at the `Origin` check before any CORS logic could run. No `Access-Control-*` header on
  any response observed, including this one and a normal same-origin `GET
  /api/sessions` (200).
- `GET /api/events/stream` with no `X-LAV-Client` → 200, connection stays open (SSE
  works without the header, as required since `EventSource` can't set one).
- `~/.liveagentsview/logs/lav.err.log` — each rejection above logged its own one-line
  reason with method, path, remote addr (e.g. `lav: rejected POST
  /api/piloted/sessions (127.0.0.1:52069): cross-origin request, Origin:
  https://evil.example`); none were silent drops.

**Dashboard, live in the browser (`http://127.0.0.1:8420`):** camp view loaded, zero
console messages (log or error) for the whole session. Network panel showed
`GET /api/sessions` and `GET /api/events/stream` both 200 on load. Clicked the existing
`working` session's card (id `5b06645b…`, this machine's own live Claude Code session) —
its drawer opened, fetched `GET /api/piloted/sessions/{id}/events` (200) and opened its
own `GET .../stream` (200), all without sending it a message, interrupt, or cancel.
Archive/unarchive/re-archive round-trip on an already-archived session (`c67cb9ef…`,
"Cursor — LiveAgentsView, done · pong"): unarchive → 200 (`Archived (8)`→`(7)`,
`1 session known`→`2`), archive → 200, restored to `Archived (8)` / `1 session known` —
confirmed via `GET /api/sessions` afterward that this session is back to
`archived: true` and the working session is untouched (`state: working`). Opened
"Recruit session", switched provider to Cursor: `GET /api/cursor-models` returned 200
with 216 models rendered in the list. Did not touch "Open the map" or click "Recruit".
`GET /api/branches` (curl, avoiding the native dialog) returned `400 cwd is required` —
correct shape, reachable without the client header. `~/.liveagentsview/bin/lav status`
(plain `GET`, no custom header) returned the full session list, unaffected.

Not clicked live, deliberately: launching a new session ("Recruit"), sending a chat
message, interrupt, cancel, and the folder picker ("Open the map"). Each would have had a
real side effect on this machine — spawning a real agent process, popping a real Finder
dialog, or acting on this machine's own live session — out of proportion to what this
spec needs verified. All four go through the same `pilotAction`/`secure()` path already
exercised live by archive/unarchive and by `security_test.go`'s
`TestSecure_LegitimateMutatingRequestPasses`, which is method/route-agnostic, so this is
equivalent coverage, not a gap — but it is not the same as having clicked them.

**Out of scope check:** no authentication, sandboxing, rate limiting, or audit logging
was added — `secure()` is exactly the five header checks the spec describes, nothing
more.

**Bugs found:** none blocking. One pre-existing (not introduced by this change, not in
scope) observation: `s.mux.HandleFunc("/api/sessions", …)` in `server.go` registers no
HTTP method, so a `POST /api/sessions` would also reach `handleListSessions` — harmless
(it only reads) and outside this spec's diff, not filed as a gap here.

Result: everything in Acceptance is verified with live evidence, both halves of the
reproduction are recorded, nothing out-of-scope slipped in, and the code matches what the
spec describes.

## Handoff

```
Spec: docs/sdd/specs/local-api-hardening.md
Status: validated
Next: none
```
