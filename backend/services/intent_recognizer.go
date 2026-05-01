package services

import (
	"regexp"
	"strings"

	"client-monitor/models"
)

// RecognizeIntent 识别用户意图
func RecognizeIntent(msg string) *models.AgentResult {
	msg = strings.TrimSpace(msg)

	// 标准化中文数字
	msg = normalizeChineseTimeDesc(msg)

	// 提醒意图
	if result := recognizeReminder(msg); result != nil {
		return result
	}

	// 记忆意图
	if result := recognizeMemory(msg); result != nil {
		return result
	}

	// 聊天意图
	if result := recognizeChat(msg); result != nil {
		return result
	}

	// 默认聊天
	return &models.AgentResult{Intent: models.IntentChat, Action: models.ActionNone}
}

// recognizeReminder 识别提醒意图
func recognizeReminder(msg string) *models.AgentResult {
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

	// ========== 增强的提醒规则 ==========

	// 半小时后提醒我xxx
	halfHourReminder := regexp.MustCompile(`(?i)(?:半|0\.5)\s*(?:小时|h)?\s*(?:之?后)\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := halfHourReminder.FindStringSubmatch(msg); len(matches) >= 2 {
		content := strings.TrimSpace(matches[1])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: "30分钟后"}
	}

	// X小时后（支持小数如1.5小时）
	hoursReminder := regexp.MustCompile(`(?i)(\d+\.?\d*)\s*(?:小时|h)\s*(?:之?后)\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := hoursReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hours := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: hours + "小时后"}
	}

	// X分钟后提醒我xxx（支持"之后"、"过"、"一下"）
	timeReminder := regexp.MustCompile(`(?i)(?:过|之后?)?\s*(\d+)\s*(?:分钟|min)\s*(?:之?后)?\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := timeReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		num := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: num + "分钟后"}
	}

	// 明天X点/X点钟（支持上午/下午/晚上）
	tomorrowReminder := regexp.MustCompile(`明天\s*(?:上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)(\d+)?\s*(?:分)?\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := tomorrowReminder.FindStringSubmatch(msg); len(matches) >= 4 {
		hour := matches[1]
		minute := matches[2]
		content := strings.TrimSpace(matches[3])
		timeStr := "明天" + hour + "点"
		if minute != "" {
			timeStr += minute + "分"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: timeStr}
	}

	// 后天X点
	dayAfterReminder := regexp.MustCompile(`后天\s*(?:上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := dayAfterReminder.FindStringSubmatch(msg); len(matches) >= 4 {
		hour := matches[1]
		minute := matches[2]
		content := strings.TrimSpace(matches[3])
		timeStr := "后天" + hour + "点"
		if minute != "" {
			timeStr += minute + "分"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: timeStr}
	}

	// 今晚X点
	tonightReminder := regexp.MustCompile(`今晚\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := tonightReminder.FindStringSubmatch(msg); len(matches) >= 4 {
		hour := matches[1]
		minute := matches[2]
		content := strings.TrimSpace(matches[3])
		timeStr := "今晚" + hour + "点"
		if minute != "" {
			timeStr += minute + "分"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: timeStr}
	}

	// 今天X点（支持上午/下午/晚上）
	todayReminder := regexp.MustCompile(`今天\s*(上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := todayReminder.FindStringSubmatch(msg); len(matches) >= 5 {
		period := matches[1]
		hour := matches[2]
		minute := matches[3]
		content := strings.TrimSpace(matches[4])
		timeStr := "今天"
		if period != "" {
			timeStr += period
		}
		timeStr += hour + "点"
		if minute != "" {
			timeStr += minute + "分"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: timeStr}
	}

	// X点半/X点一刻
	halfTimeReminder := regexp.MustCompile(`(\d+)\s*点半\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := halfTimeReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hour := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: hour + "点半"}
	}

	quarterTimeReminder := regexp.MustCompile(`(\d+)\s*点(?:一)?刻\s*(?:提醒|叫)\s*(?:我|一下)?\s*(.+)$`)
	if matches := quarterTimeReminder.FindStringSubmatch(msg); len(matches) >= 3 {
		hour := matches[1]
		content := strings.TrimSpace(matches[2])
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: hour + "点一刻"}
	}

	// 时间在后的格式：提醒我xxx，X分钟后
	timeAfterContent := regexp.MustCompile(`(?:提醒|叫)\s*(?:我|一下)?\s*(.+?)[，,]?\s*(\d+)\s*(?:分钟|小时)\s*(?:之?后)$`)
	if matches := timeAfterContent.FindStringSubmatch(msg); len(matches) >= 3 {
		content := strings.TrimSpace(matches[1])
		timeDesc := matches[2]
		// 检查是否有单位
		if strings.Contains(msg, "小时") {
			timeDesc += "小时后"
		} else {
			timeDesc += "分钟后"
		}
		return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: timeDesc}
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

	// "提醒一下xxx"
	if strings.HasPrefix(msg, "提醒一下") {
		content := strings.TrimSpace(strings.TrimPrefix(msg, "提醒一下"))
		if content != "" {
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
			{regexp.MustCompile(`(.+?)\s*后天(\d+)点$`), func(m []string) (string, string) { return m[1], "后天" + m[2] + "点" }},
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

	// ========== 兜底规则：需要同时满足动作关键词 + 时间相关信息 ==========
	// 这确保格式不匹配但有明确时间意图的消息也会走提醒流程
	actionKeywords := []string{"提醒", "叫我", "记得", "别忘了"}
	timeIndicators := []string{
		"分钟后", "小时后", "天后", "周后", // 相对时间
		"明天", "后天", "大后天", "今晚", // 绝对日期
		"下周", "上周", "这周", // 周相关
		"点半", "刻", // 具体时间点（避免单独的"点"误匹配）
		"上午", "下午", "晚上", "早上", "中午", // 时段
	}

	hasAction := false
	for _, kw := range actionKeywords {
		if strings.Contains(msg, kw) {
			hasAction = true
			break
		}
	}

	hasTime := false
	for _, ti := range timeIndicators {
		if strings.Contains(msg, ti) {
			hasTime = true
			break
		}
	}
	// 也检查数字+点的格式（如"3点"）
	if !hasTime {
		if matched, _ := regexp.MatchString(`\d+\s*点`, msg); matched {
			hasTime = true
		}
	}

	// 只有同时满足两个条件才走兜底
	if hasAction && hasTime {
		extractPattern := regexp.MustCompile(`(?:提醒|叫|记得)\s*(?:我|一下)?\s*(.+)$`)
		if matches := extractPattern.FindStringSubmatch(msg); len(matches) >= 2 {
			content := strings.TrimSpace(matches[1])
			if content != "" {
				// 把原始消息作为 timeDesc，让 LLM 解析完整的时间表达
				return &models.AgentResult{Intent: models.IntentReminder, Action: models.ActionCreate, Content: content, Time: msg}
			}
		}
	}

	return nil
}

// recognizeMemory 识别记忆意图
func recognizeMemory(msg string) *models.AgentResult {
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
func recognizeChat(msg string) *models.AgentResult {
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
