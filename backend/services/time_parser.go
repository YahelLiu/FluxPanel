package services

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimeParser 时间解析器
type TimeParser struct{}

// NewTimeParser 创建时间解析器
func NewTimeParser() *TimeParser {
	return &TimeParser{}
}

// ParseTimeDescription 解析时间描述
func ParseTimeDescription(desc string) (time.Time, error) {
	desc = strings.TrimSpace(desc)

	// 先标准化中文数字
	desc = normalizeChineseTimeDesc(desc)

	now := time.Now()

	// 半小时后
	if matched, _ := regexp.MatchString(`^(?:半|0\.5)\s*(?:小时|h)?\s*(?:之?后)?$`, desc); matched {
		return now.Add(30 * time.Minute), nil
	}

	// 一个半小时后 / 1.5小时后（支持小数）
	if matches := regexp.MustCompile(`^(\d+\.?\d*)\s*(?:小时|h)\s*(?:之?后)?$`).FindStringSubmatch(desc); len(matches) == 2 {
		hours, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return now.Add(time.Duration(hours * float64(time.Hour))), nil
		}
	}

	// X分钟后 (支持 min)
	if matches := regexp.MustCompile(`^(\d+)\s*(?:分钟|min)`).FindStringSubmatch(desc); len(matches) == 2 {
		minutes := parseInt(matches[1])
		return now.Add(time.Duration(minutes) * time.Minute), nil
	}

	// X小时后 (支持 h, hour)
	if matches := regexp.MustCompile(`^(\d+)\s*(?:小时|h|hour)`).FindStringSubmatch(desc); len(matches) == 2 {
		hours := parseInt(matches[1])
		return now.Add(time.Duration(hours) * time.Hour), nil
	}

	// 后天
	if strings.Contains(desc, "后天") {
		return parseDayAfter(desc, now)
	}

	// 大后天
	if strings.Contains(desc, "大后天") {
		return parseDayAfterAfter(desc, now)
	}

	// 明天
	if strings.Contains(desc, "明天") {
		return parseTomorrow(desc, now)
	}

	// 今晚
	if strings.Contains(desc, "今晚") {
		return parseTonight(desc, now)
	}

	// 今天
	if strings.Contains(desc, "今天") {
		return parseToday(desc, now)
	}

	// X点半
	if matches := regexp.MustCompile(`^(\d+)\s*点半$`).FindStringSubmatch(desc); len(matches) == 2 {
		hour := parseInt(matches[1])
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 30, 0, 0, now.Location()), nil
	}

	// X点一刻
	if matches := regexp.MustCompile(`^(\d+)\s*点(?:一)?刻$`).FindStringSubmatch(desc); len(matches) == 2 {
		hour := parseInt(matches[1])
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 15, 0, 0, now.Location()), nil
	}

	// 下周X
	if strings.Contains(desc, "周") || strings.Contains(desc, "星期") {
		return parseWeekday(desc, now)
	}

	// 标准时间格式
	if t, err := time.Parse("2006-01-02 15:04", desc); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", desc); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s", desc)
}

// parseTomorrow 解析明天
func parseTomorrow(desc string, now time.Time) (time.Time, error) {
	tomorrow := now.AddDate(0, 0, 1)

	var hour, minute int

	// 明天上午X点/下午X点/晚上X点
	if matches := regexp.MustCompile(`明天\s*(上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?`).FindStringSubmatch(desc); len(matches) >= 3 {
		period := matches[1]
		hour = parseInt(matches[2])
		if len(matches) >= 4 && matches[3] != "" {
			minute = parseInt(matches[3])
		}

		// 根据时段调整小时
		if period == "下午" && hour < 12 {
			hour += 12
		} else if period == "晚上" && hour < 12 {
			hour += 12
		}

		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, minute, 0, 0, tomorrow.Location()), nil
	}

	if _, err := fmt.Sscanf(desc, "明天%d点%d分", &hour, &minute); err == nil {
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, minute, 0, 0, tomorrow.Location()), nil
	}
	if _, err := fmt.Sscanf(desc, "明天%d点", &hour); err == nil {
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), hour, 0, 0, 0, tomorrow.Location()), nil
	}

	// 默认明天上午9点
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 9, 0, 0, 0, tomorrow.Location()), nil
}

