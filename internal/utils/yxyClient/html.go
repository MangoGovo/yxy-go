package yxyClient

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseHTMLAnnouncement 解析 HTML 公告，处理 p 标签换行
func ParseHTMLAnnouncement(htmlContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return ""
	}

	var result strings.Builder

	// 遍历所有 p 标签
	doc.Find("p").Each(func(i int, s *goquery.Selection) {
		// 获取 p 标签内的文本内容（会自动去除 HTML 标签）
		text := strings.TrimSpace(s.Text())
		if text != "" {
			if text != "各位师生：" {
				result.WriteString("\t")
			}
			result.WriteString(text)
			result.WriteString("\n")
		}
	})

	return strings.TrimSuffix(result.String(), "\n")
}
