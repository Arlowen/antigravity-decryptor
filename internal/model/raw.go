package model

// RawCascadeTrajectoryResponse 是 GetCascadeTrajectory 的原始响应。
// 所有字段都用 omitempty + interface{}，做宽松反序列化，
// 避免未来 API 字段变动导致解析失败。
type RawCascadeTrajectoryResponse struct {
	Trajectory   map[string]any `json:"trajectory,omitempty"`
	Status       any            `json:"status,omitempty"`
	NumTotalSteps any           `json:"numTotalSteps,omitempty"`
}

// RawAllCascadeTrajectoriesResponse 是 GetAllCascadeTrajectories 的原始响应。
type RawAllCascadeTrajectoriesResponse struct {
	// TrajectorySummaries 是 cascadeId -> summary 的 map
	TrajectorySummaries map[string]any `json:"trajectorySummaries,omitempty"`
}
