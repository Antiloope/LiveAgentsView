// Package pilotwire is the protocol and on-disk layout shared between the
// daemon (internal/pilot, the client) and the detached pilot-runner shim
// (internal/pilotrunner, the server) that keeps a piloted session's child
// process alive across a daemon restart. The two only ever talk over a
// filesystem-local Unix domain socket — never a network port — so this
// package adds no new network-reachable surface.
package pilotwire

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// pilotDir is where every session's socket, durable transcript and
// stream-offset files live, under the daemon's own data directory.
func pilotDir(lavHome string) string {
	return filepath.Join(lavHome, "pilot")
}

// SocketPath is the Unix domain socket a running pilot-runner listens on for
// session id. Its mere existence and dial-ability is how the daemon tells a
// still-running detached process apart from one that already exited.
func SocketPath(lavHome, id string) string {
	return filepath.Join(pilotDir(lavHome), id+".sock")
}

// TranscriptPath is where a session's raw provider stdout lines are appended
// durably, one per line, independent of the daemon process's own lifetime.
func TranscriptPath(lavHome, id string) string {
	return filepath.Join(pilotDir(lavHome), id+".jsonl")
}

// OffsetPath tracks the last transcript seq the daemon has fully processed
// (persisted to SQLite and broadcast), so a reconnect after a restart can
// ask the runner to replay only what it missed — no duplicates, no drops.
func OffsetPath(lavHome, id string) string {
	return filepath.Join(pilotDir(lavHome), id+".offset")
}

// StderrPath is where a session's child process's stderr is captured, kept
// only for debugging — not part of the transcript protocol.
func StderrPath(lavHome, id string) string {
	return filepath.Join(pilotDir(lavHome), id+".stderr.log")
}

// EnsureDir creates the shared pilot directory.
func EnsureDir(lavHome string) error {
	return os.MkdirAll(pilotDir(lavHome), 0o755)
}

// ClientMsg is sent from the daemon to a pilot-runner over the control
// socket.
type ClientMsg struct {
	// Op is "attach" (subscribe, replaying everything after Since),
	// "stdin" (relay Data to the child process's stdin), or "kill" (send
	// SIGKILL to the child — Cancel/Interrupt-via-kill for Cursor).
	Op    string `json:"op"`
	Since int64  `json:"since,omitempty"`
	Data  string `json:"data,omitempty"`
}

// ServerMsg is sent from a pilot-runner to the daemon: either one transcript
// line (Seq > 0) or a terminal notice that the child process exited.
type ServerMsg struct {
	Seq    int64  `json:"seq,omitempty"`
	Line   string `json:"line,omitempty"`
	Exited bool   `json:"exited,omitempty"`
	Code   int    `json:"code,omitempty"`
	Err    string `json:"err,omitempty"`
}

// maxLineBytes generously covers a single stdout line or wire frame — the
// same ceiling internal/pilot already uses for a provider CLI's own stdout.
const maxLineBytes = 16 * 1024 * 1024

// Encode writes v as one newline-delimited JSON frame.
func Encode(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = w.Write(b)
	return err
}

// NewScanner returns a line scanner sized for a wire connection or the
// durable transcript file.
func NewScanner(r io.Reader) *bufio.Scanner {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	return sc
}

// ReadOffset returns the last processed seq for id, or 0 if none was ever
// recorded (a fresh session, or one whose offset file was lost).
func ReadOffset(lavHome, id string) int64 {
	data, err := os.ReadFile(OffsetPath(lavHome, id))
	if err != nil {
		return 0
	}
	var seq int64
	if _, err := fmt.Sscanf(string(data), "%d", &seq); err != nil {
		return 0
	}
	return seq
}

// WriteOffset durably records the last processed seq for id, atomically
// (write-then-rename) so a crash mid-write never leaves a corrupt value.
func WriteOffset(lavHome, id string, seq int64) error {
	path := OffsetPath(lavHome, id)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(fmt.Sprintf("%d", seq)), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