// parseTonight 解析今晚
func parseTonight(desc string, now time.Time) (time.Time, error) {
	var hour, minute int

	// 今晚X点X分
	if matches := regexp.MustCompile(`今晚\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?`).FindStringSubmatch(desc); len(matches) >= 2 {
		hour = parseInt(matches[1])
		if len(matches) >= 3 && matches[2] != "" {
			minute = parseInt(matches[2])
		}
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	if _, err := fmt.Sscanf(desc, "今晚%d点", &hour); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
	}
	// 默认今晚8点
	return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location()), nil
}

// parseToday 解析今天
func parseToday(desc string, now time.Time) (time.Time, error) {
	var hour, minute int

	// 今天上午/下午/晚上X点
	if matches := regexp.MustCompile(`今天\s*(上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?`).FindStringSubmatch(desc); len(matches) >= 3 {
		period := matches[1]
		hour = parseInt(matches[2])
		if len(matches) >= 4 && matches[3] != "" {
			minute = parseInt(matches[3])
		}

		// 根据时段调整小时
		if period == "下午" && hour < 12 {
			hour += 12
		} else if period == "晚上" && hour < 12 {
			hour += 12
		}

		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}

	if _, err := fmt.Sscanf(desc, "今天%d点%d分", &hour, &minute); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}
	if _, err := fmt.Sscanf(desc, "今天%d点", &hour); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", desc)
}

// parseDayAfter 解析后天
func parseDayAfter(desc string, now time.Time) (time.Time, error) {
	dayAfter := now.AddDate(0, 0, 2)

	var hour, minute int

	// 后天上午/下午/晚上X点
	if matches := regexp.MustCompile(`后天\s*(上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?`).FindStringSubmatch(desc); len(matches) >= 3 {
		period := matches[1]
		hour = parseInt(matches[2])
		if len(matches) >= 4 && matches[3] != "" {
			minute = parseInt(matches[3])
		}

		// 根据时段调整小时
		if period == "下午" && hour < 12 {
			hour += 12
		} else if period == "晚上" && hour < 12 {
			hour += 12
		}

		return time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), hour, minute, 0, 0, dayAfter.Location()), nil
	}

	if _, err := fmt.Sscanf(desc, "后天%d点%d分", &hour, &minute); err == nil {
		return time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), hour, minute, 0, 0, dayAfter.Location()), nil
	}
	if _, err := fmt.Sscanf(desc, "后天%d点", &hour); err == nil {
		return time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), hour, 0, 0, 0, dayAfter.Location()), nil
	}

	// 默认后天上午9点
	return time.Date(dayAfter.Year(), dayAfter.Month(), dayAfter.Day(), 9, 0, 0, 0, dayAfter.Location()), nil
}

// parseDayAfterAfter 解析大后天
func parseDayAfterAfter(desc string, now time.Time) (time.Time, error) {
	dayAfterAfter := now.AddDate(0, 0, 3)

	var hour, minute int

	if matches := regexp.MustCompile(`大后天\s*(上午|下午|晚上)?\s*(\d+)\s*(?:点|点钟)?(\d+)?\s*(?:分)?`).FindStringSubmatch(desc); len(matches) >= 3 {
		period := matches[1]
		hour = parseInt(matches[2])
		if len(matches) >= 4 && matches[3] != "" {
			minute = parseInt(matches[3])
		}

		// 根据时段调整小时
		if period == "下午" && hour < 12 {
			hour += 12
		} else if period == "晚上" && hour < 12 {
			hour += 12
		}

		return time.Date(dayAfterAfter.Year(), dayAfterAfter.Month(), dayAfterAfter.Day(), hour, minute, 0, 0, dayAfterAfter.Location()), nil
	}

	if _, err := fmt.Sscanf(desc, "大后天%d点", &hour); err == nil {
		return time.Date(dayAfterAfter.Year(), dayAfterAfter.Month(), dayAfterAfter.Day(), hour, 0, 0, 0, dayAfterAfter.Location()), nil
	}

	// 默认大后天上午9点
	return time.Date(dayAfterAfter.Year(), dayAfterAfter.Month(), dayAfterAfter.Day(), 9, 0, 0, 0, dayAfterAfter.Location()), nil
}

