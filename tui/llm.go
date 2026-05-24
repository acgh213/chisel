package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// llm request / response types
// ---------------------------------------------------------------------------

// LLMRequest is sent to the Python backend via NDJSON.
type LLMRequest struct {
	Op       string `json:"op"`
	Text     string `json:"text,omitempty"`
	Context  string `json:"context,omitempty"`
	Question string `json:"question,omitempty"`
	Guidance string `json:"guidance,omitempty"`
}

// LLMResponse is received from the Python backend via NDJSON.
type LLMResponse struct {
	Op     string `json:"op"`
	Result string `json:"result"`
	Tokens int    `json:"tokens"`
	Status string `json:"status"` // "ok", "streaming", "error"
}

// ---------------------------------------------------------------------------
// llm client
// ---------------------------------------------------------------------------

// LLMClient manages the chisel.py subprocess and NDJSON communication.
type LLMClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
	ready  bool
	sendMu sync.Mutex // serialises Send calls so goroutines don't race on stdout
}

// NewLLMClient spawns chisel.py as a subprocess. The project dir is passed
// via the CHISEL_PROJECT environment variable.
func NewLLMClient(projectDir string) (*LLMClient, error) {
	// Locate chisel.py relative to the current executable, or fall back to
	// looking in the project directory.
	pyPath := findChiselPy(projectDir)
	if pyPath == "" {
		return nil, fmt.Errorf("chisel.py not found")
	}

	cmd := exec.Command("python", pyPath)
	cmd.Env = append(os.Environ(), "CHISEL_PROJECT="+projectDir)
	cmd.Dir = projectDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	// Capture stderr for debugging.
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting chisel.py: %w", err)
	}

	lc := &LLMClient{
		cmd:   cmd,
		stdin: stdin,
		stdout: bufio.NewScanner(stdout),
	}

	// Wait for the ready signal.
	ready := make(chan bool, 1)
	go func() {
		for lc.stdout.Scan() {
			line := lc.stdout.Text()
			var resp LLMResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}
			if resp.Op == "ready" {
				ready <- true
				return
			}
		}
		ready <- false
	}()

	select {
	case ok := <-ready:
		if !ok {
			lc.Stop()
			return nil, fmt.Errorf("chisel.py failed to start")
		}
		lc.ready = true
	case <-time.After(5 * time.Second):
		lc.Stop()
		return nil, fmt.Errorf("chisel.py startup timed out")
	}

	return lc, nil
}

// findChiselPy looks for chisel.py next to the executable, then in the
// project directory, then in PATH-relative locations.
func findChiselPy(projectDir string) string {
	// Try alongside the executable.
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "chisel.py")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// Try in the project directory.
	candidate := filepath.Join(projectDir, "chisel.py")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	// Try the working directory.
	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "chisel.py")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// Send writes a request to chisel.py and returns a channel that receives
// every response line (including streaming chunks) and closes when the
// final "ok" or "error" response arrives. Only one Send may be in flight
// at a time — concurrent calls block on sendMu.
func (lc *LLMClient) Send(req LLMRequest) (<-chan LLMResponse, error) {
	lc.sendMu.Lock()

	lc.mu.Lock()
	if !lc.ready {
		lc.mu.Unlock()
		lc.sendMu.Unlock()
		return nil, fmt.Errorf("llm client not ready")
	}
	lc.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		lc.sendMu.Unlock()
		return nil, fmt.Errorf("marshalling request: %w", err)
	}

	if _, err := lc.stdin.Write(append(data, '\n')); err != nil {
		lc.sendMu.Unlock()
		return nil, fmt.Errorf("writing to chisel.py: %w", err)
	}

	ch := make(chan LLMResponse, 64)
	go func() {
		defer func() {
			close(ch)
			lc.sendMu.Unlock()
		}()
		for lc.stdout.Scan() {
			line := lc.stdout.Text()
			var resp LLMResponse
			if err := json.Unmarshal([]byte(line), &resp); err != nil {
				continue
			}
			ch <- resp
			if resp.Status == "ok" || resp.Status == "error" {
				return
			}
		}
	}()

	return ch, nil
}

// Stop terminates the chisel.py subprocess.
func (lc *LLMClient) Stop() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if lc.stdin != nil {
		lc.stdin.Close()
	}
	if lc.cmd != nil && lc.cmd.Process != nil {
		lc.cmd.Process.Kill()
		lc.cmd.Wait()
	}
	lc.ready = false
}

// Ready returns true if the backend is alive and accepting requests.
func (lc *LLMClient) Ready() bool {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.ready
}
