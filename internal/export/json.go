package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// WriteRawJSON 把原始 JSON bytes 美化后写到 writer。
func WriteRawJSON(w io.Writer, rawJSON []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, rawJSON, "", "  "); err != nil {
		// 如果不是合法 JSON，原样输出
		_, err2 := w.Write(rawJSON)
		return err2
	}
	_, err := fmt.Fprintln(w, buf.String())
	return err
}

// WriteNormalizedJSON 把归一化对象序列化为美化 JSON 写到 writer。
func WriteNormalizedJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal normalized: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}
