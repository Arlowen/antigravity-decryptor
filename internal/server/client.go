package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 封装对本地 language server HTTP 接口的调用。
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建一个指向给定端口的 Client。
func NewClient(port int) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// post 发送 JSON POST 请求，返回原始响应 body。
func (c *Client) post(endpoint string, reqBody any) ([]byte, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + endpoint
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, url, string(data))
	}

	return data, nil
}

// GetAllCascadeTrajectories 调用
//   POST /exa.language_server_pb.LanguageServerService/GetAllCascadeTrajectories
// 返回原始 JSON bytes（宽松反序列化，由调用方解析）。
func (c *Client) GetAllCascadeTrajectories() ([]byte, error) {
	const endpoint = "/exa.language_server_pb.LanguageServerService/GetAllCascadeTrajectories"
	return c.post(endpoint, map[string]any{})
}

// GetCascadeTrajectory 调用
//   POST /exa.language_server_pb.LanguageServerService/GetCascadeTrajectory
// cascadeId 是 conversation UUID（等于 .pb 文件名 stem）。
// 返回原始 JSON bytes。
func (c *Client) GetCascadeTrajectory(cascadeID string) ([]byte, error) {
	if cascadeID == "" {
		return nil, fmt.Errorf("cascadeId must not be empty")
	}
	const endpoint = "/exa.language_server_pb.LanguageServerService/GetCascadeTrajectory"
	return c.post(endpoint, map[string]string{"cascadeId": cascadeID})
}
