package wecom

import (
	"client-monitor/database"
	"client-monitor/models"

	"gorm.io/gorm"
)

// PreferencePersister 用户偏好持久化器
type PreferencePersister struct{}

// NewPreferencePersister 创建持久化器
func NewPreferencePersister() *PreferencePersister {
	return &PreferencePersister{}
}

// Get 获取用户的适配器偏好
func (p *PreferencePersister) Get(userID string) (string, error) {
	var pref models.UserAIPreference
	err := database.DB.Where("wecom_user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return pref.AdapterName, nil
}

// Set 设置用户的适配器偏好
func (p *PreferencePersister) Set(userID, adapterName string) error {
	var pref models.UserAIPreference
	err := database.DB.Where("wecom_user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		pref = models.UserAIPreference{
			WecomUserID: userID,
			AdapterName: adapterName,
		}
		return database.DB.Create(&pref).Error
	}
	return database.DB.Model(&pref).Update("adapter_name", adapterName).Error
}
