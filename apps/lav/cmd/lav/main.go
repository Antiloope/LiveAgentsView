// Command lav is both the daemon and its own CLI: `lav serve` runs the
// daemon, `lav status` is a quick CLI-only view without opening the
// browser, and `lav pilot-runner`/`lav pilot-mcp` are internal helpers a
// piloted session's process runs under — not meant to be invoked by hand.
package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/classifier"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/daemon"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotmcp"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/pilotrunner"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/service"
	"github.com/Antiloope/LiveAgentsView/apps/lav/internal/store"
	webassets "github.com/Antiloope/LiveAgentsView/apps/lav/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "serve":
		cmdServe()
	case "status":
		cmdStatus()
	case "service":
		cmdService(os.Args[2:])
	case "pilot-runner":
		if err := pilotrunner.Run(os.Args[2:]); err != nil {
			log.Fatalf("pilot-runner: %v", err)
		}
	case "pilot-mcp":
		if err := pilotmcp.Run(os.Args[2:]); err != nil {
			log.Fatalf("pilot-mcp: %v", err)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: lav <serve|status|service install [--dry-run]>")
}

// dataDir is LiveAgentsView's own data directory from this process's
// filesystem view: LAV_HOME when set (the Docker image sets it to /data,
// bind-mounted to the host's ~/.liveagentsview), else ~/.liveagentsview
// directly for a native, non-Docker run.
func dataDir() string {
	if v := os.Getenv("LAV_HOME"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".liveagentsview"
	}
	return filepath.Join(home, ".liveagentsview")
}

func port() string {
	if v := os.Getenv("LAV_PORT"); v != "" {
		return v
	}
	return "8420"
}

// selfBinary resolves this running binary's own real path, for piloted
// sessions to re-exec as "lav pilot-runner" — resolving symlinks the same
// way cmdService already does for the same reason (a service definition or
// a spawned pilot-runner both need a real, stable path, not one that
// depends on whatever symlink happened to be used to invoke `serve`).
func selfBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

func cmdServe() {
	dir := dataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Fatalf("create data dir %s: %v", dir, err)
	}
	st, err := store.Open(filepath.Join(dir, "lav.db"))
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	webFS, err := fs.Sub(webassets.Dist, "static")
	if err != nil {
		log.Fatalf("embed frontend: %v", err)
	}

	exe, err := selfBinary()
	if err != nil {
		log.Fatalf("resolve own binary path: %v", err)
	}

	srv := daemon.New(st, classifier.NewRules(), webFS, dir, exe)
	addr := "127.0.0.1:" + port()
	log.Printf("lav daemon listening on %s (data: %s)", addr, dir)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// cmdService registers the installed lav binary as a launchd (macOS) or
// systemd --user (Linux) service pointing at itself, so the daemon survives
// a host reboot without Docker's restart policy standing in for it.
func cmdService(args []string) {
	if len(args) == 0 || args[0] != "install" {
		fmt.Fprintln(os.Stderr, "usage: lav service install [--dry-run]")
		os.Exit(1)
	}

	dryRun := false
	for _, a := range args[1:] {
		if a == "--dry-run" {
			dryRun = true
		}
	}

	exe, err := selfBinary()
	if err != nil {
		log.Fatalf("resolve own binary path: %v", err)
	}

	res, err := service.Install(service.Options{
		BinaryPath: exe,
		LavHome:    dataDir(),
		Port:       port(),
		PATH:       os.Getenv("PATH"),
		DryRun:     dryRun,
	})
	if err != nil {
		log.Fatalf("lav service install: %v", err)
	}

	for _, line := range res.Preview {
		fmt.Println(line)
	}
	if dryRun {
		fmt.Println("\n(dry run — nothing written or registered)")
		return
	}
	fmt.Println("\nInstalled and started:", res.Path)
}

func cmdStatus() {
	url := fmt.Sprintf("http://127.0.0.1:%s/api/sessions", port())
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("could not reach the daemon at %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("daemon returned %s", resp.Status)
	}
	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		log.Fatalf("read response: %v", err)
	}
	fmt.Println()
}
