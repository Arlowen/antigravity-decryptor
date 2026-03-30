package main

import (
	"fmt"
	"io"
	"os"

	"github.com/pika/antigravity-decryptor/internal/app"
)

const usage = `antigravity-decryptor — export Antigravity conversation trajectories

Usage:
  antigravity-decryptor [flags] <cascadeId|path/to/<cascadeId>.pb>
  antigravity-decryptor list [flags]

Subcommands:
  list        List all visible cascade trajectory summaries (raw JSON)
  <input>     Export a specific conversation by cascadeId or .pb file path

Flags:
  --format    Output format: raw (default), normalized, markdown
  --output    Output file path (default: stdout)
  --ls-binary Path to language server binary (default: system default, or ANTIGRAVITY_LS_PATH env var)
  --verbose   Print debug logs to stderr
  --help      Show this help

Examples:
  # Export raw JSON by cascadeId
  antigravity-decryptor 762506a2-5119-41e2-b4d9-98c944135b68

  # Export by .pb file path
  antigravity-decryptor ~/.gemini/antigravity/conversations/762506a2-5119-41e2-b4d9-98c944135b68.pb

  # Export normalized JSON to file
  antigravity-decryptor --format normalized --output out.json 762506a2-5119-41e2-b4d9-98c944135b68

  # Export markdown transcript
  antigravity-decryptor --format markdown 762506a2-5119-41e2-b4d9-98c944135b68

  # List all conversations
  antigravity-decryptor list

  # Use a custom language server binary
  antigravity-decryptor --ls-binary /path/to/language_server_macos_arm 762506a2-5119-41e2-b4d9-98c944135b68
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	var (
		format   string
		output   string
		lsBinary string
		verbose  bool
	)

	// 简单手写 flag 解析，避免引入额外依赖
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(usage)
			return nil
		case arg == "--verbose" || arg == "-v":
			verbose = true
		case arg == "--format" || arg == "-f":
			i++
			if i >= len(args) {
				return fmt.Errorf("--format requires a value")
			}
			format = args[i]
		case len(arg) > 9 && arg[:9] == "--format=":
			format = arg[9:]
		case arg == "--output" || arg == "-o":
			i++
			if i >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			output = args[i]
		case len(arg) > 9 && arg[:9] == "--output=":
			output = arg[9:]
		case arg == "--ls-binary":
			i++
			if i >= len(args) {
				return fmt.Errorf("--ls-binary requires a value")
			}
			lsBinary = args[i]
		case len(arg) > 12 && arg[:12] == "--ls-binary=":
			lsBinary = arg[12:]
		case len(arg) > 0 && arg[0] != '-':
			positional = append(positional, arg)
		default:
			return fmt.Errorf("unknown flag: %s (use --help for usage)", arg)
		}
	}

	if len(positional) == 0 {
		fmt.Print(usage)
		return nil
	}

	// list 子命令
	if positional[0] == "list" {
		var w io.Writer = os.Stdout
		if output != "" {
			f, err := os.Create(output)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer f.Close()
			w = f
		}
		return app.ListConversations(lsBinary, w, verbose)
	}

	// export 子命令
	if len(positional) > 1 {
		return fmt.Errorf("too many arguments; expected exactly one cascadeId or .pb path")
	}

	cfg := app.RunConfig{
		Input:    positional[0],
		Format:   app.OutputFormat(format),
		Output:   output,
		LSBinary: lsBinary,
		Verbose:  verbose,
	}

	return app.Run(cfg)
}
