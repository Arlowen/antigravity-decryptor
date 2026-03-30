package app

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pika/antigravity-decryptor/internal/export"
	"github.com/pika/antigravity-decryptor/internal/model"
	"github.com/pika/antigravity-decryptor/internal/server"
)

// OutputFormat 控制导出格式。
type OutputFormat string

const (
	FormatRaw        OutputFormat = "raw"
	FormatNormalized OutputFormat = "normalized"
	FormatMarkdown   OutputFormat = "markdown"
)

// RunConfig 是 Run 的配置参数。
type RunConfig struct {
	// Input 是 cascadeId（UUID 字符串），或者 .pb 文件路径。
	Input string
	// Format 是输出格式：raw / normalized / markdown。
	Format OutputFormat
	// Output 是输出文件路径，空字符串表示写到 stdout。
	Output string
	// LSBinary 是 language server 二进制路径，空表示使用默认值或环境变量。
	LSBinary string
	// Verbose 控制是否输出额外日志。
	Verbose bool
}

// Run 是主入口：解析 cascadeId、启动/复用服务、调接口、导出。
func Run(cfg RunConfig) error {
	cascadeID, err := resolveCascadeID(cfg.Input)
	if err != nil {
		return err
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[info] cascadeId: %s\n", cascadeID)
	}

	// 获取 writer
	var w io.Writer
	if cfg.Output == "" {
		w = os.Stdout
	} else {
		f, err := os.Create(cfg.Output)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		w = f
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[info] output: %s\n", cfg.Output)
		}
	}

	// 获取 server（优先复用，否则启动新进程）
	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[info] acquiring language server...\n")
	}
	srv, err := server.AcquireServer(cfg.LSBinary)
	if err != nil {
		return fmt.Errorf("language server unavailable: %w", err)
	}
	defer srv.Close()

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[info] language server HTTP port: %d\n", srv.HTTPPort)
	}

	// 调 GetCascadeTrajectory
	client := server.NewClient(srv.HTTPPort)
	rawJSON, err := client.GetCascadeTrajectory(cascadeID)
	if err != nil {
		return fmt.Errorf("GetCascadeTrajectory failed: %w", err)
	}

	// 按格式导出
	switch cfg.Format {
	case FormatRaw, "":
		return export.WriteRawJSON(w, rawJSON)

	case FormatNormalized:
		normalized, err := model.NormalizeResponse(rawJSON)
		if err != nil {
			return fmt.Errorf("normalize response: %w", err)
		}
		return export.WriteNormalizedJSON(w, normalized)

	case FormatMarkdown:
		normalized, err := model.NormalizeResponse(rawJSON)
		if err != nil {
			return fmt.Errorf("normalize response: %w", err)
		}
		return export.WriteMarkdownTranscript(w, normalized)

	default:
		return fmt.Errorf("unknown format: %s (valid: raw, normalized, markdown)", cfg.Format)
	}
}

// ListConversations 列出所有可见的 cascade summaries（现在通过强制扫描物理文件获取全量）。
func ListConversations(lsBinary string, w io.Writer, verbose bool) error {
	if verbose {
		fmt.Fprintf(os.Stderr, "[info] acquiring language server...\n")
	}
	srv, err := server.AcquireServer(lsBinary)
	if err != nil {
		return fmt.Errorf("language server unavailable: %w", err)
	}
	defer srv.Close()

	client := server.NewClient(srv.HTTPPort)

	// 1. 获取所有物理 .pb UUID
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("user home dir: %w", err)
	}
	dir := filepath.Join(home, ".gemini", "antigravity", "conversations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read conversations dir: %w", err)
	}

	var uuids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".pb") {
			uuids = append(uuids, strings.TrimSuffix(e.Name(), ".pb"))
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "[info] found %d trajectories in local storage\n", len(uuids))
	}

	// 2. 构造与原生 GetAll 相同的返回值结构
	type summaryInfo struct {
		Summary       string   `json:"summary,omitempty"`
		StepCount     any      `json:"stepCount,omitempty"`
		Status        any      `json:"status,omitempty"`
		WorkspaceUris []string `json:"workspaceUris,omitempty"`
		CreatedTime   string   `json:"createdTime,omitempty"`
	}

	summaries := make(map[string]summaryInfo)

	for _, uuid := range uuids {
		rawJSON, err := client.GetCascadeTrajectory(uuid)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "[warn] failed to fetch cascade %s: %v\n", uuid, err)
			}
			continue
		}

		nt, err := model.NormalizeResponse(rawJSON)
		if err != nil || nt == nil {
			continue
		}

		// 尝试生成一句 summary（类似原生的逻辑）
		var title string
		var firstTime string
		for _, step := range nt.Steps {
			if firstTime == "" && step.CreatedAt != "" {
				firstTime = step.CreatedAt
			}
			if step.Type == "CORTEX_STEP_TYPE_TASK_BOUNDARY" && title == "" {
				// NormalizedText format: "**Task**: [name]\n[summary]"
				lines := strings.Split(step.Text, "\n")
				title = strings.TrimPrefix(lines[0], "**Task**: ")
			}
		}
		if title == "" {
			for _, step := range nt.Steps {
				if step.Type == "CORTEX_STEP_TYPE_USER_INPUT" && step.Text != "" {
					title = step.Text
					if len(title) > 60 {
						title = title[:60] + "..."
					}
					break
				}
			}
		}
		if title == "" {
			title = "Untitled Conversation"
		}

		summaries[uuid] = summaryInfo{
			Summary:       title,
			StepCount:     nt.NumTotalSteps,
			Status:        nt.Status,
			WorkspaceUris: nt.WorkspaceURIs,
			CreatedTime:   firstTime,
		}
	}

	result := map[string]any{
		"trajectorySummaries": summaries,
	}

	return export.WriteNormalizedJSON(w, result)
}

// resolveCascadeID 从输入解析 cascadeId：
//   - 如果输入是 .pb 文件路径，取文件名 stem
//   - 否则直接当做 cascadeId
func resolveCascadeID(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("input is required: provide a cascadeId (UUID) or a .pb file path")
	}

	// 如果是文件路径（包含路径分隔符，或者以 .pb 结尾，或者文件存在）
	if strings.HasSuffix(input, ".pb") || strings.Contains(input, string(filepath.Separator)) {
		base := filepath.Base(input)
		stem := strings.TrimSuffix(base, ".pb")
		if stem == "" {
			return "", fmt.Errorf("cannot extract cascadeId from path: %s", input)
		}
		return stem, nil
	}

	// 否则直接当 cascadeId
	return input, nil
}
