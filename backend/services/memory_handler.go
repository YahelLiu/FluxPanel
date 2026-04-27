package services

import (
	"fmt"
	"strings"

	"client-monitor/database"
	"client-monitor/models"
)

// MemoryHandler 记忆处理器
type MemoryHandler struct{}

// NewMemoryHandler 创建记忆处理器
func NewMemoryHandler() *MemoryHandler {
	return &MemoryHandler{}
}

// Create 创建记忆
func (h *MemoryHandler) Create(userID uint, content string) (string, error) {
	memory := models.Memory{
		UserID:  userID,
		Content: content,
	}
	if err := database.DB.Create(&memory).Error; err != nil {
		return "", fmt.Errorf("保存记忆失败: %w", err)
	}
	return "好的，我记住了。", nil
}

// List 查看记忆列表
func (h *MemoryHandler) List(userID uint) (string, error) {
	var memories []models.Memory
	if err := database.DB.Where("user_id = ?", userID).Order("created_at desc").Limit(20).Find(&memories).Error; err != nil {
		return "", fmt.Errorf("查询记忆失败: %w", err)
	}

	if len(memories) == 0 {
		return "我还没有记住任何东西。", nil
	}

	var sb strings.Builder
	sb.WriteString("我记住了这些：\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, m.Content))
	}
	return sb.String(), nil
}

// Delete 删除记忆
func (h *MemoryHandler) Delete(userID uint, keyword string) (string, error) {
	var memories []models.Memory
	if err := database.DB.Where("user_id = ? AND content ILIKE ?", userID, "%"+keyword+"%").Find(&memories).Error; err != nil {
		return "", fmt.Errorf("查询记忆失败: %w", err)
	}

	if len(memories) == 0 {
		return "没有找到匹配的记忆。", nil
	}
	if len(memories) > 1 {
		return "找到多个匹配的记忆，请更具体一些。", nil
	}

	if err := database.DB.Delete(&memories[0]).Error; err != nil {
		return "", fmt.Errorf("删除记忆失败: %w", err)
	}
	return fmt.Sprintf("已忘记：%s", memories[0].Content), nil
}
