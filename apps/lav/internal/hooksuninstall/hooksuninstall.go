// Package hooksuninstall removes exactly what "lav init" used to add to
// Claude Code, Codex and Cursor's config — the symmetric inverse of the
// installer that piloted-only mode retires. It never touches anything in
// those files it did not itself add.
package hooksuninstall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Options struct {
	// Home is where provider config lives (contains .claude/, .codex/,
	// .cursor/), from this process's own filesystem view.
	Home string
	// LavHome is LiveAgentsView's own data directory, from this process's
	// filesystem view — where the forwarder scripts being removed live.
	LavHome string
	// LavHomeRef is how the *host* sees LavHome — see installer.Options'
	// identical field for why this can differ from LavHome. Needed here to
	// recognize which config entries are LiveAgentsView's own: they were
	// written pointing at this host path, not at LavHome directly, when
	// the two differ.
	LavHomeRef string
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
	// Backups is every config file copied aside before being rewritten,
	// original path -> backup path. Empty in a dry run.
	Backups map[string]string
}

const (
	claudeSettingsRel = ".claude/settings.json"
	codexConfigRel    = ".codex/config.toml"
	cursorHooksRel    = ".cursor/hooks.json"
)

func Uninstall(opt Options) (Result, error) {
	res := Result{Backups: map[string]string{}}

	steps := []struct {
		name string
		fn   func(Options, *Result) (bool, []string, error)
	}{
		{"claude-code", uninstallClaudeCode},
		{"codex", uninstallCodex},
		{"cursor", uninstallCursor},
	}

	for _, step := range steps {
		ok, lines, err := step.fn(opt, &res)
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

var claudeCodeEvents = []string{
	"SessionStart", "SessionEnd", "UserPromptSubmit", "Notification", "Stop", "StopFailure", "SubagentStop",
}

func uninstallClaudeCode(opt Options, res *Result) (bool, []string, error) {
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "claude-code-hook.sh")
	settingsPath := filepath.Join(opt.Home, claudeSettingsRel)

	raw, existed, err := readJSONFile(settingsPath)
	if err != nil {
		return false, nil, err
	}
	if !existed {
		return false, nil, nil
	}
	hooksField, _ := raw["hooks"].(map[string]any)
	if hooksField == nil {
		return false, nil, nil
	}

	preview := []string{fmt.Sprintf("Claude Code (%s):", settingsPath)}
	changed := false
	for _, event := range claudeCodeEvents {
		existingList, _ := hooksField[event].([]any)
		if len(existingList) == 0 {
			continue
		}
		kept := existingList[:0:0]
		removed := 0
		for _, item := range existingList {
			if entryMatchesScript(item, "hooks", scriptPathRef) {
				removed++
				continue
			}
			kept = append(kept, item)
		}
		if removed == 0 {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooksField, event)
			preview = append(preview, fmt.Sprintf("  - remove %q hook entirely (it was ours only)", event))
		} else {
			hooksField[event] = kept
			preview = append(preview, fmt.Sprintf("  - remove our %q hook (kept: %d other)", event, len(kept)))
		}
	}

	if !changed {
		return false, nil, nil
	}
	raw["hooks"] = hooksField
	preview = append(preview, fmt.Sprintf("  - delete helper script %s", scriptPathRef))

	if !opt.DryRun {
		backup, err := backupFile(settingsPath)
		if err != nil {
			return false, nil, err
		}
		res.Backups[settingsPath] = backup
		if err := writeJSONFile(settingsPath, raw); err != nil {
			return false, nil, err
		}
		removeScript(opt.LavHome, "claude-code-hook.sh")
	}
	return true, preview, nil
}

// --- Codex -------------------------------------------------------------

var notifyLineRe = regexp.MustCompile(`(?m)^[ \t]*notify[ \t]*=[ \t]*(\[[^\n]*\])[ \t]*\n?`)

