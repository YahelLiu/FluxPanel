package handlers

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"client-monitor/database"
	"client-monitor/models"
	"client-monitor/skill"

	"github.com/gin-gonic/gin"
)

// ListSkills GET /api/skills - 列出所有 skills
func ListSkills(c *gin.Context) {
	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	skills, err := manager.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换为响应格式
	var result []gin.H
	for _, s := range skills {
		result = append(result, gin.H{
			"id":           s.ID,
			"name":         s.Name,
			"description":  s.Description,
			"type":         string(s.Type),
			"source":       s.Source,
			"enabled":      s.Enabled,
			"trusted":      s.Trusted,
			"allowed_tools": s.AllowedTools,
			"triggers":     s.Triggers,
			"created_at":   s.CreatedAt,
			"updated_at":   s.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"skills": result})
}

// GetSkill GET /api/skills/:id - 获取 skill 详情
func GetSkill(c *gin.Context) {
	skillID := c.Param("id")

	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	s, err := manager.Get(skillID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           s.ID,
		"name":         s.Name,
		"description":  s.Description,
		"type":         string(s.Type),
		"source":       s.Source,
		"enabled":      s.Enabled,
		"trusted":      s.Trusted,
		"allowed_tools": s.AllowedTools,
		"triggers":     s.Triggers,
		"created_at":   s.CreatedAt,
		"updated_at":   s.UpdatedAt,
	})
}

// UploadSkill POST /api/skills/upload - 上传 zip 包安装 skill
func UploadSkill(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	// 检查文件类型
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only zip files are supported"})
		return
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "skill-upload-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	defer os.RemoveAll(tempDir)

	// 保存上传的 zip 文件
	zipPath := filepath.Join(tempDir, header.Filename)
	dst, err := os.Create(zipPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create zip file"})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save zip file"})
		return
	}
	dst.Close()

	// 解压
	extractDir := filepath.Join(tempDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to unzip: %v", err)})
		return
	}

	// 查找 SKILL.md
	skillDir, err := findSkillDir(extractDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 安装 skill
	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	s, err := manager.Import(skillDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to import skill: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s installed successfully", s.Name),
		"skill": gin.H{
			"id":          s.ID,
			"name":        s.Name,
			"description": s.Description,
			"type":        string(s.Type),
		},
	})
}

// InstallSkillFromURL POST /api/skills/install - 从 URL 安装 skill
func InstallSkillFromURL(c *gin.Context) {
	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "skill-install-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temp dir"})
		return
	}
	defer os.RemoveAll(tempDir)

	// 下载 zip 文件
	zipPath := filepath.Join(tempDir, "download.zip")
	if err := downloadFile(req.URL, zipPath); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to download: %v", err)})
		return
	}

	// 解压
	extractDir := filepath.Join(tempDir, "extracted")
	if err := unzip(zipPath, extractDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to unzip: %v", err)})
		return
	}

	// 查找 SKILL.md
	skillDir, err := findSkillDir(extractDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 安装 skill
	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	s, err := manager.Import(skillDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("failed to import skill: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s installed successfully", s.Name),
		"skill": gin.H{
			"id":          s.ID,
			"name":        s.Name,
			"description": s.Description,
			"type":        string(s.Type),
		},
	})
}

// EnableSkill PUT /api/skills/:id/enable - 启用/禁用 skill
func EnableSkill(c *gin.Context) {
	skillID := c.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	if err := manager.SetEnabled(skillID, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action := "enabled"
	if !req.Enabled {
		action = "disabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s %s", skillID, action),
	})
}

// SetSkillTools PUT /api/skills/:id/tools - 设置 skill 允许的工具
func SetSkillTools(c *gin.Context) {
	skillID := c.Param("id")

	var req struct {
		Tools []string `json:"tools"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	if err := manager.SetAllowedTools(skillID, req.Tools); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s tools updated", skillID),
	})
}

// DeleteSkill DELETE /api/skills/:id - 删除 skill
func DeleteSkill(c *gin.Context) {
	skillID := c.Param("id")

	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	if err := manager.Remove(skillID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s deleted", skillID),
	})
}

// GetUserSkillSettings GET /api/skills/user/:userId - 获取用户的 skill 设置
func GetUserSkillSettings(c *gin.Context) {
	userID := c.Param("userId")

	var settings []models.UserSkillSetting
	database.DB.Preload("Skill").Where("user_id = ?", userID).Find(&settings)

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// SetUserSkillEnabled PUT /api/skills/user/:userId/:skillId - 设置用户 skill 启用状态
func SetUserSkillEnabled(c *gin.Context) {
	userID := c.Param("userId")
	skillID := c.Param("skillId")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	manager := skill.GetManager()
	if manager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill manager not initialized"})
		return
	}

	if err := manager.SetUserEnabled(userID, skillID, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("skill %s %s for user %s", skillID, map[bool]string{true: "enabled", false: "disabled"}[req.Enabled], userID),
	})
}

// ListAvailableTools GET /api/skills/tools - 列出所有可用工具
func ListAvailableTools(c *gin.Context) {
	registry := skill.GetToolRegistry()
	if registry == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tool registry not initialized"})
		return
	}

	tools := registry.List()
	var result []gin.H
	for _, t := range tools {
		result = append(result, gin.H{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		})
	}

	c.JSON(http.StatusOK, gin.H{"tools": result})
}

// ========== 辅助函数 ==========

// unzip 解压 zip 文件
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 创建目录
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// 解压文件
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// findSkillDir 查找包含 SKILL.md 的目录
func findSkillDir(root string) (string, error) {
	// 先检查根目录
	if _, err := os.Stat(filepath.Join(root, "SKILL.md")); err == nil {
		return root, nil
	}

	// 遍历子目录
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", fmt.Errorf("failed to read directory")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(root, entry.Name())
			if _, err := os.Stat(filepath.Join(subDir, "SKILL.md")); err == nil {
				return subDir, nil
			}
		}
	}

	return "", fmt.Errorf("SKILL.md not found in uploaded package")
}

// downloadFile 下载文件
func downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
