// Package service installs the daemon as a real OS-level user service:
// launchd on macOS, systemd --user on Linux. It replaces the Docker
// restart-policy stand-in with the native mechanism the stack decision
// already commits to.
package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Label is both the launchd job label and the basis for its plist filename.
const Label = "dev.liveagentsview.lav"

type Options struct {
	// BinaryPath is the absolute path to the installed lav binary the
	// service definition points at.
	BinaryPath string
	// LavHome is LiveAgentsView's data directory, passed to the service as
	// LAV_HOME. Logs are written under LavHome/logs.
	LavHome string
	Port    string
	// PATH is baked into the service definition as-is. Both launchd and
	// systemd --user start services with their own minimal PATH, not the
	// installing shell's — internal/pilot spawns `claude`/`agent` by bare
	// name, so without this the daemon can't find them even though the
	// installing user's shell can. Callers pass os.Getenv("PATH") captured
	// at `lav service install` time, when it still reflects that shell.
	PATH   string
	DryRun bool
}

// defaultPATH is used when the caller has no PATH to capture (PATH is
// always set in a real shell, but service.Install has no other fallback).
func defaultPATH(home string) string {
	return home + "/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
}

type Result struct {
	Preview []string
	// Path is the service definition file that was written (plist or unit).
	Path string
}

// Install registers the daemon as a user service for the current GOOS.
// Idempotent: re-running replaces the previous definition and re-registers.
func Install(opt Options) (Result, error) {
	switch runtime.GOOS {
	case "darwin":
		return installDarwin(opt)
	case "linux":
		return installLinux(opt)
	default:
		return Result{}, fmt.Errorf("no native service support for GOOS=%s", runtime.GOOS)
	}
}

// --- macOS (launchd) -------------------------------------------------------

const darwinPlistTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%[1]s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%[2]s</string>
		<string>serve</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>LAV_HOME</key>
		<string>%[3]s</string>
		<key>LAV_PORT</key>
		<string>%[4]s</string>
		<key>PATH</key>
		<string>%[5]s</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%[3]s/logs/lav.out.log</string>
	<key>StandardErrorPath</key>
	<string>%[3]s/logs/lav.err.log</string>
</dict>
</plist>
`

func installDarwin(opt Options) (Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Result{}, fmt.Errorf("resolve home dir: %w", err)
	}
	plistPath := filepath.Join(home, "Library", "LaunchAgents", Label+".plist")
	logsDir := filepath.Join(opt.LavHome, "logs")
	envPath := opt.PATH
	if envPath == "" {
		envPath = defaultPATH(home)
	}
	content := fmt.Sprintf(darwinPlistTmpl, Label, opt.BinaryPath, opt.LavHome, opt.Port, envPath)

	preview := []string{
		fmt.Sprintf("launchd (%s):", plistPath),
		fmt.Sprintf("  - point at %s serve, LAV_HOME=%s LAV_PORT=%s", opt.BinaryPath, opt.LavHome, opt.Port),
		fmt.Sprintf("  - PATH=%s", envPath),
		fmt.Sprintf("  - logs under %s", logsDir),
		"  - RunAtLoad + KeepAlive, bootstrapped into gui/<uid> (starts now and on every login)",
	}
	if opt.DryRun {
		return Result{Preview: preview, Path: plistPath}, nil
	}

	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create %s: %w", logsDir, err)
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create %s: %w", filepath.Dir(plistPath), err)
	}
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", plistPath, err)
	}

	target := fmt.Sprintf("gui/%d", os.Getuid())
	// Ignore bootout errors: expected to fail when nothing was loaded yet
	// (first install). Re-running the install must not require the caller
	// to uninstall first.
	exec.Command("launchctl", "bootout", target, plistPath).Run() //nolint:errcheck
	if out, err := exec.Command("launchctl", "bootstrap", target, plistPath).CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("launchctl bootstrap: %w: %s", err, out)
	}
	if out, err := exec.Command("launchctl", "enable", target+"/"+Label).CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("launchctl enable: %w: %s", err, out)
	}

	return Result{Preview: preview, Path: plistPath}, nil
}

// --- Linux (systemd --user) -------------------------------------------------

const linuxUnitTmpl = `[Unit]
Description=LiveAgentsView daemon
After=network.target

[Service]
ExecStart=%s serve
Environment=LAV_HOME=%s
Environment=LAV_PORT=%s
Environment=PATH=%s
Restart=on-failure

[Install]
WantedBy=default.target
`

func installLinux(opt Options) (Result, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Result{}, fmt.Errorf("resolve home dir: %w", err)
	}
	unitPath := filepath.Join(home, ".config", "systemd", "user", "lav.service")
	envPath := opt.PATH
	if envPath == "" {
		envPath = defaultPATH(home)
	}
	content := fmt.Sprintf(linuxUnitTmpl, opt.BinaryPath, opt.LavHome, opt.Port, envPath)

	preview := []string{
		fmt.Sprintf("systemd --user (%s):", unitPath),
		fmt.Sprintf("  - point at %s serve, LAV_HOME=%s LAV_PORT=%s", opt.BinaryPath, opt.LavHome, opt.Port),
		fmt.Sprintf("  - PATH=%s", envPath),
		"  - daemon-reload, then enable --now lav.service",
	}
	if opt.DryRun {
		return Result{Preview: preview, Path: unitPath}, nil
	}

	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create %s: %w", filepath.Dir(unitPath), err)
	}
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", unitPath, err)
	}

	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("systemctl daemon-reload: %w: %s", err, out)
	}
	if out, err := exec.Command("systemctl", "--user", "enable", "--now", "lav.service").CombinedOutput(); err != nil {
		return Result{}, fmt.Errorf("systemctl enable --now: %w: %s", err, out)
	}

	return Result{Preview: preview, Path: unitPath}, nil
}
