package services

import (
	"fmt"
	"strings"
	"time"

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
func (h *MemoryHandler) Create(userID uint, content string, category string, importance int, source string) (*models.Memory, error) {
	// 设置默认值
	if category == "" {
		category = "fact"
	}
	if importance == 0 {
		importance = 5
	}
	if source == "" {
		source = "explicit"
	}

	memory := models.Memory{
		UserID:     userID,
		Content:    content,
		Category:   category,
		Importance: importance,
		Source:     source,
		Status:     "active",
	}
	if err := database.DB.Create(&memory).Error; err != nil {
		return nil, fmt.Errorf("保存记忆失败: %w", err)
	}
	return &memory, nil
}

// Search 搜索记忆（关键词匹配）
func (h *MemoryHandler) Search(userID uint, query string, limit int) ([]models.Memory, error) {
	if limit <= 0 {
		limit = 5
	}

	var memories []models.Memory
	query = strings.TrimSpace(query)

	// 空查询返回所有记忆
	if query == "" {
		err := database.DB.Where("user_id = ? AND status = ?", userID, "active").
			Order("importance desc, created_at desc").
			Limit(limit).
			Find(&memories).Error
		return memories, err
	}

	// 关键词匹配
	err := database.DB.Where("user_id = ? AND status = ? AND content ILIKE ?", userID, "active", "%"+query+"%").
		Order("importance desc, created_at desc").
		Limit(limit).
		Find(&memories).Error
	return memories, err
}

// Update 更新记忆
func (h *MemoryHandler) Update(userID uint, memoryID uint, content string) (*models.Memory, error) {
	var memory models.Memory
	if err := database.DB.Where("id = ? AND user_id = ?", memoryID, userID).First(&memory).Error; err != nil {
		return nil, fmt.Errorf("记忆不存在: %w", err)
	}

	oldContent := memory.Content
	memory.Content = content
	memory.UpdatedAt = time.Now()

	if err := database.DB.Save(&memory).Error; err != nil {
		return nil, fmt.Errorf("更新记忆失败: %w", err)
	}

	// 返回时记录旧内容用于日志
	_ = oldContent
	return &memory, nil
}

// Delete 删除记忆（软删除，设置 status 为 deleted）
func (h *MemoryHandler) Delete(userID uint, memoryID uint) error {
	result := database.DB.Model(&models.Memory{}).
		Where("id = ? AND user_id = ?", memoryID, userID).
		Update("status", "deleted")
	if result.Error != nil {
		return fmt.Errorf("删除记忆失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("记忆不存在")
	}
	return nil
}

// FindByContent 精确查找记忆（用于去重）
func (h *MemoryHandler) FindByContent(userID uint, content string) (*models.Memory, error) {
	var memory models.Memory
	err := database.DB.Where("user_id = ? AND content = ? AND status = ?", userID, content, "active").First(&memory).Error
	if err != nil {
		return nil, nil // 未找到返回 nil，不是错误
	}
	return &memory, nil
}

// FindSimilarByCategory 在同分类中查找相似记忆
func (h *MemoryHandler) FindSimilarByCategory(userID uint, category string, keyword string) (*models.Memory, error) {
	var memory models.Memory
	err := database.DB.Where("user_id = ? AND category = ? AND status = ? AND content ILIKE ?", userID, category, "active", "%"+keyword+"%").
		First(&memory).Error
	if err != nil {
		return nil, nil
	}
	return &memory, nil
}

// UpdateLastUsed 更新最后使用时间
func (h *MemoryHandler) UpdateLastUsed(memoryID uint) error {
	now := time.Now()
	return database.DB.Model(&models.Memory{}).Where("id = ?", memoryID).Update("last_used_at", now).Error
}

// List 查看记忆列表（兼容旧接口）
func (h *MemoryHandler) List(userID uint) (string, error) {
	memories, err := h.Search(userID, "", 20)
	if err != nil {
		return "", fmt.Errorf("查询记忆失败: %w", err)
	}

	if len(memories) == 0 {
		return "我还没有记住任何东西。", nil
	}

	var sb strings.Builder
	sb.WriteString("我记住了这些：\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, m.Category, m.Content))
	}
	return sb.String(), nil
}

// ListByCategory 按分类查看记忆
func (h *MemoryHandler) ListByCategory(userID uint, category string) ([]models.Memory, error) {
	var memories []models.Memory
	query := database.DB.Where("user_id = ? AND status = ?", userID, "active")

	if category != "" && category != "all" {
		query = query.Where("category = ?", category)
	}

	err := query.Order("importance desc, created_at desc").Find(&memories).Error
	return memories, err
}

// DeleteByKeyword 按关键词删除记忆（兼容旧接口）
func (h *MemoryHandler) DeleteByKeyword(userID uint, keyword string) (string, error) {
	var memories []models.Memory
	if err := database.DB.Where("user_id = ? AND status = ? AND content ILIKE ?", userID, "active", "%"+keyword+"%").Find(&memories).Error; err != nil {
		return "", fmt.Errorf("查询记忆失败: %w", err)
	}

	if len(memories) == 0 {
		return "没有找到匹配的记忆。", nil
	}
	if len(memories) > 1 {
		return "找到多个匹配的记忆，请更具体一些。", nil
	}

	if err := h.Delete(userID, memories[0].ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("已忘记：%s", memories[0].Content), nil
}

// CreateLegacy 创建记忆（兼容旧接口）
func (h *MemoryHandler) CreateLegacy(userID uint, content string) (string, error) {
	_, err := h.Create(userID, content, "fact", 5, "explicit")
	if err != nil {
		return "", err
	}
	return "好的，我记住了。", nil
}