// parseWeekday 解析周几
func parseWeekday(desc string, now time.Time) (time.Time, error) {
	weekdayMap := map[string]time.Weekday{
		"周一": time.Monday, "周二": time.Tuesday, "周三": time.Wednesday,
		"周四": time.Thursday, "周五": time.Friday, "周六": time.Saturday, "周日": time.Sunday,
		"星期一": time.Monday, "星期二": time.Tuesday, "星期三": time.Wednesday,
		"星期四": time.Thursday, "星期五": time.Friday, "星期六": time.Saturday, "星期日": time.Sunday,
	}

	for name, weekday := range weekdayMap {
		if strings.Contains(desc, name) {
			daysUntil := int(weekday - now.Weekday())
			if daysUntil <= 0 {
				daysUntil += 7
			}
			target := now.AddDate(0, 0, daysUntil)

			var hour int
			if _, err := fmt.Sscanf(desc, "%s%d点", name, &hour); err == nil {
				return time.Date(target.Year(), target.Month(), target.Day(), hour, 0, 0, 0, target.Location()), nil
			}
			return time.Date(target.Year(), target.Month(), target.Day(), 9, 0, 0, 0, target.Location()), nil
		}
	}

	return time.Time{}, fmt.Errorf("无法解析时间: %s", desc)
}

// parseInt 解析整数
func parseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// chineseNumMap 中文数字映射
var chineseNumMap = map[rune]int{
	'零': 0, '〇': 0,
	'一': 1, '壹': 1,
	'二': 2, '贰': 2, '两': 2,
	'三': 3, '叁': 3,
	'四': 4, '肆': 4,
	'五': 5, '伍': 5,
	'六': 6, '陆': 6,
	'七': 7, '柒': 7,
	'八': 8, '捌': 8,
	'九': 9, '玖': 9,
	'十': 10, '拾': 10,
	'百': 100, '佰': 100,
}

// parseChineseNumber 解析中文数字（支持"十"、"二十"、"十五"、"一百二十"等）
func parseChineseNumber(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// 先尝试阿拉伯数字
	if n, err := strconv.Atoi(s); err == nil {
		return n, true
	}

	// 中文数字解析
	result := 0
	temp := 0
	hasValidChar := false

	for _, c := range s {
		if val, ok := chineseNumMap[c]; ok {
			hasValidChar = true
			if val >= 10 {
				// 十、百等单位
				if temp == 0 {
					temp = 1 // "十" 默认是"一十"
				}
				if val == 10 {
					result += temp * 10
				} else if val == 100 {
					result += temp * 100
				}
				temp = 0
			} else {
				temp = val
			}
		} else if c >= '0' && c <= '9' {
			// 混合阿拉伯数字
			hasValidChar = true
			temp = temp*10 + int(c-'0')
		} else if c == '点' || c == '.' {
			// 小数点，暂时跳过
			continue
		}
	}

	result += temp
	return result, hasValidChar
}

// normalizeChineseTimeDesc 将中文时间描述中的中文数字转换为阿拉伯数字
func normalizeChineseTimeDesc(desc string) string {
	// 替换常见的中文数字时间表达
	replacements := []struct {
		pattern *regexp.Regexp
		replace func([]string) string
	}{
		// 半小时 -> 30分钟
		{regexp.MustCompile(`半小时`), func(m []string) string { return "30分钟" }},
		// 一个半小时 -> 90分钟
		{regexp.MustCompile(`一个?半小时`), func(m []string) string { return "90分钟" }},
		// 两小时/两个小时 -> 2小时
		{regexp.MustCompile(`两|两个?`), func(m []string) string { return "2" }},
	}

	result := desc
	for _, r := range replacements {
		result = r.pattern.ReplaceAllStringFunc(result, func(s string) string {
			return r.replace([]string{s})
		})
	}

	// 处理"十分钟后"这种格式
	// 中文数字 + 单位
	chineseNumPattern := regexp.MustCompile(`([零〇一二三四五六七八九十百两壹贰叁肆伍陆柒捌玖拾佰]+)\s*(分钟|小时|天|周)`)
	result = chineseNumPattern.ReplaceAllStringFunc(result, func(s string) string {
		matches := chineseNumPattern.FindStringSubmatch(s)
		if len(matches) >= 3 {
			if num, ok := parseChineseNumber(matches[1]); ok {
				return fmt.Sprintf("%d%s", num, matches[2])
			}
		}
		return s
	})

	// 处理"十点"这种格式
	chineseTimePattern := regexp.MustCompile(`([零〇一二三四五六七八九十两壹贰叁肆伍陆柒捌玖拾]+)\s*点`)
	result = chineseTimePattern.ReplaceAllStringFunc(result, func(s string) string {
		matches := chineseTimePattern.FindStringSubmatch(s)
		if len(matches) >= 2 {
			if num, ok := parseChineseNumber(matches[1]); ok {
				return fmt.Sprintf("%d点", num)
			}
		}
		return s
	})

	return result
}