func uninstallCodex(opt Options, res *Result) (bool, []string, error) {
	scriptPath := filepath.Join(opt.LavHome, "bin", "codex-notify.sh")
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "codex-notify.sh")
	configPath := filepath.Join(opt.Home, codexConfigRel)

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	content := string(data)
	if !strings.Contains(content, scriptPathRef) {
		return false, nil, nil
	}

	preview := []string{fmt.Sprintf("Codex (%s):", configPath)}

	original, hadChain := recoverChainedNotifyTarget(scriptPath)
	var newContent string
	if hadChain {
		quoted := make([]string, len(original))
		for i, a := range original {
			quoted[i] = fmt.Sprintf("%q", a)
		}
		newLine := fmt.Sprintf("notify = [%s]", strings.Join(quoted, ", "))
		newContent = notifyLineRe.ReplaceAllString(content, newLine+"\n")
		preview = append(preview, fmt.Sprintf("  - restore the pre-existing notify = %s (chained target it wrapped)", strings.Join(quoted, ", ")))
	} else {
		newContent = notifyLineRe.ReplaceAllString(content, "")
		preview = append(preview, "  - remove the notify line entirely (nothing was configured before lav init)")
	}
	preview = append(preview, fmt.Sprintf("  - delete helper script %s", scriptPathRef))

	if !opt.DryRun {
		backup, err := backupFile(configPath)
		if err != nil {
			return false, nil, err
		}
		res.Backups[configPath] = backup
		if err := os.WriteFile(configPath, []byte(newContent), 0o644); err != nil {
			return false, nil, fmt.Errorf("write %s: %w", configPath, err)
		}
		removeScript(opt.LavHome, "codex-notify.sh")
	}
	return true, preview, nil
}

// recoverChainedNotifyTarget reads the forwarder script "lav init" wrote and
// pulls the pre-existing notify target it chained onto, if any — the config
// file's own notify line only ever pointed at the script itself, so the
// original target (if there was one) only still exists inside it. Returns
// ok=false when the script shows no chained target (the comment line
// initCodex writes when none was found), meaning notify had nothing before.
func recoverChainedNotifyTarget(scriptPath string) ([]string, bool) {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, false
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != `payload="$1"` {
			continue
		}
		if i+1 >= len(lines) {
			return nil, false
		}
		chainLine := strings.TrimSpace(lines[i+1])
		if chainLine == "" || strings.HasPrefix(chainLine, "#") {
			return nil, false
		}
		return unquoteChainArgv(chainLine), true
	}
	return nil, false
}

// unquoteChainArgv parses the argv initCodex wrote as space-joined %q
// literals, up to the fixed `"$payload" >/dev/null 2>&1 || true` suffix.
func unquoteChainArgv(line string) []string {
	var argv []string
	for _, tok := range splitQuotedTokens(line) {
		if !strings.HasPrefix(tok, `"`) {
			break
		}
		s, err := strconv.Unquote(tok)
		if err != nil {
			break
		}
		if s == "$payload" {
			break
		}
		argv = append(argv, s)
	}
	return argv
}

// splitQuotedTokens splits on whitespace outside double quotes, keeping a
// quoted token (with its quotes) intact even when it contains spaces or
// escaped characters — exactly what strconv.Unquote expects as input.
func splitQuotedTokens(s string) []string {
	var toks []string
	i, n := 0, len(s)
	for i < n {
		for i < n && s[i] == ' ' {
			i++
		}
		if i >= n {
			break
		}
		if s[i] != '"' {
			j := i
			for j < n && s[j] != ' ' {
				j++
			}
			toks = append(toks, s[i:j])
			i = j
			continue
		}
		j := i + 1
		for j < n && s[j] != '"' {
			if s[j] == '\\' && j+1 < n {
				j++
			}
			j++
		}
		if j < n {
			j++ // include closing quote
		}
		toks = append(toks, s[i:j])
		i = j
	}
	return toks
}

// --- Cursor --------------------------------------------------------------

var cursorEvents = []string{"sessionStart", "stop", "sessionEnd", "postToolUseFailure"}

