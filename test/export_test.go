package test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pika/antigravity-decryptor/internal/export"
	"github.com/pika/antigravity-decryptor/internal/model"
)

func TestWriteMarkdownTranscriptFiltersInternalStepsByDefault(t *testing.T) {
	trajectory := &model.NormalizedTrajectory{
		CascadeID:    "cascade-1",
		TrajectoryID: "trajectory-1",
		Steps: []model.NormalizedStep{
			{Index: 0, Type: "CORTEX_STEP_TYPE_USER_INPUT", Text: "hello"},
			{Index: 1, Type: "CORTEX_STEP_TYPE_PLANNER_RESPONSE", Text: "visible answer"},
			{Index: 2, Type: "CORTEX_STEP_TYPE_PLANNER_RESPONSE", Text: "*Thinking:*\nsecret"},
			{Index: 3, Type: "CORTEX_STEP_TYPE_EPHEMERAL_MESSAGE", Text: "<EPHEMERAL_MESSAGE>"},
			{Index: 4, Type: "CORTEX_STEP_TYPE_CONVERSATION_HISTORY", Text: "history"},
			{Index: 5, Type: "CORTEX_STEP_TYPE_TASK_BOUNDARY", Text: "**Task**: hidden"},
		},
	}

	var buf bytes.Buffer
	if err := export.WriteMarkdownTranscript(&buf, trajectory, export.MarkdownOptions{}); err != nil {
		t.Fatalf("WriteMarkdownTranscript returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"hello", "visible answer"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q: %s", want, output)
		}
	}

	for _, unwanted := range []string{"secret", "<EPHEMERAL_MESSAGE>", "history", "**Task**: hidden"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("expected output to omit %q: %s", unwanted, output)
		}
	}
}

func TestWriteMarkdownTranscriptIncludesInternalStepsWhenRequested(t *testing.T) {
	trajectory := &model.NormalizedTrajectory{
		CascadeID:    "cascade-1",
		TrajectoryID: "trajectory-1",
		Steps: []model.NormalizedStep{
			{Index: 0, Type: "CORTEX_STEP_TYPE_PLANNER_RESPONSE", Text: "*Thinking:*\nsecret"},
			{Index: 1, Type: "CORTEX_STEP_TYPE_EPHEMERAL_MESSAGE", Text: "<EPHEMERAL_MESSAGE>"},
		},
	}

	var buf bytes.Buffer
	if err := export.WriteMarkdownTranscript(&buf, trajectory, export.MarkdownOptions{IncludeInternal: true}); err != nil {
		t.Fatalf("WriteMarkdownTranscript returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"secret", "<EPHEMERAL_MESSAGE>"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q: %s", want, output)
		}
	}
}
