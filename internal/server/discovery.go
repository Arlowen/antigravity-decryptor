package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DiscoveryFile 是 language server 写出的 ls_*.json 的结构。
// 字段名是根据日志行为推断的，宽松反序列化，遇到未知字段不报错。
type DiscoveryFile struct {
	HTTPPort  int    `json:"httpPort"`
	HTTPSPort int    `json:"httpsPort"`
	PID       int    `json:"pid"`
	CSRFToken string `json:"csrfToken"`
}

type discoveryRecord struct {
	File    string
	ModTime time.Time
	Data    DiscoveryFile
}

// daemonDir 返回 ~/.gemini/antigravity/daemon/
func daemonDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gemini", "antigravity", "daemon")
}

// FindHTTPPortFromDiscovery 优先读 ~daemon/ls_*.json，返回 HTTP 端口。
// 如果没有 json 文件，返回 0, ErrNoDiscovery。
func FindHTTPPortFromDiscovery() (int, error) {
	record, err := latestDiscoveryRecord(func(record discoveryRecord) bool {
		return record.Data.HTTPPort != 0
	})
	if err != nil {
		return 0, err
	}
	return record.Data.HTTPPort, nil
}

func WaitForHTTPPortFromDiscoveryPID(pid int, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		record, err := latestDiscoveryRecord(func(record discoveryRecord) bool {
			return record.Data.PID == pid && record.Data.HTTPPort != 0
		})
		switch {
		case err == nil:
			return record.Data.HTTPPort, nil
		case err != nil && err != ErrNoDiscovery:
			return 0, err
		}
		time.Sleep(250 * time.Millisecond)
	}

	return 0, fmt.Errorf("language server did not publish discovery info for pid %d within %s", pid, timeout)
}

func latestDiscoveryRecord(match func(record discoveryRecord) bool) (discoveryRecord, error) {
	dir := daemonDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return discoveryRecord{}, fmt.Errorf("cannot read daemon dir %s: %w", dir, err)
	}

	var latest discoveryRecord
	found := false

	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "ls_") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		file := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(file)
		if err != nil {
			return discoveryRecord{}, fmt.Errorf("read discovery file %s: %w", file, err)
		}

		var df DiscoveryFile
		if err := json.Unmarshal(data, &df); err != nil {
			return discoveryRecord{}, fmt.Errorf("parse discovery file %s: %w", file, err)
		}

		record := discoveryRecord{
			File:    file,
			ModTime: info.ModTime(),
			Data:    df,
		}
		if !match(record) {
			continue
		}
		if !found || record.ModTime.After(latest.ModTime) {
			latest = record
			found = true
		}
	}

	if !found {
		return discoveryRecord{}, ErrNoDiscovery
	}

	return latest, nil
}

// ErrNoDiscovery 没有找到 discovery json 文件。
var ErrNoDiscovery = fmt.Errorf("no language server discovery file found")
