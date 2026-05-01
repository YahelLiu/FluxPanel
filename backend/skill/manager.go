package skill

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"client-monitor/database"
	"client-monitor/models"

	"gorm.io/gorm"
)

// Manager 管理 skill 生命周期
type Manager struct {
	parser    *Parser
	loader    *Loader
	cache     sync.Map // skillID -> *Skill
	skillsDir string
}

// 全局 Manager 实例
var globalManager *Manager

// content 加载锁
var skillMu sync.Mutex

// GetManager 获取全局 Manager
func GetManager() *Manager {
	return globalManager
}

// NewManager 创建 Skill Manager
func NewManager(skillsDir string) *Manager {
	m := &Manager{
		parser:    NewParser(),
		loader:    NewLoader(),
		skillsDir: skillsDir,
	}
	// 启动时加载所有 skills
	m.loadAllFromDB()

	// 设置全局实例
	globalManager = m

	return m
}

// loadAllFromDB 从数据库加载所有 skills
func (m *Manager) loadAllFromDB() {
	var dbSkills []models.Skill
	if err := database.DB.Where("enabled = ?", true).Find(&dbSkills).Error; err != nil {
		log.Printf("[skill] 加载数据库 skills 失败: %v", err)
		return
	}

	for _, dbSkill := range dbSkills {
		skill := m.convertToSkill(dbSkill)

		// 如果没有设置 AllowedTools，根据名称自动设置
		if len(skill.AllowedTools) == 0 {
			skill.AllowedTools = m.getDefaultTools(skill)
			// 更新数据库
			toolsJSON, _ := json.Marshal(skill.AllowedTools)
			database.DB.Model(&models.Skill{}).
				Where("skill_id = ?", skill.ID).
				Update("allowed_tools", string(toolsJSON))
			log.Printf("[skill] 自动设置 %s 允许工具: %v", skill.ID, skill.AllowedTools)
		}

		m.cache.Store(skill.ID, skill)
	}

	log.Printf("[skill] 已加载 %d 个 skills", len(dbSkills))
}

// convertToSkill 将数据库模型转换为 Skill
func (m *Manager) convertToSkill(dbSkill models.Skill) *Skill {
	var triggers []string
	if dbSkill.Triggers != "" {
		json.Unmarshal([]byte(dbSkill.Triggers), &triggers)
	}

	var allowedTools []string
	if dbSkill.AllowedTools != "" {
		json.Unmarshal([]byte(dbSkill.AllowedTools), &allowedTools)
	}

	return &Skill{
		ID:           dbSkill.SkillID,
		Name:         dbSkill.Name,
		Description:  dbSkill.Description,
		Type:         SkillType(dbSkill.Type),
		Source:       dbSkill.Source,
		Path:         dbSkill.Path,
		Version:      dbSkill.Version,
		Author:       dbSkill.Author,
		Trusted:      dbSkill.Trusted,
		AllowedTools: allowedTools,
		Enabled:      dbSkill.Enabled,
		ContentHash:  dbSkill.ContentHash,
		Triggers:     triggers,
		contentLoaded: false, // 懒加载
		CreatedAt:    dbSkill.CreatedAt,
		UpdatedAt:    dbSkill.UpdatedAt,
	}
}

// Import 从路径导入 skill
func (m *Manager) Import(path string) (*Skill, error) {
	// 查找 SKILL.md
	skillMdPath := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(skillMdPath); os.IsNotExist(err) {
		return nil, ErrSkillMdNotFound
	}

	// 解析 SKILL.md
	skill, err := m.parser.ParseFile(skillMdPath)
	if err != nil {
		return nil, err
	}

	// 加载 references 和 templates
	m.parser.LoadReferences(skill)
	m.parser.LoadTemplates(skill)

	// 根据名称设置默认允许的工具
	if len(skill.AllowedTools) == 0 {
		skill.AllowedTools = m.getDefaultTools(skill)
	}

	// 保存到数据库
	dbSkill := m.convertToDBModel(skill)
	dbSkill.Source = "uploaded"

	var existing models.Skill
	result := database.DB.Where("skill_id = ?", skill.ID).First(&existing)
	if result.Error == gorm.ErrRecordNotFound {
		// 新建
		if err := database.DB.Create(dbSkill).Error; err != nil {
			return nil, err
		}
	} else {
		// 更新
		dbSkill.ID = existing.ID
		if err := database.DB.Save(dbSkill).Error; err != nil {
			return nil, err
		}
	}

	// 更新缓存
	m.cache.Store(skill.ID, skill)

	log.Printf("[skill] 导入 skill: %s (tools: %v)", skill.ID, skill.AllowedTools)
	return skill, nil
}

