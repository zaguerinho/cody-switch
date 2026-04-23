package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zaguerinho/claude-switch/agent-hub/internal/daemon"
	"github.com/zaguerinho/claude-switch/agent-hub/internal/server"
	"github.com/zaguerinho/claude-switch/agent-hub/internal/store"
)

var foreground bool

func newServeCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the agent-hub server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(version)
		},
	}
	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground (don't daemonize)")
	return cmd
}

func runServe(version string) error {
	base := baseDir()
	pf := daemon.NewPIDFile(base)

	// Check for existing server
	if running, pid := pf.IsRunning(); running {
		msg := fmt.Sprintf("server already running (PID %d)", pid)
		if jsonOutput {
			return printError(msg)
		}
		fmt.Println(msg)
		return nil
	}
	pf.RemoveIfStale()

	if !foreground {
		// Re-exec as daemon
		exe, err := os.Executable()
		if err != nil {
			return printError("cannot find executable: " + err.Error())
		}
		args := []string{"serve", "--foreground",
			"--api-port", strconv.Itoa(resolveAPIPort()),
			"--ui-port", strconv.Itoa(uiPort),
		}
		proc := exec.Command(exe, args...)
		proc.Stdout = nil
		proc.Stderr = nil
		proc.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

		if err := proc.Start(); err != nil {
			return printError("start daemon: " + err.Error())
		}

		// Wait for the daemon to be ready (poll health endpoint)
		client := &http.Client{Timeout: 500 * time.Millisecond}
		healthURL := fmt.Sprintf("http://127.0.0.1:%d/api/v1/health", resolveAPIPort())
		ready := false
		for i := 0; i < 20; i++ {
			time.Sleep(100 * time.Millisecond)
			resp, err := client.Get(healthURL)
			if err == nil {
				resp.Body.Close()
				ready = true
				break
			}
		}
		if !ready {
			return printError("server started but not responding — check logs")
		}

		if jsonOutput {
			fmt.Printf(`{"ok":true,"data":{"pid":%d,"api_port":%d,"ui_port":%d}}`+"\n",
				proc.Process.Pid, resolveAPIPort(), uiPort)
		} else {
			fmt.Printf("agent-hub started (PID %d)\n", proc.Process.Pid)
			fmt.Printf("  API:       http://127.0.0.1:%d\n", resolveAPIPort())
			fmt.Printf("  Dashboard: http://127.0.0.1:%d\n", uiPort)
		}
		return nil
	}

	// Foreground mode: run the server
	fs, err := store.New(base)
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}
	if err := fs.Init(); err != nil {
		log.Printf("store init warning: %v", err)
	}

	if err := pf.Write(os.Getpid()); err != nil {
		return fmt.Errorf("write PID: %w", err)
	}
	defer pf.Remove()

	srv := server.New(fs, resolveAPIPort(), uiPort, version)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	return srv.Run(ctx)
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the agent-hub server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStop()
		},
	}
}

func runStop() error {
	pf := daemon.NewPIDFile(baseDir())
	running, pid := pf.IsRunning()
	if !running {
		if pid > 0 {
			pf.Remove()
		}
		msg := "server is not running"
		if jsonOutput {
			return printError(msg)
		}
		fmt.Println(msg)
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return printError("find process: " + err.Error())
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return printError("send SIGTERM: " + err.Error())
	}

	// Wait for shutdown
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if r, _ := pf.IsRunning(); !r {
			break
		}
	}

	pf.Remove()
	if jsonOutput {
		fmt.Printf(`{"ok":true,"data":{"stopped":%d}}` + "\n", pid)
	} else {
		fmt.Printf("agent-hub stopped (PID %d)\n", pid)
	}
	return nil
}

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check if the server is running",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := apiCall("GET", "/api/v1/health", nil)
			return handleResponse(resp, err, func(data any) {
				fmt.Println("agent-hub is running")
			})
		},
	}
}
