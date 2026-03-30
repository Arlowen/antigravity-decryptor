package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/pika/antigravity-decryptor/internal/model"
)

type MarkdownOptions struct {
	IncludeInternal bool
}

// WriteMarkdownTranscript 把归一化轨迹输出为可读 markdown。
// 默认只输出用户可见且有实质文本内容的 step；可通过选项包含内部步骤。
func WriteMarkdownTranscript(w io.Writer, t *model.NormalizedTrajectory, opts MarkdownOptions) error {
	fmt.Fprintf(w, "# Conversation Transcript\n\n")
	fmt.Fprintf(w, "- **cascadeId**: `%s`\n", t.CascadeID)
	fmt.Fprintf(w, "- **trajectoryId**: `%s`\n", t.TrajectoryID)
	fmt.Fprintf(w, "- **type**: `%s`\n", t.TrajectoryType)
	if len(t.WorkspaceURIs) > 0 {
		fmt.Fprintf(w, "- **workspaces**: %s\n", strings.Join(t.WorkspaceURIs, ", "))
	}
	fmt.Fprintf(w, "- **totalSteps**: %v\n", t.NumTotalSteps)
	fmt.Fprintf(w, "\n---\n\n")

	for _, step := range t.Steps {
		if !includeMarkdownStep(step, opts) {
			continue
		}
		role := stepRole(step.Type)
		if step.CreatedAt != "" {
			fmt.Fprintf(w, "### [%d] %s (%s)\n\n", step.Index, role, step.CreatedAt)
		} else {
			fmt.Fprintf(w, "### [%d] %s\n\n", step.Index, role)
		}
		fmt.Fprintf(w, "%s\n\n", step.Text)
	}
	return nil
}

func includeMarkdownStep(step model.NormalizedStep, opts MarkdownOptions) bool {
	if step.Text == "" {
		return false
	}
	if opts.IncludeInternal {
		return true
	}

	switch step.Type {
	case "CORTEX_STEP_TYPE_EPHEMERAL_MESSAGE",
		"CORTEX_STEP_TYPE_CONVERSATION_HISTORY",
		"CORTEX_STEP_TYPE_TASK_BOUNDARY",
		"CORTEX_STEP_TYPE_KNOWLEDGE_ARTIFACTS":
		return false
	case "CORTEX_STEP_TYPE_PLANNER_RESPONSE":
		return !strings.HasPrefix(step.Text, "*Thinking:*")
	default:
		return true
	}
}

// stepRole 将 step type 映射到可读标签。
func stepRole(stepType string) string {
	switch stepType {
	case "CORTEX_STEP_TYPE_USER_INPUT":
		return "👤 User"
	case "CORTEX_STEP_TYPE_PLANNER_RESPONSE":
		return "🤖 Assistant"
	case "CORTEX_STEP_TYPE_NOTIFY_USER":
		return "📢 Notify"
	case "CORTEX_STEP_TYPE_TASK_BOUNDARY":
		return "🏁 Task Boundary"
	case "CORTEX_STEP_TYPE_CONVERSATION_HISTORY":
		return "📜 History"
	case "CORTEX_STEP_TYPE_EPHEMERAL_MESSAGE":
		return "💬 Ephemeral"
	case "CORTEX_STEP_TYPE_KNOWLEDGE_ARTIFACTS":
		return "📚 Knowledge"
	default:
		if stepType != "" {
			return stepType
		}
		return "step"
	}
}