func uninstallCursor(opt Options, res *Result) (bool, []string, error) {
	scriptPathRef := filepath.Join(opt.refHome(), "bin", "cursor-hook.sh")
	hooksPath := filepath.Join(opt.Home, cursorHooksRel)

	raw, existed, err := readJSONFile(hooksPath)
	if err != nil {
		return false, nil, err
	}
	if !existed {
		return false, nil, nil
	}
	hooksField, _ := raw["hooks"].(map[string]any)
	if hooksField == nil {
		return false, nil, nil
	}

	preview := []string{fmt.Sprintf("Cursor (%s):", hooksPath)}
	changed := false
	for _, event := range cursorEvents {
		existingList, _ := hooksField[event].([]any)
		if len(existingList) == 0 {
			continue
		}
		kept := existingList[:0:0]
		removed := 0
		for _, item := range existingList {
			if entryMatchesScript(item, "", scriptPathRef) {
				removed++
				continue
			}
			kept = append(kept, item)
		}
		if removed == 0 {
			continue
		}
		changed = true
		if len(kept) == 0 {
			delete(hooksField, event)
			preview = append(preview, fmt.Sprintf("  - remove %q hook entirely (it was ours only)", event))
		} else {
			hooksField[event] = kept
			preview = append(preview, fmt.Sprintf("  - remove our %q hook (kept: %d other)", event, len(kept)))
		}
	}

	if !changed {
		return false, nil, nil
	}
	raw["hooks"] = hooksField

	// hooksField is the file's only top-level key (see initCursor): once every
	// event list under it is empty, the whole file is LiveAgentsView's alone
	// and deleting it is the non-destructive move — writing back an
	// almost-empty {"hooks":{}} would just leave inert clutter.
	onlyOurs := len(hooksField) == 0
	for k := range raw {
		if k != "hooks" && raw[k] != nil {
			onlyOurs = false
		}
	}

	if onlyOurs {
		preview = append(preview, fmt.Sprintf("  - delete %s (only our entries were in it)", hooksPath))
	} else {
		preview = append(preview, fmt.Sprintf("  - delete helper script %s", scriptPathRef))
	}

	if !opt.DryRun {
		if onlyOurs {
			backup, err := backupFile(hooksPath)
			if err != nil {
				return false, nil, err
			}
			res.Backups[hooksPath] = backup
			if err := os.Remove(hooksPath); err != nil {
				return false, nil, fmt.Errorf("remove %s: %w", hooksPath, err)
			}
		} else {
			backup, err := backupFile(hooksPath)
			if err != nil {
				return false, nil, err
			}
			res.Backups[hooksPath] = backup
			if err := writeJSONFile(hooksPath, raw); err != nil {
				return false, nil, err
			}
		}
		removeScript(opt.LavHome, "cursor-hook.sh")
	}
	return true, preview, nil
}

// --- shared helpers --------------------------------------------------------

// entryMatchesScript mirrors installer.hasCommandPrefix's own matching rule
// so uninstall recognizes exactly the entries install would have recognized
// as already-present on a second run.
func entryMatchesScript(item any, nestedKey, scriptPath string) bool {
	m, ok := item.(map[string]any)
	if !ok {
		return false
	}
	if nestedKey == "" {
		cmd, _ := m["command"].(string)
		return strings.HasPrefix(cmd, scriptPath)
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
		return nil, false, fmt.Errorf("parse %s: %w (refusing to touch a file this cannot safely round-trip)", path, err)
	}
	return m, true, nil
}

func writeJSONFile(path string, v map[string]any) error {
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

// backupFile copies path aside before it gets rewritten. Returns the backup
// path, or "" if path does not exist (nothing to back up).
func backupFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s for backup: %w", path, err)
	}
	backup := fmt.Sprintf("%s.bak-%s", path, time.Now().UTC().Format("20060102T150405Z"))
	if err := os.WriteFile(backup, data, 0o600); err != nil {
		return "", fmt.Errorf("write backup %s: %w", backup, err)
	}
	return backup, nil
}

// removeScript deletes a forwarder script "lav init" wrote. Missing is not
// an error — uninstall is idempotent.
func removeScript(lavHome, name string) {
	_ = os.Remove(filepath.Join(lavHome, "bin", name))
}
