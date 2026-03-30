package model

import (
	"encoding/json"
	"fmt"
)

// NormalizedTrajectory 是从原始响应中提取的高价值字段，按文档建议结构归一化。
type NormalizedTrajectory struct {
	CascadeID     string            `json:"cascadeId"`
	TrajectoryID  string            `json:"trajectoryId"`
	TrajectoryType string           `json:"trajectoryType,omitempty"`
	Status        any               `json:"status,omitempty"`
	NumTotalSteps any               `json:"numTotalSteps,omitempty"`
	WorkspaceURIs []string          `json:"workspaceUris,omitempty"`
	Steps         []NormalizedStep  `json:"steps,omitempty"`
}

// NormalizedStep 对应一个 step 的归一化视图。
type NormalizedStep struct {
	Index     int    `json:"index"`
	Type      string `json:"type,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	Text      string `json:"text,omitempty"`
}

// NormalizeResponse 从原始 JSON bytes 提取归一化字段。
// 宽松解析：字段缺失不报错，只提取能拿到的。
func NormalizeResponse(rawJSON []byte) (*NormalizedTrajectory, error) {
	// 先解成 map 做宽松遍历
	var raw map[string]any
	if err := json.Unmarshal(rawJSON, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal raw response: %w", err)
	}

	result := &NormalizedTrajectory{}

	// 顶层字段
	if v, ok := raw["status"]; ok {
		result.Status = v
	}
	if v, ok := raw["numTotalSteps"]; ok {
		result.NumTotalSteps = v
	}

	// trajectory 子对象
	trajRaw, ok := raw["trajectory"]
	if !ok {
		return result, nil
	}
	traj, ok := trajRaw.(map[string]any)
	if !ok {
		return result, nil
	}

	result.CascadeID = strField(traj, "cascadeId")
	result.TrajectoryID = strField(traj, "trajectoryId")
	result.TrajectoryType = strField(traj, "trajectoryType")

	// workspaceUris
	if wuRaw, ok := traj["workspaceUris"]; ok {
		if wuSlice, ok := wuRaw.([]any); ok {
			for _, u := range wuSlice {
				if s, ok := u.(string); ok {
					result.WorkspaceURIs = append(result.WorkspaceURIs, s)
				}
			}
		}
	}

	// steps[]
	if stepsRaw, ok := traj["steps"]; ok {
		if stepsSlice, ok := stepsRaw.([]any); ok {
			for i, stepRaw := range stepsSlice {
				step, ok := stepRaw.(map[string]any)
				if !ok {
					continue
				}
				ns := NormalizedStep{Index: i}
				ns.Type = strField(step, "stepType")
				ns.CreatedAt = strField(step, "createdAt")
				ns.Text = extractStepText(step)
				result.Steps = append(result.Steps, ns)
			}
		}
	}

	return result, nil
}

// strField 安全取 map 中的 string 值。
func strField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// extractStepText 尝试从 step 中提取人类可读文本，适配多种 step 结构。
func extractStepText(step map[string]any) string {
	// 常见路径：step.userInput.text / step.plannerResponse.text / step.notifyUser.message
	for _, key := range []string{"userInput", "plannerResponse", "notifyUser", "conversationHistory", "taskBoundary"} {
		if sub, ok := step[key].(map[string]any); ok {
			for _, textKey := range []string{"text", "message", "content", "summary"} {
				if s := strField(sub, textKey); s != "" {
					return s
				}
			}
		}
	}
	// fallback：直接取顶层的 text
	return strField(step, "text")
}
