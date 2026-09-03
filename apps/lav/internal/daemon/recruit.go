// Handlers backing the recruit panel's three data needs that a browser
// cannot supply on its own: a real native folder picker, the target
// directory's actual git branches, and Cursor's own live model catalog.
package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// handlePickDirectory pops the real macOS folder-choose dialog (the same
// NSOpenPanel Finder itself uses) on the host and returns the chosen
// absolute path. A browser <input type=file> cannot do this: it never
// exposes an absolute filesystem path, by design. A cancelled dialog is not
// an error — it is reported as 204 so the caller can just no-op.
func (s *Server) handlePickDirectory(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "Choose a repository or worktree")`)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && strings.Contains(string(ee.Stderr), "User canceled") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "open folder picker: "+err.Error(), http.StatusInternalServerError)
		return
	}
	path := strings.TrimRight(strings.TrimSpace(string(out)), "/")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"path": path})
}

// handleListBranches returns whether ?cwd= is a git repository, its current
// branch and its local branch list. A directory that is not a git repo is
// not an error — it can still host a shared territory — so this reports
// is_repo:false with empty branches rather than failing, letting the
// recruit panel explain why own territory is unavailable there.
func (s *Server) handleListBranches(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		http.Error(w, "cwd is required", http.StatusBadRequest)
		return
	}

	resp := struct {
		IsRepo   bool     `json:"is_repo"`
		Current  string   `json:"current"`
		Branches []string `json:"branches"`
	}{Branches: []string{}}

	w.Header().Set("Content-Type", "application/json")
	if err := exec.Command("git", "-C", cwd, "rev-parse", "--is-inside-work-tree").Run(); err != nil {
		json.NewEncoder(w).Encode(resp)
		return
	}
	resp.IsRepo = true
	if out, err := exec.Command("git", "-C", cwd, "branch", "--show-current").Output(); err == nil {
		resp.Current = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("git", "-C", cwd, "branch", "--format=%(refname:short)").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			if line = strings.TrimSpace(line); line != "" {
				resp.Branches = append(resp.Branches, line)
			}
		}
	}
	json.NewEncoder(w).Encode(resp)
}

// cursorModelOption is one entry from `agent --list-models` — the id passed
// to --model, and its human label as the CLI itself names it (never a
// quality claim this daemon invented).
type cursorModelOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

const cursorModelsTTL = 5 * time.Minute

// cursorModelsCache spawns `agent --list-models` (~220 entries, confirmed
// live against the installed CLI) at most once per TTL — the recruit panel
// may be opened many times a minute and the list changes rarely enough that
// a short cache beats a fresh subprocess every time.
type cursorModelsCache struct {
	mu        sync.Mutex
	models    []cursorModelOption
	fetchedAt time.Time
}

var cursorModelLine = regexp.MustCompile(`^([a-zA-Z0-9.\-]+)\s+-\s+(.+)$`)

func (c *cursorModelsCache) get() ([]cursorModelOption, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.models != nil && time.Since(c.fetchedAt) < cursorModelsTTL {
		return c.models, nil
	}

	out, err := exec.Command("agent", "--list-models").Output()
	if err != nil {
		return nil, err
	}
	models := []cursorModelOption{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		m := cursorModelLine.FindStringSubmatch(strings.TrimSpace(sc.Text()))
		if m == nil {
			continue // header/blank/"Tip: ..." lines — not a "<id> - <label>" row
		}
		models = append(models, cursorModelOption{ID: m[1], Label: m[2]})
	}
	c.models = models
	c.fetchedAt = time.Now()
	return models, nil
}

func (s *Server) handleCursorClasses(w http.ResponseWriter, r *http.Request) {
	models, err := s.cursorModels.get()
	if err != nil {
		http.Error(w, "list cursor models: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}
