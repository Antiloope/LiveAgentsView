// Package installer implements "lav init": it merges LiveAgentsView's hooks
// into each provider's existing configuration without overwriting anything
// already there, previews the exact change, and writes small forwarder
// scripts that POST hook payloads to the daemon.
//
// See docs/03-decisions.md 2026-09-01 "`lav init` merges hooks
// non-destructively, with a preview" (closes Q-07).
package installer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Options struct {
	// Home is where provider config lives (contains .claude/, .codex/,
	// .cursor/), from this process's own filesystem view.
	Home string
	// LavHome is LiveAgentsView's own data directory, from this process's
	// filesystem view — where forwarder scripts actually get written.
	LavHome string
	// LavHomeRef is how the *host* sees LavHome, when this process runs
	// inside a container with LavHome bind-mounted at a different path
	// (e.g. LavHome=/data inside Docker, but the host's Claude Code process
	// that will later execute the script sees it at ~/.liveagentsview).
	// Left empty, LavHome is used directly (the common case: not running
	// inside a container, or LavHome already *is* the host path).
	LavHomeRef string
	Port       string
	DryRun     bool
}

func (o Options) refHome() string {
	if o.LavHomeRef != "" {
		return o.LavHomeRef
	}
	return o.LavHome
}

type Result struct {
	Preview   []string
	Providers []string
}

const (
	claudeSettingsRel = ".claude/settings.json"
	codexConfigRel    = ".codex/config.toml"
	cursorHooksRel    = ".cursor/hooks.json"
)

func Init(opt Options) (Result, error) {
	var res Result

	steps := []struct {
		name string
		fn   func(Options) (bool, []string, error)
	}{
		{"claude-code", initClaudeCode},
		{"codex", initCodex},
		{"cursor", initCursor},
	}

	for _, step := range steps {
		ok, lines, err := step.fn(opt)
		if err != nil {
			return res, fmt.Errorf("%s: %w", step.name, err)
		}
		if ok {
			res.Preview = append(res.Preview, lines...)
			res.Providers = append(res.Providers, step.name)
		}
	}
	return res, nil
}

// --- Claude Code -----------------------------------------------------------

const claudeCodeScriptTmpl = `#!/bin/sh
# Written by "lav init". Forwards a Claude Code hook payload to the LiveAgentsView daemon.
event="$1"
repo=""
branch=""
worktree=""
if command -v git >/dev/null 2>&1; then
	top=$(git rev-parse --show-toplevel 2>/dev/null)
	if [ -n "$top" ]; then
		repo=$(basename "$top")
	fi
	branch=$(git branch --show-current 2>/dev/null)
	gitdir=$(git rev-parse --git-dir 2>/dev/null)
	commondir=$(git rev-parse --git-common-dir 2>/dev/null)
	if [ -n "$gitdir" ] && [ -n "$commondir" ] && [ "$gitdir" != "$commondir" ] && [ -n "$top" ]; then
		worktree=$(basename "$top")
	fi
fi
curl -s -m 5 -X POST "%s/hooks/claude-code?event=${event}&repo=${repo}&branch=${branch}&worktree=${worktree}" \
	-H "Content-Type: application/json" \
	--data-binary @- >/dev/null 2>&1 || true
`

var claudeCodeEvents = []string{
	"SessionStart", "SessionEnd", "UserPromptSubmit", "Notification", "Stop", "StopFailure", "SubagentStop",
}

func initClaudeCode(opt Options) (bool, []string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s", opt.Port)
	scriptPath := filepath.Join(opt.LavHome, "bin", "claude-code-hook.sh")
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "claude-code-hook.sh")
	settingsPath := filepath.Join(opt.Home, claudeSettingsRel)

	raw, existed, err := readJSONFile(settingsPath)
	if err != nil {
		return false, nil, err
	}
	hooksField, _ := raw["hooks"].(map[string]any)
	if hooksField == nil {
		hooksField = map[string]any{}
	}

	preview := []string{fmt.Sprintf("Claude Code (%s):", settingsPath)}
	if !existed {
		preview = append(preview, "  - file does not exist yet, will be created")
	}
	preview = append(preview, fmt.Sprintf("  - write helper script %s", scriptPathRef))

	changed := false
	for _, event := range claudeCodeEvents {
		existingList, _ := hooksField[event].([]any)
		if hasCommandPrefix(existingList, "hooks", scriptPathRef) {
			continue
		}
		entry := map[string]any{
			"matcher": "",
			"hooks": []any{
				map[string]any{
					"type":    "command",
					"command": fmt.Sprintf("%s %s", scriptPathRef, event),
				},
			},
		}
		hooksField[event] = append(existingList, entry)
		changed = true
		if len(existingList) > 0 {
			preview = append(preview, fmt.Sprintf("  - append to existing %q hooks (kept: %d)", event, len(existingList)))
		} else {
			preview = append(preview, fmt.Sprintf("  - add %q hook (none existed)", event))
		}
	}
	raw["hooks"] = hooksField

	if !changed {
		return true, append(preview, "  - already configured, nothing to change"), nil
	}

	if !opt.DryRun {
		if err := writeScript(scriptPath, fmt.Sprintf(claudeCodeScriptTmpl, url)); err != nil {
			return false, nil, err
		}
		if err := writeJSONFile(settingsPath, raw); err != nil {
			return false, nil, err
		}
	}
	return true, preview, nil
}

