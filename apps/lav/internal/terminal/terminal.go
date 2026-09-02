// Package terminal opens a terminal window at a session's working
// directory. Only meaningful when the daemon runs natively on the host
// (service.Install's whole point): the process spawns the terminal itself,
// no separate helper process needed.
package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// linuxCandidates are tried in order when $TERMINAL is unset. Each entry's
// flag (if any) sets the working directory explicitly; entries with an
// empty flag rely on the spawned process inheriting exec.Cmd.Dir instead.
var linuxCandidates = []struct {
	bin  string
	flag string
}{
	{"gnome-terminal", "--working-directory="},
	{"konsole", "--workdir"},
	{"x-terminal-emulator", ""},
	{"xterm", ""},
}

// Open launches a new terminal window at dir.
func Open(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("stat %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	switch runtime.GOOS {
	case "darwin":
		// `open -a Terminal <dir>` opens a new Terminal.app window with dir
		// as its working directory, same as double-clicking a folder.
		return exec.Command("open", "-a", "Terminal", dir).Start()
	case "linux":
		return openLinux(dir)
	default:
		return fmt.Errorf("no terminal launcher for GOOS=%s", runtime.GOOS)
	}
}

// openLinux is best-effort and not live-verified (built and reviewed on
// macOS) — see docs/sdd/specs/native-host-runtime.md Validation.
func openLinux(dir string) error {
	if term := os.Getenv("TERMINAL"); term != "" {
		cmd := exec.Command(term)
		cmd.Dir = dir
		return cmd.Start()
	}
	for _, c := range linuxCandidates {
		path, err := exec.LookPath(c.bin)
		if err != nil {
			continue
		}
		var cmd *exec.Cmd
		if c.flag != "" {
			cmd = exec.Command(path, c.flag+dir)
		} else {
			cmd = exec.Command(path)
			cmd.Dir = dir
		}
		return cmd.Start()
	}
	return fmt.Errorf("no terminal emulator found ($TERMINAL, gnome-terminal, konsole, x-terminal-emulator, xterm)")
}