// getDefaultTools 根据技能名称返回默认允许的工具
func (m *Manager) getDefaultTools(skill *Skill) []string {
	switch skill.ID {
	case "reminder":
		return []string{"reminder_create", "reminder_list", "reminder_cancel"}
	case "memory":
		return []string{"memory_save", "memory_list", "memory_delete"}
	case "translator":
		return []string{"translator"}
	case "weather":
		return []string{"weather_get", "weather_send"}
	default:
		// 检查类型
		if skill.Type == SkillTypeTool {
			// 尝试匹配名称
			name := skill.ID
			if containsAny(name, "reminder", "提醒") {
				return []string{"reminder_create", "reminder_list", "reminder_cancel"}
			}
			if containsAny(name, "memory", "记忆") {
				return []string{"memory_save", "memory_list", "memory_delete"}
			}
			if containsAny(name, "translat", "翻译") {
				return []string{"translator"}
			}
			if containsAny(name, "weather", "天气") {
				return []string{"weather_get", "weather_send"}
			}
		}
		return []string{}
	}
}

// containsAny 检查字符串是否包含任意一个子串
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// convertToDBModel 将 Skill 转换为数据库模型
func (m *Manager) convertToDBModel(skill *Skill) *models.Skill {
	triggersJSON, _ := json.Marshal(skill.Triggers)
	toolsJSON, _ := json.Marshal(skill.AllowedTools)

	return &models.Skill{
		SkillID:      skill.ID,
		Name:         skill.Name,
		Description:  skill.Description,
		Type:         string(skill.Type),
		Source:       skill.Source,
		Path:         skill.Path,
		Version:      skill.Version,
		Author:       skill.Author,
		Trusted:      skill.Trusted,
		AllowedTools: string(toolsJSON),
		Enabled:      skill.Enabled,
		ContentHash:  skill.ContentHash,
		Triggers:     string(triggersJSON),
	}
}

// Remove 删除 skill
func (m *Manager) Remove(skillID string) error {
	// 从数据库删除
	if err := database.DB.Where("skill_id = ?", skillID).Delete(&models.Skill{}).Error; err != nil {
		return err
	}

	// 从缓存删除
	m.cache.Delete(skillID)

	log.Printf("[skill] 删除 skill: %s", skillID)
	return nil
}

// Get 获取 skill (带懒加载内容)
func (m *Manager) Get(skillID string) (*Skill, error) {
	// 先查缓存
	if cached, ok := m.cache.Load(skillID); ok {
		skill := cached.(*Skill)
		// 懒加载内容
		if !skill.contentLoaded && skill.Path != "" {
			m.loadSkillContent(skill)
		}
		return skill, nil
	}

	// 查数据库
	var dbSkill models.Skill
	if err := database.DB.Where("skill_id = ?", skillID).First(&dbSkill).Error; err != nil {
		return nil, ErrSkillNotFound
	}

	skill := m.convertToSkill(dbSkill)
	m.cache.Store(skillID, skill)

	return skill, nil
}

// LoadSkillContent 加载 skill 内容（懒加载）
func (m *Manager) LoadSkillContent(skill *Skill) {
	skillMu.Lock()
	defer skillMu.Unlock()

	if skill.contentLoaded {
		return
	}

	skillMdPath := filepath.Join(skill.Path, "SKILL.md")
	if fullSkill, err := m.parser.ParseFile(skillMdPath); err == nil {
		skill.content = fullSkill.content
		skill.contentLoaded = true
		m.parser.LoadReferences(skill)
		m.parser.LoadTemplates(skill)
		log.Printf("[skill] 懒加载 skill 内容: %s", skill.ID)
	}
}

// loadSkillContent 内部方法，不加锁版本
func (m *Manager) loadSkillContent(skill *Skill) {
	skillMdPath := filepath.Join(skill.Path, "SKILL.md")
	if fullSkill, err := m.parser.ParseFile(skillMdPath); err == nil {
		skill.content = fullSkill.content
		skill.contentLoaded = true
		m.parser.LoadReferences(skill)
		m.parser.LoadTemplates(skill)
	}
}

