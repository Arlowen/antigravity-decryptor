package server

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	// DefaultBinaryPath 是官方 language server 二进制的默认路径。
	// 可通过环境变量 ANTIGRAVITY_LS_PATH 或 CLI 参数 --ls-binary 覆盖。
	DefaultBinaryPath = "/Applications/Antigravity.app/Contents/Resources/app/extensions/antigravity/bin/language_server_macos_arm"

	// startupTimeout 等待 language server 就绪的最长时间。
	startupTimeout = 30 * time.Second
)

// Server 代表一个正在运行的 language server 实例（或已有的复用实例）。
type Server struct {
	HTTPPort int
}

// Close 为兼容调用方保留；standalone server 需要跨命令复用，因此这里不主动关闭。
func (s *Server) Close() {}

// AcquireServer 优先复用已有的 language server（读 discovery json），
// 如果没有可用服务则拉起新进程。
//
// binaryPath 为空时使用 DefaultBinaryPath；也可通过环境变量 ANTIGRAVITY_LS_PATH 覆盖。
func AcquireServer(binaryPath string) (*Server, error) {
	if binaryPath == "" {
		if env := os.Getenv("ANTIGRAVITY_LS_PATH"); env != "" {
			binaryPath = env
		} else {
			binaryPath = DefaultBinaryPath
		}
	}

	// 1. 尝试复用已有服务
	if port, err := FindHTTPPortFromDiscovery(); err == nil {
		if isServerAlive(port) {
			return &Server{HTTPPort: port}, nil
		}
		// discovery 文件存在但进程已死，继续拉起新进程
	}

	// 2. 拉起新的 language server
	return launchServer(binaryPath)
}

// isServerAlive 对 GetAllCascadeTrajectories 发一个简单 POST，判断服务是否存活。
func isServerAlive(port int) bool {
	url := fmt.Sprintf("http://127.0.0.1:%d/exa.language_server_pb.LanguageServerService/GetAllCascadeTrajectories", port)
	req, err := http.NewRequest("POST", url, strings.NewReader("{}"))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func waitForServerAlive(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if isServerAlive(port) {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("language server on port %d did not become ready within %s", port, timeout)
}

// launchServer 拉起新的 language server 进程，等待 discovery 文件出现后返回。
func launchServer(binaryPath string) (*Server, error) {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("language server binary not found at %s (set ANTIGRAVITY_LS_PATH to override): %w", binaryPath, err)
	}

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", os.DevNull, err)
	}
	defer devNull.Close()

	cmd := exec.Command(
		binaryPath,
		"-standalone",
		"-persistent_mode",
		"-override_ide_name=antigravity",
		"-override_ide_version=0.0.0",
	)
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start language server: %w", err)
	}

	port, err := WaitForHTTPPortFromDiscoveryPID(cmd.Process.Pid, startupTimeout)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	if err := waitForServerAlive(port, 5*time.Second); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}

	_ = cmd.Process.Release()
	return &Server{HTTPPort: port}, nil
}
