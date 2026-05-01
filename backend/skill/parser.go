package skill

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser 解析 SKILL.md 文件
type Parser struct{}

// NewParser 创建解析器
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile 解析 SKILL.md 文件，返回 Skill 结构
func (p *Parser) ParseFile(path string) (*Skill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return p.ParseContent(content, path)
}

// ParseContent 解析 SKILL.md 内容
func (p *Parser) ParseContent(content []byte, path string) (*Skill, error) {
	// 解析 YAML frontmatter
	config, body, err := p.parseFrontmatter(content)
	if err != nil {
		return nil, err
	}

	// 验证必要字段
	if config.Name == "" {
		return nil, fmt.Errorf("skill 缺少 name 字段")
	}
	if config.Description == "" {
		return nil, fmt.Errorf("skill 缺少 description 字段")
	}

	// 计算内容哈希
	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])[:16]

	// 确定 skill 类型
	skillType := SkillTypeInstruction
	if config.Type != "" {
		switch config.Type {
		case "tool":
			skillType = SkillTypeTool
		case "resource":
			skillType = SkillTypeResource
		case "instruction":
			skillType = SkillTypeInstruction
		}
	}

	skill := &Skill{
		ID:           config.Name,
		Name:         config.Name,
		Description:  config.Description,
		Type:         skillType,
		Version:      config.Version,
		Author:       config.Author,
		Path:         filepath.Dir(path),
		Triggers:     config.Triggers,
		ContentHash:  contentHash,
		Enabled:      true,
		Trusted:      false,
		AllowedTools: []string{},
		content: &SkillContent{
			SystemPrompt: strings.TrimSpace(body),
			References:   make(map[string]string),
			Templates:    make(map[string]string),
		},
		contentLoaded: true,
	}

	return skill, nil
}

// parseFrontmatter 解析 YAML frontmatter
// 格式:
// ---
// name: example
// description: example skill
// ---
// # Skill content
func (p *Parser) parseFrontmatter(content []byte) (*SkillConfig, string, error) {
	// 检查是否以 --- 开头
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return nil, "", fmt.Errorf("SKILL.md 必须以 YAML frontmatter 开头 (---)")
	}

	// 查找结束的 ---
	endIndex := bytes.Index(content[4:], []byte("\n---"))
	if endIndex == -1 {
		return nil, "", fmt.Errorf("未找到 YAML frontmatter 结束标记 (---)")
	}

	frontmatter := content[4 : endIndex+4]
	body := content[endIndex+8:] // 跳过 \n---

	var config SkillConfig
	if err := yaml.Unmarshal(frontmatter, &config); err != nil {
		return nil, "", fmt.Errorf("解析 YAML frontmatter 失败: %w", err)
	}

	return &config, string(body), nil
}

// ParseMetadataOnly 只解析元数据，不加载完整内容
func (p *Parser) ParseMetadataOnly(path string) (*SkillMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// 检查第一行是否为 ---
	if !scanner.Scan() || scanner.Text() != "---" {
		return nil, fmt.Errorf("SKILL.md 必须以 YAML frontmatter 开头 (---)")
	}

	// 读取 frontmatter
	var frontmatterLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		frontmatterLines = append(frontmatterLines, line)
	}

	frontmatter := strings.Join(frontmatterLines, "\n")
	var config SkillConfig
	if err := yaml.Unmarshal([]byte(frontmatter), &config); err != nil {
		return nil, fmt.Errorf("解析 YAML frontmatter 失败: %w", err)
	}

	if config.Name == "" {
		return nil, fmt.Errorf("skill 缺少 name 字段")
	}

	// 确定类型
	skillType := SkillTypeInstruction
	switch config.Type {
	case "tool":
		skillType = SkillTypeTool
	case "resource":
		skillType = SkillTypeResource
	}

	return &SkillMetadata{
		ID:          config.Name,
		Name:        config.Name,
		Description: config.Description,
		Type:        skillType,
		Triggers:    config.Triggers,
	}, nil
}

// LoadReferences 加载 references 目录下的文件
func (p *Parser) LoadReferences(skill *Skill) error {
	if skill.content == nil {
		skill.content = &SkillContent{
			References: make(map[string]string),
			Templates:  make(map[string]string),
		}
	}

	refsDir := filepath.Join(skill.Path, "references")
	if _, err := os.Stat(refsDir); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}

	return filepath.WalkDir(refsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") && !strings.HasSuffix(path, ".txt") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(refsDir, path)
		skill.content.References[relPath] = string(content)
		return nil
	})
}

// LoadTemplates 加载 templates 目录下的文件
func (p *Parser) LoadTemplates(skill *Skill) error {
	if skill.content == nil {
		skill.content = &SkillContent{
			References: make(map[string]string),
			Templates:  make(map[string]string),
		}
	}

	tplsDir := filepath.Join(skill.Path, "templates")
	if _, err := os.Stat(tplsDir); os.IsNotExist(err) {
		return nil // 目录不存在，跳过
	}

	return filepath.WalkDir(tplsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(tplsDir, path)
		skill.content.Templates[relPath] = string(content)
		return nil
	})
}