// List 返回所有 skills
func (m *Manager) List() ([]*Skill, error) {
	var skills []*Skill

	// 从缓存获取
	m.cache.Range(func(key, value interface{}) bool {
		skills = append(skills, value.(*Skill))
		return true
	})

	return skills, nil
}

// SetEnabled 全局启用/禁用 skill
func (m *Manager) SetEnabled(skillID string, enabled bool) error {
	// 更新数据库
	if err := database.DB.Model(&models.Skill{}).
		Where("skill_id = ?", skillID).
		Update("enabled", enabled).Error; err != nil {
		return err
	}

	// 更新缓存
	if cached, ok := m.cache.Load(skillID); ok {
		skill := cached.(*Skill)
		skill.Enabled = enabled
		if !enabled {
			m.cache.Delete(skillID)
		}
	}

	log.Printf("[skill] 设置 skill %s enabled=%v", skillID, enabled)
	return nil
}

// SetUserEnabled 为用户启用/禁用 skill
func (m *Manager) SetUserEnabled(userID, skillID string, enabled bool) error {
	// 查找 skill
	var dbSkill models.Skill
	if err := database.DB.Where("skill_id = ?", skillID).First(&dbSkill).Error; err != nil {
		return ErrSkillNotFound
	}

	// 查找或创建用户设置
	var userSetting models.UserSkillSetting
	result := database.DB.Where("user_id = ? AND skill_id = ?", userID, dbSkill.ID).First(&userSetting)
	if result.Error == gorm.ErrRecordNotFound {
		userSetting = models.UserSkillSetting{
			UserID:  userID,
			SkillID: dbSkill.ID,
			Enabled: enabled,
		}
		return database.DB.Create(&userSetting).Error
	}

	return database.DB.Model(&userSetting).Update("enabled", enabled).Error
}

// GetUserEnabledSkills 获取用户启用的 skills
func (m *Manager) GetUserEnabledSkills(userID string) ([]*Skill, error) {
	var dbSkills []models.Skill
	database.DB.Table("skills").
		Select("skills.*").
		Joins("LEFT JOIN user_skill_settings ON skills.id = user_skill_settings.skill_id").
		Where("user_skill_settings.user_id = ? AND user_skill_settings.enabled = ? AND skills.enabled = ?", userID, true, true).
		Find(&dbSkills)

	var skills []*Skill
	for _, dbSkill := range dbSkills {
		skill := m.convertToSkill(dbSkill)
		skills = append(skills, skill)
	}

	return skills, nil
}

// ScanDirectory 扫描目录并导入所有 skills
func (m *Manager) ScanDirectory(dir string) ([]*Skill, error) {
	var skills []*Skill

	log.Printf("[skill] 扫描目录: %s", dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("[skill] 读取目录失败: %v", err)
		return nil, err
	}

	log.Printf("[skill] 找到 %d 个条目", len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())
		skillMdPath := filepath.Join(skillPath, "SKILL.md")

		if _, err := os.Stat(skillMdPath); err == nil {
			skill, err := m.Import(skillPath)
			if err != nil {
				log.Printf("[skill] 导入 %s 失败: %v", skillPath, err)
				continue
			}
			skills = append(skills, skill)
		} else {
			log.Printf("[skill] 跳过 %s: 无 SKILL.md", skillPath)
		}
	}

	log.Printf("[skill] 扫描完成，导入 %d 个 skills", len(skills))
	return skills, nil
}

// SetAllowedTools 设置 skill 允许的工具
func (m *Manager) SetAllowedTools(skillID string, tools []string) error {
	toolsJSON, _ := json.Marshal(tools)

	if err := database.DB.Model(&models.Skill{}).
		Where("skill_id = ?", skillID).
		Update("allowed_tools", string(toolsJSON)).Error; err != nil {
		return err
	}

	// 更新缓存
	if cached, ok := m.cache.Load(skillID); ok {
		skill := cached.(*Skill)
		skill.AllowedTools = tools
	}

	log.Printf("[skill] 设置 skill %s 允许工具: %v", skillID, tools)
	return nil
}

// 错误定义
var (
	ErrSkillMdNotFound = &SkillError{Message: "SKILL.md 文件不存在"}
	ErrSkillNotFound   = &SkillError{Message: "skill 不存在"}
	ErrSkillDisabled   = &SkillError{Message: "skill 已禁用"}
	ErrToolNotAllowed  = &SkillError{Message: "工具不被允许"}
)

// SkillError skill 错误
type SkillError struct {
	Message string
}

func (e *SkillError) Error() string {
	return e.Message
}