// --- Codex -------------------------------------------------------------

var notifyLineRe = regexp.MustCompile(`(?m)^[ \t]*notify[ \t]*=[ \t]*(\[[^\n]*\])[ \t]*$`)

const codexNotifyScriptTmpl = `#!/bin/sh
# Written by "lav init". Forwards the Codex "notify" payload to LiveAgentsView
# and, if one was already configured, chains to the previous notify target too.
payload="$1"
%s
curl -s -m 5 -X POST "%s/hooks/codex" \
	-H "Content-Type: application/json" \
	--data-binary "$payload" >/dev/null 2>&1 || true
`

func initCodex(opt Options) (bool, []string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s", opt.Port)
	scriptPath := filepath.Join(opt.LavHome, "bin", "codex-notify.sh")
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "codex-notify.sh")
	configPath := filepath.Join(opt.Home, codexConfigRel)

	preview := []string{fmt.Sprintf("Codex (%s):", configPath)}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		preview = append(preview, "  - file does not exist yet, will be created")
		data = []byte{}
	} else if err != nil {
		return false, nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	content := string(data)

	if strings.Contains(content, scriptPathRef) {
		return true, append(preview, "  - already configured, nothing to change"), nil
	}

	chainLine := "# no previous notify target found, nothing to chain"
	if m := notifyLineRe.FindStringSubmatch(content); m != nil {
		argv := extractStringArray(m[1])
		if len(argv) > 0 {
			quoted := make([]string, len(argv))
			for i, a := range argv {
				quoted[i] = fmt.Sprintf("%q", a)
			}
			chainLine = strings.Join(quoted, " ") + ` "$payload" >/dev/null 2>&1 || true`
			preview = append(preview, fmt.Sprintf("  - found existing notify = %s, chaining to it (original target kept, not replaced)", m[1]))
		}
	} else {
		preview = append(preview, "  - no existing notify target found")
	}

	newLine := fmt.Sprintf("notify = [%q]", scriptPathRef)
	var newContent string
	if notifyLineRe.MatchString(content) {
		newContent = notifyLineRe.ReplaceAllString(content, newLine)
		preview = append(preview, fmt.Sprintf("  - replace the notify line with %s (original target preserved inside the wrapper script)", newLine))
	} else {
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		newContent = content + newLine + "\n"
		preview = append(preview, fmt.Sprintf("  - add %s", newLine))
	}
	preview = append(preview, fmt.Sprintf("  - write helper script %s", scriptPathRef))

	if !opt.DryRun {
		if err := writeScript(scriptPath, fmt.Sprintf(codexNotifyScriptTmpl, chainLine, url)); err != nil {
			return false, nil, err
		}
		if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
			return false, nil, fmt.Errorf("create %s: %w", filepath.Dir(configPath), err)
		}
		if err := os.WriteFile(configPath, []byte(newContent), 0o644); err != nil {
			return false, nil, fmt.Errorf("write %s: %w", configPath, err)
		}
	}
	return true, preview, nil
}

// extractStringArray turns a TOML array literal of plain strings, e.g.
// ["/a/b", "turn-ended"], into its elements. Best-effort: arrays holding
// anything other than simple double-quoted strings are not supported and
// yield nil (the caller then just skips chaining rather than guessing).
func extractStringArray(arr string) []string {
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(arr), "["), "]")
	var out []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if len(part) < 2 || part[0] != '"' || part[len(part)-1] != '"' {
			return nil
		}
		out = append(out, part[1:len(part)-1])
	}
	return out
}

