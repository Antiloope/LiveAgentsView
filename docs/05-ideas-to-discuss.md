# Ideas to discuss

**Unagreed** proposals. Nothing here is definition until discussed and moved to the
right document (or to [03-decisions.md](03-decisions.md)).

Format:

```
## IDEA-NN — Title

**Proposal:** …
**Status:** unagreed
```

---

## IDEA-01 — Attention priority taxonomy

**Proposal:** Classify every attention event into a fixed set of levels that determine
whether and how it notifies:

| Level | What it is | Signal | Notifies |
|---|---|---|---|
| **P0 · Blocked** | Cannot continue without you | Permission request | Yes, loud |
| **P1 · Decision** | It asked you something | End of turn classified as a question | Yes |
| **P2 · Done** | It closed the task | End of turn classified as finished | Silent, grouped |
| **P3 · Suspicious** | Quiet too long, or looping | No activity for N min while WORKING | No, visible only |
| **Failed** | Process died or errored | Non-zero exit, error in the stream | Yes |
| *(none)* | Working | Tool calls, output | Never |

Two parts of this are already decided and are cited in [02-scope.md](02-scope.md): P2 for
completion, and rules-based end-of-turn classification. The rest of the table — in
particular the P3 definition and the exact notification behaviour per level — still needs
sign-off.

**Status:** unagreed

## IDEA-02 — Attention depends on human presence, not just agent state

**Proposal:** Model attention as `agent state × is a human watching that session`, rather
than as a property of the agent alone. An agent finishing its turn while the user is
sitting in that terminal is not an attention event — they already saw it. The same event
in a tab abandoned 40 minutes ago is. In piloted mode nobody is watching by definition,
so every end of turn counts.

Open sub-question: how presence is actually detected (terminal focus, recent
`UserPromptSubmit`, tmux active pane) — see Q-06.

**Status:** unagreed

## IDEA-03 — Attention items must auto-resolve

**Proposal:** An attention item disappears when it has been handled **by any route**,
including answering directly in the terminal. Claude Code's `UserPromptSubmit` hook gives
this for free. Without it the queue fills with stale items, the user stops trusting it,
and the dashboard becomes a dead to-do list — the most common failure mode of this kind
of tool.

**Status:** unagreed

## IDEA-04 — Throttling and deduplication per session

**Proposal:** Collapse repeated events from the same session into a single queue item.
An agent asking for permission 40 times inside a bash loop is one item, not forty.

**Status:** unagreed

## IDEA-05 — Authentication for non-local access

**Proposal:** When the daemon is exposed beyond `127.0.0.1` (for example over Tailscale
to a phone), require a token even inside a private tailnet, because approving a permission
is literally executing code on the user's machine. Needs a concrete mechanism: token file
in `~/.liveagentsview/`, pairing flow, expiry.

**Status:** unagreed

## IDEA-06 — Session launch modes for piloted agents

**Proposal:** Piloted sessions could run either as a plain child process or inside a tmux
session that LiveAgentsView owns, so that "open in the terminal" can attach to the exact
same live session instead of resuming a new one. Claude Code already has `--worktree` and
`--tmux` natively, which should be reused rather than reimplemented.

**2026-09-02:** live-tested — tmux (and Claude Code's own `--bg`/Cursor's `agent
persist`) cannot carry the headless `-p`/`stream-json` driver protocol piloted mode
depends on; all three are built for a human re-attaching to an interactive terminal, not
a second machine-readable channel. See
[03-decisions.md](03-decisions.md) 2026-09-02 "No CLI-native background/persistent-
session feature fits piloted mode". Restart continuity is being pursued instead via a
supervisor LiveAgentsView builds itself —
[piloted-only-mode](sdd/specs/piloted-only-mode.md).
This idea's narrower proposal — a human attaching a real terminal to a piloted
session — is untouched by that finding and stays open/unagreed.

**Status:** unagreed

## IDEA-07 — `lav version`

