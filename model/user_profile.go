package model

// GetUserProfileCondition 查询条件
type GetUserProfileCondition struct {
	UserID *string `json:"user_id"`
	Key    *string `json:"key"`
}

// UpdateUserProfileCondition 更新条件
type UpdateUserProfileCondition struct {
	Value       *string  `json:"value"`
	Confidence  *float32 `json:"confidence"`
	SourceMsgID *int64   `json:"source_msg_id"`
	Meta        *string  `json:"meta"`
}

// UpsertUserProfileCondition 插入或更新条件
type UpsertUserProfileCondition struct {
	UserID      string   `json:"user_id"`
	Key         string   `json:"key"`
	Value       string   `json:"value"`
	Confidence  float32  `json:"confidence"`
	SourceMsgID *int64   `json:"source_msg_id"`
	Meta        *string  `json:"meta"`
}