// --- Cursor --------------------------------------------------------------

const cursorHookScriptTmpl = `#!/bin/sh
# Written by "lav init". Forwards a cursor-agent hook payload to LiveAgentsView.
event="$1"
repo=""
branch=""
worktree=""
if command -v git >/dev/null 2>&1; then
	top=$(git rev-parse --show-toplevel 2>/dev/null)
	if [ -n "$top" ]; then
		repo=$(basename "$top")
	fi
	branch=$(git branch --show-current 2>/dev/null)
	gitdir=$(git rev-parse --git-dir 2>/dev/null)
	commondir=$(git rev-parse --git-common-dir 2>/dev/null)
	if [ -n "$gitdir" ] && [ -n "$commondir" ] && [ "$gitdir" != "$commondir" ] && [ -n "$top" ]; then
		worktree=$(basename "$top")
	fi
fi
curl -s -m 5 -X POST "%s/hooks/cursor?event=${event}&repo=${repo}&branch=${branch}&worktree=${worktree}" \
	-H "Content-Type: application/json" \
	--data-binary @- >/dev/null 2>&1 || true
`

var cursorEvents = []string{"sessionStart", "stop", "sessionEnd", "postToolUseFailure"}

func initCursor(opt Options) (bool, []string, error) {
	url := fmt.Sprintf("http://127.0.0.1:%s", opt.Port)
	scriptPath := filepath.Join(opt.LavHome, "bin", "cursor-hook.sh")
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "cursor-hook.sh")
	hooksPath := filepath.Join(opt.Home, cursorHooksRel)

	raw, existed, err := readJSONFile(hooksPath)
	if err != nil {
		return false, nil, err
	}
	hooksField, _ := raw["hooks"].(map[string]any)
	if hooksField == nil {
		hooksField = map[string]any{}
	}

	preview := []string{fmt.Sprintf("Cursor (%s):", hooksPath)}
	if !existed {
		preview = append(preview, "  - file does not exist yet, will be created")
	}
	preview = append(preview, fmt.Sprintf("  - write helper script %s", scriptPathRef))
	preview = append(preview, "  - note: unconfirmed whether these fire for sessions launched from the Cursor IDE itself, only confirmed for cursor-agent CLI sessions (see spec's Event model notes)")

	changed := false
	for _, event := range cursorEvents {
		existingList, _ := hooksField[event].([]any)
		if hasCommandPrefix(existingList, "", scriptPathRef) {
			continue
		}
		entry := map[string]any{"command": fmt.Sprintf("%s %s", scriptPathRef, event)}
		hooksField[event] = append(existingList, entry)
		changed = true
		if len(existingList) > 0 {
			preview = append(preview, fmt.Sprintf("  - append to existing %q hooks (kept: %d)", event, len(existingList)))
		} else {
			preview = append(preview, fmt.Sprintf("  - add %q hook (none existed)", event))
		}
	}
	raw["hooks"] = hooksField

	if !changed {
		return true, append(preview, "  - already configured, nothing to change"), nil
	}

	if !opt.DryRun {
		if err := writeScript(scriptPath, fmt.Sprintf(cursorHookScriptTmpl, url)); err != nil {
			return false, nil, err
		}
		if err := writeJSONFile(hooksPath, raw); err != nil {
			return false, nil, err
		}
	}
	return true, preview, nil
}

// --- shared helpers --------------------------------------------------------

// hasCommandPrefix reports whether any entry in list already points at
// scriptPath, so "lav init" is idempotent instead of appending duplicates
// on every run. nestedKey is "hooks" for Claude Code's {matcher, hooks:[...]}
// shape, or "" for Cursor's flatter {command: "..."} shape.
func hasCommandPrefix(list []any, nestedKey, scriptPath string) bool {
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if nestedKey == "" {
			if cmd, _ := m["command"].(string); strings.HasPrefix(cmd, scriptPath) {
				return true
			}
			continue
		}
		nested, _ := m[nestedKey].([]any)
		for _, h := range nested {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := hm["command"].(string); strings.HasPrefix(cmd, scriptPath) {
				return true
			}
		}
	}
	return false
}

func readJSONFile(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read %s: %w", path, err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, true, nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w (refusing to touch a file lav init cannot safely round-trip)", path, err)
	}
	return m, true, nil
}

func writeJSONFile(path string, v map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func writeScript(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