**Proposal:** A version/build-identity subcommand (and equivalent dashboard footer) so it
is possible to tell which build a running service is actually on, instead of the only way
to be sure being "reinstall and see". Matters more now that
[native-host-runtime](sdd/specs/native-host-runtime.md) makes the daemon a long-lived
service the user does not restart on every code change.

**Status:** unagreed

## IDEA-08 — Uninstall path

**Proposal:** A symmetric teardown for everything `lav init` and
`lav service install` write: `lav service uninstall` (unload/disable the launchd/systemd
unit, remove the plist/unit file) and a way to remove the hooks/notify entries from
`~/.claude/settings.json`, `~/.codex/config.toml` and `~/.cursor/hooks.json` without hand
editing them. No destructive action should require reading Go source to reverse.

**2026-09-02:** the hooks/notify-removal half is being built as part of
[piloted-only-mode](sdd/specs/piloted-only-mode.md), which removes `lav init` and
`internal/installer` entirely and uninstalls the hooks they already wrote to this real
machine's configs — see [03-decisions.md](03-decisions.md) 2026-09-02. The
`lav service uninstall` half (launchd/systemd unit teardown) is unrelated and stays
unagreed/open.

**Status:** unagreed

## IDEA-09 — Friendlier local dashboard address

**Proposal:** Reach the dashboard at something nicer than `127.0.0.1:8420` — a local
hostname (e.g. via mDNS/`.local`) or a fixed alias — as a small UX polish on top of the
already-decided 127.0.0.1-only binding. Not remote access (that stays IDEA-05 territory);
just a nicer name for the same loopback address.

**Status:** unagreed

## IDEA-10 — Prebuilt binary distribution and a first-run installer wizard

**Proposal:** Host prebuilt `lav` binaries per OS/arch (mac/linux, amd64/arm64) somewhere
installable without cloning the repo and running Docker to compile — the current
`lav-service-install.sh` path is fine for the author dogfooding, but is friction for
anyone else. Pair it with a first-run console wizard that asks how the user wants the
daemon to run (register as a systemd/launchd service now, or just start it by hand) rather
than always installing a service. Where to host the binaries (GitHub Releases, a package
manager, something else) is itself part of what needs deciding here.

**Status:** unagreed

## IDEA-11 — A permission layer owned by LiveAgentsView

**Proposal:** Bring tool permissions back, but as system configuration rather than as a
chat question. Three pieces: a **durable policy** (rules of the form "tool + path pattern
→ allow / ask / deny", global and per character) that LiveAgentsView evaluates itself, so
it only asks about what no rule covers; **durable pending requests** (a row with a status,
not a map in the daemon's memory, so a restart re-shows the question instead of stranding
the character forever); and a bridge between them — answering with "remember this" writes
a rule. Enforcement differs by race and would have to be declared: Claude Code can be
gated (LiveAgentsView already sits in front of it), Cursor has no channel for it today,
though the decision log records `.cursor/hooks.json`'s `beforeShellExecution` firing for
`cursor-agent` — whether it can actually *block* a call is unverified and is what would
have to be tested first.

**2026-09-03:** proposed and explicitly deferred by Rodrigo in the same session that
removed permission management entirely — "luego en el futuro podríamos ver si necesitamos
implementarlo o no". See [03-decisions.md](03-decisions.md) 2026-09-03 "Permission
management is dropped; every race runs auto-approving". Reviving this also revives the
`blocked` activity and gives [IDEA-01](05-ideas-to-discuss.md)'s P0 level content again.

**Status:** unagreed

## IDEA-12 — Changing a character's class after it is created

**Proposal:** Let a character change class (its model) without being recreated — for
example dropping a character to a smaller, cheaper model once the hard part of the work is
done. [03-decisions.md](03-decisions.md) 2026-09-03 "Vocabulary: character, race, class,
quest" already establishes that class is the changeable half and race is not, and the
mechanism is nearly free: the model is a flag passed when a process starts, so a class
change is a sleep and a wake with a different `--model` for Claude Code, and takes effect
on the next turn for Cursor. What is not defined is the product side: whether the change
applies from the next quest or mid-quest, what the transcript shows, and whether class
history is worth keeping.

**Status:** unagreed
