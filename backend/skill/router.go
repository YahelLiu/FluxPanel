package skill

import (
	"log"
	"strings"
	"sync"
)

// Router Skill 路由器
type Router struct {
	manager *Manager
	mu      sync.RWMutex
}

// NewRouter 创建 Skill Router
func NewRouter(manager *Manager) *Router {
	return &Router{
		manager: manager,
	}
}

// FindEligible 返回用户消息可用的 skills
// 过滤: enabled + user_enabled + trusted
func (r *Router) FindEligible(userID, message string) ([]*Skill, error) {
	allSkills, err := r.manager.List()
	if err != nil {
		return nil, err
	}

	// 获取用户特定设置
	userSettings, _ := r.manager.GetUserEnabledSkills(userID)
	userEnabledMap := make(map[string]bool)
	for _, s := range userSettings {
		userEnabledMap[s.ID] = true
	}

	var eligible []*Skill
	for _, skill := range allSkills {
		// 检查全局启用
		if !skill.Enabled {
			continue
		}

		// 检查用户是否特别禁用
		if userEnabled, ok := userEnabledMap[skill.ID]; ok {
			if !userEnabled {
				continue
			}
		}

		eligible = append(eligible, skill)
	}

	return eligible, nil
}

// SelectActive 选择要激活的 skills
// 使用关键词匹配
func (r *Router) SelectActive(eligible []*Skill, message string) []*Skill {
	messageLower := strings.ToLower(message)

	var active []*Skill
	for _, skill := range eligible {
		// 检查触发关键词
		for _, trigger := range skill.Triggers {
			if strings.Contains(messageLower, strings.ToLower(trigger)) {
				// 触发懒加载内容
				if !skill.contentLoaded && skill.Path != "" && r.manager != nil {
					r.manager.LoadSkillContent(skill)
				}
				active = append(active, skill)
				log.Printf("[skill] 匹配 skill %s (trigger: %s)", skill.ID, trigger)
				break
			}
		}
	}

	// 限制最多激活 3 个 skills
	if len(active) > 3 {
		active = active[:3]
	}

	return active
}

// Route 返回消息的 active skills
func (r *Router) Route(userID, message string) ([]*Skill, error) {
	eligible, err := r.FindEligible(userID, message)
	if err != nil {
		return nil, err
	}

	if len(eligible) == 0 {
		return nil, nil
	}

	return r.SelectActive(eligible, message), nil
}
