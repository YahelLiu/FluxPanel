package services

import (
	"fmt"
	"regexp"
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
	now := time.Now()

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
	var hour int
	if _, err := fmt.Sscanf(desc, "今晚%d点", &hour); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
	}
	// 默认今晚8点
	return time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location()), nil
}

// parseToday 解析今天
func parseToday(desc string, now time.Time) (time.Time, error) {
	var hour, minute int
	if _, err := fmt.Sscanf(desc, "今天%d点%d分", &hour, &minute); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
	}
	if _, err := fmt.Sscanf(desc, "今天%d点", &hour); err == nil {
		return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location()), nil
	}
	return time.Time{}, fmt.Errorf("无法解析时间: %s", desc)
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
