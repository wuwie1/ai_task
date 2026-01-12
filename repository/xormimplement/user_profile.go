package xormimplement

import (
	"ai_task/entity"
	"ai_task/model"
	"ai_task/repository"
	"fmt"
	"time"

	"xorm.io/builder"
)

type UserProfileRepository struct {
	session *Session
}

func NewUserProfileRepository(session *Session) repository.UserProfileRepository {
	return &UserProfileRepository{session: session}
}

func (r *UserProfileRepository) Upsert(req *model.UpsertUserProfileCondition) error {
	if req == nil {
		return fmt.Errorf("upsert request cannot be nil")
	}
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Key == "" {
		return fmt.Errorf("key is required")
	}

	// 先尝试获取现有记录
	existing := &entity.UserProfile{}
	has, err := r.session.Table(entity.TableNameUserProfile).
		Where(builder.Eq{
			entity.UserProfileFieldUserID: req.UserID,
			entity.UserProfileFieldKey:    req.Key,
		}).
		Get(existing)

	if err != nil {
		return fmt.Errorf("failed to check existing user_profile: %w", err)
	}

	meta := "{}"
	if req.Meta != nil {
		meta = *req.Meta
	}

	if has {
		// 更新现有记录
		updateData := map[string]interface{}{
			entity.UserProfileFieldValue:      req.Value,
			entity.UserProfileFieldConfidence: req.Confidence,
			entity.UserProfileFieldUpdatedAt:  time.Now(),
		}
		if req.SourceMsgID != nil {
			updateData[entity.UserProfileFieldSourceMsgID] = *req.SourceMsgID
		}
		if req.Meta != nil {
			updateData[entity.UserProfileFieldMeta] = meta
		}

		_, err = r.session.Table(entity.TableNameUserProfile).
			Where(builder.Eq{
				entity.UserProfileFieldUserID: req.UserID,
				entity.UserProfileFieldKey:    req.Key,
			}).
			Update(updateData)
		if err != nil {
			return fmt.Errorf("failed to update user_profile: %w", err)
		}
	} else {
		// 插入新记录
		newProfile := &entity.UserProfile{
			UserID:      req.UserID,
			Key:         req.Key,
			Value:       req.Value,
			Confidence:  req.Confidence,
			UpdatedAt:   time.Now(),
			SourceMsgID: req.SourceMsgID,
			Meta:        meta,
		}
		_, err = r.session.Table(entity.TableNameUserProfile).Insert(newProfile)
		if err != nil {
			return fmt.Errorf("failed to insert user_profile: %w", err)
		}
	}

	return nil
}

func (r *UserProfileRepository) Get(userID, key string) (*entity.UserProfile, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	result := &entity.UserProfile{}
	ok, err := r.session.Table(entity.TableNameUserProfile).
		Where(builder.Eq{
			entity.UserProfileFieldUserID: userID,
			entity.UserProfileFieldKey:    key,
		}).
		Get(result)
	if err != nil {
		return nil, fmt.Errorf("failed to get user_profile: %w", err)
	}

	if !ok {
		return nil, nil
	}

	return result, nil
}

func (r *UserProfileRepository) List(condition *model.GetUserProfileCondition) ([]*entity.UserProfile, error) {
	if condition == nil {
		return nil, fmt.Errorf("get condition cannot be nil")
	}

	session := r.session.Table(entity.TableNameUserProfile)
	var conds []builder.Cond

	if condition.UserID != nil && *condition.UserID != "" {
		conds = append(conds, builder.Eq{entity.UserProfileFieldUserID: *condition.UserID})
	}
	if condition.Key != nil && *condition.Key != "" {
		conds = append(conds, builder.Eq{entity.UserProfileFieldKey: *condition.Key})
	}

	if len(conds) > 0 {
		session = session.Where(builder.And(conds...))
	}

	var results []*entity.UserProfile
	err := session.Find(&results)
	if err != nil {
		return nil, fmt.Errorf("failed to list user_profile: %w", err)
	}

	return results, nil
}

func (r *UserProfileRepository) Delete(userID, key string) error {
	if userID == "" {
		return fmt.Errorf("user_id is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}

	_, err := r.session.Table(entity.TableNameUserProfile).
		Where(builder.Eq{
			entity.UserProfileFieldUserID: userID,
			entity.UserProfileFieldKey:    key,
		}).
		Delete(&entity.UserProfile{})
	if err != nil {
		return fmt.Errorf("failed to delete user_profile: %w", err)
	}

	return nil
}
