package services

import (
	"regexp"
	"strings"

	"client-monitor/models"
)

// IntentRecognizer 意图识别器
type IntentRecognizer struct{}

// NewIntentRecognizer 创建意图识别器
func NewIntentRecognizer() *IntentRecognizer {
	return &IntentRecognizer{}
}

// Recognize 识别用户意图
func (r *IntentRecognizer) Recognize(msg string) *models.AgentResult {
	msg = strings.TrimSpace(msg)

	// 提醒意图
	if result := r.recognizeReminder(msg); result != nil {
		return result
	}

	// 记忆意图
	if result := r.recognizeMemory(msg); result != nil {
		return result
	}

	// 聊天意图
	if result := r.recognizeChat(msg); result != nil {
		return result
	}

	// 默认聊天
	return &models.AgentResult{Intent: models.IntentChat, Action: models.ActionNone}
}

// recognizeReminder 识别提醒意图
func (r *IntentRecognizer) recognizeReminder(msg string) *models.AgentResult {
	// 查看提醒列表
	if matched, _ := regexp.MatchString(`^(我有哪些|查看|列出).*(提醒|reminder)`, msg); matched {
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionList}
	}
	if matched, _ := regexp.MatchString(`^提醒列表$`, msg); matched {
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionList}
	}

	// 取消提醒
	if matched, _ := regexp.MatchString(`(取消|删除|关掉).*(提醒)`, msg); matched {
		re := regexp.MustCompile(`(?:取消|删除|关掉)\s*(?:提醒)?\s*(.+)`)
		matches := re.FindStringSubmatch(msg)
		content := ""
		if len(matches) > 1 {
			content = strings.TrimSpace(matches[1])
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCancel, Content: content}
	}

	// 创建提醒 - X分钟后提醒我xxx（宽松匹配，不要求开头）
	timeReminder := regexp.MustCompile(`(\d+)\s*(分钟|min|小时|hour|h)后\s*(?:提醒|叫)\s*我\s*(.+)$`)
	if matches := timeReminder.FindStringSubmatch(msg); len(matches) == 4 {
		num := matches[1]
		unit := matches[2]
		content := strings.TrimSpace(matches[3])
		if unit == "min" {
			unit = "分钟"
		} else if unit == "h" || unit == "hour" {
			unit = "小时"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: num + unit + "后"}
	}

	// 明天X点提醒我xxx
	tomorrowReminder := regexp.MustCompile(`明天(\d+)点(?:\d+分)?\s*(?:提醒|叫)\s*我\s*(.+)$`)
	if matches := tomorrowReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hour := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: "明天" + hour + "点"}
	}

	// 今晚X点提醒我xxx
	tonightReminder := regexp.MustCompile(`今晚(\d+)点\s*(?:提醒|叫)\s*我\s*(.+)$`)
	if matches := tonightReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hour := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: "今晚" + hour + "点"}
	}

	// 今天X点提醒我xxx
	todayReminder := regexp.MustCompile(`今天(\d+)点(?:\d+分)?\s*(?:提醒|叫)\s*我\s*(.+)$`)
	if matches := todayReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hour := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: "今天" + hour + "点"}
	}

	// "提醒我xxx" - 简单格式
	if strings.HasPrefix(msg, "提醒我") {
		content := strings.TrimSpace(strings.TrimPrefix(msg, "提醒我"))
		if content != "" {
			if matched, _ := regexp.MatchString(`\d+\s*(分钟|小时)后`, content); matched {
				re := regexp.MustCompile(`(.+?)\s*(\d+\s*(?:分钟|小时)后)$`)
				if matches := re.FindStringSubmatch(content); len(matches) == 3 {
					return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: strings.TrimSpace(matches[1]), Time: matches[2]}
				}
			}
			return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content}
		}
	}

	// "帮我xxx" - 作为提醒
	if strings.HasPrefix(msg, "帮我") || strings.HasPrefix(msg, "记得") || strings.HasPrefix(msg, "别忘了") {
		content := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(msg, "帮我"), "记得"), "别忘了"))
		timePatterns := []struct {
			re      *regexp.Regexp
			extract func([]string) (string, string)
		}{
			{regexp.MustCompile(`(.+?)\s*(\d+)\s*(分钟|小时)后$`), func(m []string) (string, string) { return m[1], m[2] + m[3] + "后" }},
			{regexp.MustCompile(`(.+?)\s*明天(\d+)点$`), func(m []string) (string, string) { return m[1], "明天" + m[2] + "点" }},
		}
		for _, tp := range timePatterns {
			if matches := tp.re.FindStringSubmatch(content); len(matches) >= 3 {
				remindContent, timeDesc := tp.extract(matches)
				return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: strings.TrimSpace(remindContent), Time: timeDesc}
			}
		}
		if content != "" {
			return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content}
		}
	}

	return nil
}

