package entity

import "time"

const (
	TableNameUserProfile = "user_profile"

	UserProfileFieldID          = "id"
	UserProfileFieldUserID      = "user_id"
	UserProfileFieldKey         = "key"
	UserProfileFieldValue       = "value"
	UserProfileFieldConfidence  = "confidence"
	UserProfileFieldUpdatedAt   = "updated_at"
	UserProfileFieldSourceMsgID = "source_msg_id"
	UserProfileFieldMeta        = "meta"
)

type UserProfile struct {
	ID          int64     `xorm:"pk autoincr id" json:"id"`
	UserID      string    `xorm:"user_id" json:"user_id"`
	Key         string    `xorm:"key" json:"key"`
	Value       string    `xorm:"value" json:"value"`
	Confidence  float32   `xorm:"confidence" json:"confidence"`
	UpdatedAt   time.Time `xorm:"updated_at" json:"updated_at"`
	SourceMsgID *int64    `xorm:"source_msg_id" json:"source_msg_id"`
	Meta        string    `xorm:"meta" json:"meta"` // JSONB 类型，存储为 JSON 字符串
}

func (e *UserProfile) TableName() string {
	return TableNameUserProfile
}
