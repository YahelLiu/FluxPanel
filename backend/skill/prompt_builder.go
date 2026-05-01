package skill

import (
	"fmt"
	"strings"
)

// PromptBuilder 构建 skill prompts
type PromptBuilder struct{}

// NewPromptBuilder 创建 PromptBuilder
func NewPromptBuilder() *PromptBuilder {
	return &PromptBuilder{}
}

// BuildCatalog 构建 skill catalog prompt (只有元数据)
func (b *PromptBuilder) BuildCatalog(skills []*SkillMetadata) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("你可以使用以下技能:\n\n")

	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("- **%s**: %s\n", skill.Name, skill.Description))
		if len(skill.Triggers) > 0 {
			sb.WriteString(fmt.Sprintf("  触发词: %s\n", strings.Join(skill.Triggers, ", ")))
		}
	}

	sb.WriteString("\n当用户请求与技能描述匹配时，系统会自动激活相应技能。\n")

	return sb.String()
}

// BuildActivePrompt 构建 active skills 的 prompt
func (b *PromptBuilder) BuildActivePrompt(skills []*Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("以下技能已激活，请按照技能说明处理用户请求:\n\n")

	for _, skill := range skills {
		sb.WriteString(fmt.Sprintf("## 技能: %s\n", skill.Name))
		sb.WriteString(fmt.Sprintf("描述: %s\n\n", skill.Description))

		// 加载完整内容
		if skill.contentLoaded && skill.content != nil {
			sb.WriteString(skill.content.SystemPrompt)
			sb.WriteString("\n\n")

			// 添加参考文档
			if len(skill.content.References) > 0 {
				sb.WriteString("### 参考资料\n")
				for name, content := range skill.content.References {
					sb.WriteString(fmt.Sprintf("**%s**:\n%s\n\n", name, content))
				}
			}

			// 添加模板
			if len(skill.content.Templates) > 0 {
				sb.WriteString("### 模板\n")
				for name, content := range skill.content.Templates {
					sb.WriteString(fmt.Sprintf("**%s**:\n%s\n\n", name, content))
				}
			}
		}

		// 显示允许的工具
		if len(skill.AllowedTools) > 0 {
			sb.WriteString(fmt.Sprintf("可用工具: %s\n\n", strings.Join(skill.AllowedTools, ", ")))
		}
	}

	return sb.String()
}

// BuildSystemPrompt 构建 skill 上下文的系统 prompt
func (b *PromptBuilder) BuildSystemPrompt(basePrompt string, skills []*Skill) string {
	activePrompt := b.BuildActivePrompt(skills)
	if activePrompt == "" {
		return basePrompt
	}

	if basePrompt == "" {
		return activePrompt
	}

	return basePrompt + "\n\n" + activePrompt
}

// BuildToolCatalog 构建可用工具的描述
func (b *PromptBuilder) BuildToolCatalog(skills []*Skill, registry *ToolRegistry) string {
	var tools []string
	toolSet := make(map[string]bool)

	for _, skill := range skills {
		for _, toolName := range skill.AllowedTools {
			if !toolSet[toolName] {
				toolSet[toolName] = true
				tools = append(tools, toolName)
			}
		}
	}

	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("可用工具:\n")

	for _, toolName := range tools {
		tool := registry.Get(toolName)
		if tool != nil {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description))
			if len(tool.Parameters) > 0 {
				sb.WriteString("  参数:\n")
				for param, spec := range tool.Parameters {
					required := ""
					if spec.Required {
						required = " (必需)"
					}
					sb.WriteString(fmt.Sprintf("  - %s%s: %s\n", param, required, spec.Description))
				}
			}
		}
	}

	return sb.String()
}
