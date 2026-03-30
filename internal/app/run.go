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

// ListConversations 列出所有可见的 cascade summaries（用于 list 子命令）。
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
	rawJSON, err := client.GetAllCascadeTrajectories()
	if err != nil {
		return fmt.Errorf("GetAllCascadeTrajectories failed: %w", err)
	}

	return export.WriteRawJSON(w, rawJSON)
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
