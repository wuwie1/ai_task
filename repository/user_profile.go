package repository

import (
	"ai_web/test/entity"
	"ai_web/test/model"
)

type UserProfileRepository interface {
	Upsert(req *model.UpsertUserProfileCondition) error
	Get(userID, key string) (*entity.UserProfile, error)
	List(condition *model.GetUserProfileCondition) ([]*entity.UserProfile, error)
	Delete(userID, key string) error
}

