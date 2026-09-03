---
title: Local API hardening — reject cross-site requests to the daemon
slug: local-api-hardening
status: ready
created: 2026-09-03
updated: 2026-09-03
next: implement
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

- [ ] One middleware wraps every route, so a route added later is covered without anyone
      remembering to. Verified by adding a throwaway route in a test and confirming it is
      protected without touching the middleware.
- [ ] A request whose `Origin` header is present and is not the daemon's own origin is
      rejected with `403` before any handler runs.
- [ ] A request whose `Sec-Fetch-Site` header is present and is neither `same-origin` nor
      `none` is rejected with `403`.
- [ ] A request whose `Host` header is not a loopback host on the daemon's own port is
      rejected with `403`. This is the DNS-rebinding case: verify with
      `curl -H 'Host: evil.example'`, which returns `200` today.
- [ ] A request carrying a body must declare `Content-Type: application/json`; anything
      else is rejected with `415` and the body is never decoded.
- [ ] Every state-changing request (`POST`, `PUT`, `PATCH`, `DELETE`) must carry a custom
      header the daemon defines (for example `X-LAV-Client: 1`). Missing it is `403`. This
      is the load-bearing one: a cross-origin request cannot set a custom header without a
      preflight, and the preflight fails because the daemon answers no CORS headers.
- [ ] The daemon never sends `Access-Control-Allow-Origin` or any other CORS header, and
      never answers an `OPTIONS` preflight affirmatively. Verified against the response
      headers, which carry none today — this must not regress.

### Nothing legitimate breaks

- [ ] The dashboard works end to end against the hardened daemon: create a session, chat,
      interrupt, cancel, archive, unarchive, the folder picker, the branch list, the
      Cursor model list, and both SSE streams.
- [ ] `lav status` still works. Verify the CLI sends whatever the middleware requires.
- [ ] `GET /healthz` stays reachable without the custom header, so a supervisor or a
      shell one-liner can still probe the daemon.

### Proof it is closed

- [ ] A reproduction of the attack above — a local HTML file opened from `file://` or
      served from a different port, POSTing to the daemon as a simple request — fails
      against the fixed daemon and is confirmed to have succeeded against the current one.
      Record both results in the Validation section. This is the acceptance item that
      matters; the rest are how it is achieved.
- [ ] The failure mode is a clear `403` with a one-line reason in the daemon log, not a
      silent drop, so a legitimate client that is misconfigured is diagnosable.

### Build

- [ ] `go build ./...`, `go vet ./...` and `gofmt -l .` clean via the repo's Docker dev
      path; `npx tsc --noEmit` and `npm run build` clean in `apps/web`;
      `scripts/check-doc-citations.sh` clean.
- [ ] Verified live against the real daemon on this machine, through the real browser UI —
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

**Files.** `internal/daemon/server.go` (the wrapper and its wiring), a new test file
beside it, `apps/web/src/api.ts` (the header on mutating calls), and `cmd/lav/main.go`
only if the CLI grows a mutating call.

## Validation

Filled in by whoever validates. Must include both halves of the reproduction: the attack
succeeding against the pre-fix daemon and failing against the fixed one.

## Handoff

```
Spec: docs/sdd/specs/local-api-hardening.md
Status: ready
Next: implement
```
