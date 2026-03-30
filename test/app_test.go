package test

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pika/antigravity-decryptor/internal/app"
	"github.com/pika/antigravity-decryptor/internal/model"
)

func TestFirstUserInputTitleTruncatesByRunes(t *testing.T) {
	title := app.FirstUserInputTitle([]model.NormalizedStep{
		{
			Type: "CORTEX_STEP_TYPE_USER_INPUT",
			Text: strings.Repeat("你", 61),
		},
	})

	if !utf8.ValidString(title) {
		t.Fatalf("title is not valid UTF-8: %q", title)
	}
	if strings.ContainsRune(title, utf8.RuneError) {
		t.Fatalf("title contains replacement rune: %q", title)
	}

	trimmed := strings.TrimSuffix(title, "...")
	if got := utf8.RuneCountInString(trimmed); got != 60 {
		t.Fatalf("expected 60 runes before ellipsis, got %d in %q", got, title)
	}
}
