package agent

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"client-monitor/database"
	"client-monitor/models"
)

// ClaudeAgent Claude CLI 适配器
type ClaudeAgent struct {
	cliPath string
	cwd     string
}

// NewClaudeAgent 创建 Claude CLI 适配器
func NewClaudeAgent() *ClaudeAgent {
	cliPath := "claude"
	if path, err := exec.LookPath("claude"); err == nil {
		cliPath = path
	}

	return &ClaudeAgent{
		cliPath: cliPath,
	}
}

// Chat 实现 Agent 接口
func (a *ClaudeAgent) Chat(ctx context.Context, conversationID string, message string) (string, error) {
	// 构建带上下文的 prompt
	prompt := a.buildPrompt(conversationID, message)

	// 调用 claude CLI
	cmd := exec.CommandContext(ctx, a.cliPath, "--print", prompt)
	if a.cwd != "" {
		cmd.Dir = a.cwd
	}

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("claude CLI 错误: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("claude CLI 执行失败: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// buildPrompt 构建 prompt，包含对话历史
func (a *ClaudeAgent) buildPrompt(conversationID string, message string) string {
	var sb strings.Builder

	// 获取用户信息
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", conversationID).First(&user).Error; err == nil {
		// 获取记忆
		var memories []models.Memory
		database.DB.Where("user_id = ?", user.ID).Order("created_at desc").Limit(5).Find(&memories)
		if len(memories) > 0 {
			sb.WriteString("关于用户的重要信息：\n")
			for _, m := range memories {
				sb.WriteString("- " + m.Content + "\n")
			}
			sb.WriteString("\n")
		}

		// 获取最近对话历史
		var conversations []models.Conversation
		database.DB.Where("user_id = ?", user.ID).Order("created_at desc").Limit(5).Find(&conversations)
		if len(conversations) > 0 {
			sb.WriteString("最近的对话：\n")
			for i := len(conversations) - 1; i >= 0; i-- {
				c := conversations[i]
				if c.Role == "user" {
					sb.WriteString("用户: " + c.Content + "\n")
				} else {
					sb.WriteString("助手: " + c.Content + "\n")
				}
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("用户: " + message)

	return sb.String()
}

// ResetSession 清除对话历史
func (a *ClaudeAgent) ResetSession(ctx context.Context, conversationID string) (string, error) {
	var user models.AIUser
	if err := database.DB.Where("wecom_user_id = ?", conversationID).First(&user).Error; err != nil {
		return "", err
	}

	// 删除对话历史
	database.DB.Where("user_id = ?", user.ID).Delete(&models.Conversation{})
	return "会话已重置", nil
}

// Info 返回 Agent 信息
func (a *ClaudeAgent) Info() AgentInfo {
	status := "available"
	if _, err := exec.LookPath("claude"); err != nil {
		status = "unavailable"
	}

	return AgentInfo{
		Name:    "claude",
		Type:    "cli",
		Model:   "claude-sonnet-4",
		Command: a.cliPath,
		Status:  status,
	}
}

// SetCwd 设置工作目录
func (a *ClaudeAgent) SetCwd(cwd string) {
	a.cwd = cwd
}
