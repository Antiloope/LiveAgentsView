// Package territory sets up and tears down where a character's process
// actually runs: either a git worktree LiveAgentsView administers, or a
// directory the user picked, left exactly as it is. LiveAgentsView never
// runs a git command against a shared territory — not here, not anywhere.
package territory

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/model"
)

// Spec is what the caller asks for when a character is created.
type Spec struct {
	Mode        model.TerritoryMode
	SourceRepo  string // the directory the user picked
	Branch      string // desired branch name; empty picks one automatically for Own
	CharacterID string
}

// IsGitRepo reports whether dir is inside a git working tree.
func IsGitRepo(dir string) bool {
	return exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree").Run() == nil
}

func worktreesRoot(lavHome string) string { return filepath.Join(lavHome, "worktrees") }

// Prepare sets up a character's territory. Shared mode only validates the
// directory exists — no git command touches it. Own mode creates a git
// worktree under worktreesRoot(lavHome)/<repo-basename>/<character-id>, on
// spec.Branch if given (a new branch unless one by that name already
// exists) or an auto-generated one otherwise.
func Prepare(lavHome string, spec Spec) (model.Territory, error) {
	if spec.Mode == model.TerritoryShared {
		info, err := os.Stat(spec.SourceRepo)
		if err != nil || !info.IsDir() {
			return model.Territory{}, fmt.Errorf("territory must be an existing directory")
		}
		return model.Territory{
			Mode: model.TerritoryShared, Path: spec.SourceRepo, Source: spec.SourceRepo,
			Branch: currentBranch(spec.SourceRepo),
		}, nil
	}

	if !IsGitRepo(spec.SourceRepo) {
		return model.Territory{}, fmt.Errorf("own territory needs a git repository; %s is not one", spec.SourceRepo)
	}
	branch := strings.TrimSpace(spec.Branch)
	if branch == "" {
		branch = "lav/" + spec.CharacterID[:8]
	}
	if busy, where := checkedOutElsewhere(spec.SourceRepo, branch); busy {
		return model.Territory{}, fmt.Errorf("branch %s is already checked out at %s", branch, where)
	}

	path := filepath.Join(worktreesRoot(lavHome), filepath.Base(spec.SourceRepo), spec.CharacterID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return model.Territory{}, fmt.Errorf("create worktrees directory: %w", err)
	}

	// git worktree add [-b <new-branch>] <path> [<start-point>]: an existing
	// branch is checked out by naming it as the trailing start-point with no
	// -b; a new one is created (from HEAD, the default start-point when none
	// is given) via -b and must NOT also be repeated as the start-point —
	// git would try to resolve it as an existing ref and fail.
	args := []string{"-C", spec.SourceRepo, "worktree", "add", path}
	if branchExists(spec.SourceRepo, branch) {
		args = append(args, branch)
	} else {
		args = append(args, "-b", branch)
	}
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		return model.Territory{}, fmt.Errorf("git worktree add: %s", strings.TrimSpace(string(out)))
	}

	return model.Territory{Mode: model.TerritoryOwn, Path: path, Source: spec.SourceRepo, Branch: branch}, nil
}

// Remove deletes an own-territory worktree if it has no uncommitted
// changes, reporting removed=false (the worktree is left as-is) otherwise.
// A shared territory is never touched.
func Remove(t model.Territory) (removed bool, err error) {
	if t.Mode != model.TerritoryOwn {
		return false, nil
	}
	if _, err := os.Stat(t.Path); os.IsNotExist(err) {
		return true, nil
	}
	out, err := exec.Command("git", "-C", t.Path, "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("check worktree status: %w", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		return false, nil
	}
	source := t.Source
	if source == "" {
		source = sourceOf(t.Path)
	}
	if source == "" {
		return false, fmt.Errorf("could not resolve %s's source repository", t.Path)
	}
	if out, err := exec.Command("git", "-C", source, "worktree", "remove", t.Path).CombinedOutput(); err != nil {
		return false, fmt.Errorf("git worktree remove: %s", strings.TrimSpace(string(out)))
	}
	return true, nil
}

// SweepOrphans removes every own-territory worktree under lavHome whose
// character id (its directory's own name, one level under the per-repo
// folder) is not in known. A dirty orphan is left in place, same as a
// dismissed character's uncommitted worktree — this never discards work
// nobody asked it to.
func SweepOrphans(lavHome string, known map[string]bool) {
	repos, err := os.ReadDir(worktreesRoot(lavHome))
	if err != nil {
		return
	}
	for _, repo := range repos {
		if !repo.IsDir() {
			continue
		}
		repoDir := filepath.Join(worktreesRoot(lavHome), repo.Name())
		entries, err := os.ReadDir(repoDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || known[e.Name()] {
				continue
			}
			path := filepath.Join(repoDir, e.Name())
			Remove(model.Territory{Mode: model.TerritoryOwn, Path: path, Source: sourceOf(path)})
		}
	}
}

// sourceOf reads a worktree's ".git" file (which holds "gitdir: <source>/.git/worktrees/<name>")
// to find the repository it was created from — used when no character row
// (and so no persisted Source) exists for it any more.
func sourceOf(worktreePath string) string {
	data, err := os.ReadFile(filepath.Join(worktreePath, ".git"))
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	const prefix = "gitdir: "
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	gitdir := strings.TrimPrefix(line, prefix)
	idx := strings.Index(gitdir, string(filepath.Separator)+".git"+string(filepath.Separator)+"worktrees"+string(filepath.Separator))
	if idx == -1 {
		return ""
	}
	return gitdir[:idx]
}

func checkedOutElsewhere(repo, branch string) (bool, string) {
	out, err := exec.Command("git", "-C", repo, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return false, ""
	}
	ref := "branch refs/heads/" + branch
	var current string
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = strings.TrimPrefix(line, "worktree ")
		case line == ref:
			return true, current
		}
	}
	return false, ""
}

func branchExists(repo, branch string) bool {
	return exec.Command("git", "-C", repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

func currentBranch(repo string) string {
	if !IsGitRepo(repo) {
		return ""
	}
	out, err := exec.Command("git", "-C", repo, "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
