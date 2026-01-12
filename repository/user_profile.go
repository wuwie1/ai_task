package repository

import (
	"ai_task/entity"
	"ai_task/model"
)

type UserProfileRepository interface {
	Upsert(req *model.UpsertUserProfileCondition) error
	Get(userID, key string) (*entity.UserProfile, error)
	List(condition *model.GetUserProfileCondition) ([]*entity.UserProfile, error)
	Delete(userID, key string) error
}
