package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
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
	cmd      *exec.Cmd // 非 nil 说明是本次启动的，nil 说明是复用的
}

// Close 关闭本次启动的 language server 进程（如果是复用的则不做任何事）。
func (s *Server) Close() {
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

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
	return resp.StatusCode < 500
}

// launchServer 拉起新的 language server 进程，等待它输出 HTTP 端口后返回。
func launchServer(binaryPath string) (*Server, error) {
	if _, err := os.Stat(binaryPath); err != nil {
		return nil, fmt.Errorf("language server binary not found at %s (set ANTIGRAVITY_LS_PATH to override): %w", binaryPath, err)
	}

	cmd := exec.Command(
		binaryPath,
		"-standalone",
		"-persistent_mode",
		"-override_ide_name=antigravity",
		"-override_ide_version=0.0.0",
	)

	// 用 io.Pipe 把 stdout 和 stderr 合并为一个 reader
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		return nil, fmt.Errorf("start language server: %w", err)
	}

	// 进程结束后关闭 pipe writer，让 scanner 退出
	go func() {
		_ = cmd.Wait()
		_ = pw.Close()
	}()

	portCh := make(chan int, 1)
	errCh := make(chan error, 1)

	// 扫描合并后的输出流
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			if port, ok := ParseHTTPPortFromLog(line); ok {
				select {
				case portCh <- port:
				default:
				}
			}
		}
	}()

	// 同时也等 discovery 文件写出（language server 初始化完成后写）
	go func() {
		deadline := time.Now().Add(startupTimeout)
		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)
			if port, err := FindHTTPPortFromDiscovery(); err == nil {
				select {
				case portCh <- port:
				default:
				}
				return
			}
		}
		errCh <- fmt.Errorf("language server did not write discovery file within %s", startupTimeout)
	}()

	select {
	case port := <-portCh:
		// 等一小段时间让服务真正准备好
		time.Sleep(500 * time.Millisecond)
		return &Server{HTTPPort: port, cmd: cmd}, nil
	case e := <-errCh:
		_ = cmd.Process.Kill()
		return nil, e
	case <-time.After(startupTimeout):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("timeout waiting for language server to start (%s)", startupTimeout)
	}
}