// recognizeMemory 识别记忆意图
func (r *IntentRecognizer) recognizeMemory(msg string) *models.AgentResult {
	// 记住xxx
	if strings.HasPrefix(msg, "记住") {
		content := strings.TrimSpace(strings.TrimPrefix(msg, "记住"))
		if content != "" {
			return &models.AgentResult{Intent: models.IntentMemory, Action: models.ActionCreate, Content: content}
		}
	}

	// "我的xxx是"
	if matched, _ := regexp.MatchString(`^我的\w+是`, msg); matched {
		return &models.AgentResult{Intent: models.IntentMemory, Action: models.ActionCreate, Content: msg}
	}

	// 查看记忆
	if matched, _ := regexp.MatchString(`^(我有哪些|查看|列出).*(记忆|记住了什么)`, msg); matched {
		return &models.AgentResult{Intent: models.IntentMemory, Action: models.ActionList}
	}
	if matched, _ := regexp.MatchString(`^你记住了`, msg); matched {
		return &models.AgentResult{Intent: models.IntentMemory, Action: models.ActionList}
	}

	// 删除记忆
	if matched, _ := regexp.MatchString(`(忘掉|删除|清除).*(记忆)`, msg); matched {
		re := regexp.MustCompile(`(?:忘掉|删除|清除)\s*(?:记忆)?\s*(.+)`)
		matches := re.FindStringSubmatch(msg)
		content := ""
		if len(matches) > 1 {
			content = strings.TrimSpace(matches[1])
		}
		return &models.AgentResult{Intent: models.IntentMemory, Action: models.ActionCancel, Content: content}
	}

	return nil
}

// recognizeChat 识别聊天意图
func (r *IntentRecognizer) recognizeChat(msg string) *models.AgentResult {
	// 简单问候
	greetings := []string{"你好", "您好", "嗨", "hi", "hello", "早上好", "晚上好", "在吗"}
	for _, g := range greetings {
		if strings.EqualFold(strings.TrimSpace(msg), g) {
			return &models.AgentResult{Intent: models.IntentChat, Action: models.ActionNone}
		}
	}

	// 如果包含提醒或记住关键词，不识别为聊天
	if strings.Contains(msg, "提醒") || strings.Contains(msg, "记住") {
		return nil
	}

	// 常见聊天关键词
	chatKeywords := []string{"什么", "怎么", "为什么", "如何", "是不是", "吗", "呢", "谁", "哪", "多少", "几"}
	for _, kw := range chatKeywords {
		if strings.Contains(msg, kw) {
			return &models.AgentResult{Intent: models.IntentChat, Action: models.ActionNone}
		}
	}

	// 常见聊天模式
	chatPatterns := []string{
		`^(什么|怎么|为什么|如何|谁|哪|多少|几)`,
		`(好吗|行吗|可以吗|对吗|是不是|对不对|呢吗)$`,
		`帮我(看|写|做|想|分析|查|找|解释|翻译)`,
		`^(谢谢|感谢|辛苦了|好的|好吧|嗯|哦|好)`,
		`(你觉得|你认为|你感觉|你喜欢)`,
		`^(讲个|说个|来个|给我)`,
		`(是什么|是怎么|是什么意思)`,
		`(怎么样|如何|好不好)`,
	}
	for _, pattern := range chatPatterns {
		if matched, _ := regexp.MatchString(pattern, msg); matched {
			return &models.AgentResult{Intent: models.IntentChat, Action: models.ActionNone}
		}
	}

	return nil
}
