package yxyClient

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseHTMLAnnouncement 解析 HTML 公告，处理 p 标签换行
func ParseHTMLAnnouncement(htmlContent string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return []string{}
	}

	tags := doc.Find("p")
	result := make([]string, 0, tags.Length())
	// 遍历所有 p 标签
	tags.Each(func(i int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if text != "" {
			result = append(result, text)
		}
	})
	return result
}
