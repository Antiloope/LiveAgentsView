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

**Status:** unagreed
