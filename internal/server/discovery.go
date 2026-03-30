package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DiscoveryFile 是 language server 写出的 ls_*.json 的结构。
// 字段名是根据日志行为推断的，宽松反序列化，遇到未知字段不报错。
type DiscoveryFile struct {
	HTTPPort  int    `json:"httpPort"`
	HTTPSPort int    `json:"httpsPort"`
	PID       int    `json:"pid"`
	Token     string `json:"token"`
}

// daemonDir 返回 ~/.gemini/antigravity/daemon/
func daemonDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini", "antigravity", "daemon")
}

// FindHTTPPortFromDiscovery 优先读 ~daemon/ls_*.json，返回 HTTP 端口。
// 如果没有 json 文件，返回 0, ErrNoDiscovery。
func FindHTTPPortFromDiscovery() (int, error) {
	dir := daemonDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("cannot read daemon dir %s: %w", dir, err)
	}

	// 按文件名排序取最新
	var jsonFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "ls_") && strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, filepath.Join(dir, e.Name()))
		}
	}
	if len(jsonFiles) == 0 {
		return 0, ErrNoDiscovery
	}

	sort.Strings(jsonFiles)
	// 取最后一个（最新）
	latestFile := jsonFiles[len(jsonFiles)-1]

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return 0, fmt.Errorf("read discovery file %s: %w", latestFile, err)
	}

	var df DiscoveryFile
	if err := json.Unmarshal(data, &df); err != nil {
		return 0, fmt.Errorf("parse discovery file %s: %w", latestFile, err)
	}

	if df.HTTPPort == 0 {
		return 0, fmt.Errorf("discovery file %s has httpPort=0", latestFile)
	}

	return df.HTTPPort, nil
}

// portFromLogLine 从日志行提取 HTTP 端口，例如：
//   Language server listening on random port at 52094 for HTTP
var httpPortPattern = regexp.MustCompile(`Language server listening on random port at (\d+) for HTTP\b`)

// ParseHTTPPortFromLog 从日志行（可以是 stdout 流或 log 文件内容）提取 HTTP 端口。
func ParseHTTPPortFromLog(line string) (int, bool) {
	m := httpPortPattern.FindStringSubmatch(line)
	if m == nil {
		return 0, false
	}
	port, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return port, true
}

// ErrNoDiscovery 没有找到 discovery json 文件。
var ErrNoDiscovery = fmt.Errorf("no language server discovery file found")
