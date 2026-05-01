package skill

import (
	"os"
	"path/filepath"
)

// Loader 文件系统加载器
type Loader struct{}

// NewLoader 创建加载器
func NewLoader() *Loader {
	return &Loader{}
}

// LoadSkill 从目录加载 skill
func (l *Loader) LoadSkill(dir string) (*Skill, error) {
	skillMdPath := filepath.Join(dir, "SKILL.md")
	parser := NewParser()
	return parser.ParseFile(skillMdPath)
}

// Exists 检查 skill 目录是否存在
func (l *Loader) Exists(path string) bool {
	skillMdPath := filepath.Join(path, "SKILL.md")
	_, err := os.Stat(skillMdPath)
	return err == nil
}

// ListSkills 列出目录下的所有 skill
func (l *Loader) ListSkills(dir string) ([]string, error) {
	var skillPaths []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(dir, entry.Name())
		if l.Exists(skillPath) {
			skillPaths = append(skillPaths, skillPath)
		}
	}

	return skillPaths, nil
}

// ReadReference 读取 references 文件
func (l *Loader) ReadReference(skillPath, filename string) (string, error) {
	refPath := filepath.Join(skillPath, "references", filename)
	content, err := os.ReadFile(refPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ReadTemplate 读取 templates 文件
func (l *Loader) ReadTemplate(skillPath, filename string) (string, error) {
	tplPath := filepath.Join(skillPath, "templates", filename)
	content, err := os.ReadFile(tplPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
