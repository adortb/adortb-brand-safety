package classifier

import (
	"net/url"
	"strings"
)

// URLClassifier 根据域名/URL 路径推断 IAB 内容类别。
// 采用规则匹配，按域名前缀和路径关键词匹配。
type URLClassifier struct {
	rules []domainRule
}

type domainRule struct {
	domainSuffix string   // 匹配域名后缀，如 "cnn.com"
	pathKeywords []string // 路径中包含任一关键词则激活（空表示只看域名）
	categories   []string // 对应的 IAB 类别 ID
	safetyScore  float64  // 0.0(危险) ~ 1.0(安全)
}

// NewURLClassifier 创建带默认规则集的分类器。
func NewURLClassifier() *URLClassifier {
	return &URLClassifier{rules: defaultDomainRules()}
}

// Classify 分析 URL，返回匹配的 IAB 类别列表和安全评分。
// 未匹配到任何规则时返回 Uncategorized + 中性评分。
func (c *URLClassifier) Classify(rawURL string) (categories []string, safetyScore float64) {
	domain, path := extractDomainPath(rawURL)

	for _, rule := range c.rules {
		if !strings.HasSuffix(domain, rule.domainSuffix) {
			continue
		}
		if len(rule.pathKeywords) == 0 {
			return rule.categories, rule.safetyScore
		}
		for _, kw := range rule.pathKeywords {
			if strings.Contains(path, kw) {
				return rule.categories, rule.safetyScore
			}
		}
		// 域名命中但路径不匹配 → 返回基础类别（第一个规则的类别）
		return rule.categories, rule.safetyScore
	}

	return []string{CatUncategorized}, 0.7
}

func extractDomainPath(rawURL string) (domain, path string) {
	if rawURL == "" {
		return "", ""
	}
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, ""
	}
	host := strings.ToLower(u.Hostname())
	// 去掉 www. 前缀
	host = strings.TrimPrefix(host, "www.")
	return host, strings.ToLower(u.Path)
}

func defaultDomainRules() []domainRule {
	return []domainRule{
		// 新闻媒体
		{domainSuffix: "cnn.com", categories: []string{CatNews, "IAB-12-1"}, safetyScore: 0.80},
		{domainSuffix: "bbc.com", categories: []string{CatNews, "IAB-12-1"}, safetyScore: 0.85},
		{domainSuffix: "nytimes.com", categories: []string{CatNews, "IAB-12-1"}, safetyScore: 0.85},
		{domainSuffix: "reuters.com", categories: []string{CatNews, "IAB-12-1"}, safetyScore: 0.90},
		{domainSuffix: "theguardian.com", categories: []string{CatNews, "IAB-12-2"}, safetyScore: 0.85},
		{domainSuffix: "foxnews.com", categories: []string{CatNews, "IAB-12-2"}, safetyScore: 0.75},
		{domainSuffix: "dailymail.co.uk", categories: []string{CatNews, "IAB-1-2"}, safetyScore: 0.70},
		// 科技
		{domainSuffix: "techcrunch.com", categories: []string{CatTech, CatBusiness}, safetyScore: 0.95},
		{domainSuffix: "theverge.com", categories: []string{CatTech}, safetyScore: 0.95},
		{domainSuffix: "wired.com", categories: []string{CatTech, CatScience}, safetyScore: 0.95},
		{domainSuffix: "github.com", categories: []string{CatTech}, safetyScore: 0.98},
		{domainSuffix: "stackoverflow.com", categories: []string{CatTech, CatEducation}, safetyScore: 0.98},
		// 财经
		{domainSuffix: "bloomberg.com", categories: []string{CatBusiness, CatPersonalFin}, safetyScore: 0.92},
		{domainSuffix: "wsj.com", categories: []string{CatBusiness, CatNews}, safetyScore: 0.90},
		{domainSuffix: "ft.com", categories: []string{CatBusiness, CatPersonalFin}, safetyScore: 0.90},
		// 购物/电商
		{domainSuffix: "amazon.com", categories: []string{CatShopping}, safetyScore: 0.95},
		{domainSuffix: "ebay.com", categories: []string{CatShopping}, safetyScore: 0.93},
		// 娱乐
		{domainSuffix: "imdb.com", categories: []string{CatArts, "IAB-1-5"}, safetyScore: 0.95},
		{domainSuffix: "spotify.com", categories: []string{CatArts, "IAB-1-6"}, safetyScore: 0.95},
		{domainSuffix: "youtube.com", categories: []string{CatArts}, safetyScore: 0.80},
		// 体育
		{domainSuffix: "espn.com", categories: []string{CatSports}, safetyScore: 0.95},
		{domainSuffix: "nba.com", categories: []string{CatSports}, safetyScore: 0.97},
		// 旅游
		{domainSuffix: "tripadvisor.com", categories: []string{CatTravel}, safetyScore: 0.95},
		{domainSuffix: "booking.com", categories: []string{CatTravel}, safetyScore: 0.95},
		// 健康
		{domainSuffix: "webmd.com", categories: []string{CatHealthFitness}, safetyScore: 0.90},
		{domainSuffix: "mayoclinic.org", categories: []string{CatHealthFitness}, safetyScore: 0.95},
		// 赌博（低安全分）
		{domainSuffix: "bet365.com", categories: []string{CatGambling}, safetyScore: 0.10},
		{domainSuffix: "draftkings.com", categories: []string{CatGambling}, safetyScore: 0.20},
		{domainSuffix: "fanduel.com", categories: []string{CatGambling}, safetyScore: 0.20},
		// 成人（极低安全分）
		{domainSuffix: "pornhub.com", categories: []string{CatAdult}, safetyScore: 0.01},
		{domainSuffix: "xvideos.com", categories: []string{CatAdult}, safetyScore: 0.01},
		// 教育
		{domainSuffix: "coursera.org", categories: []string{CatEducation}, safetyScore: 0.98},
		{domainSuffix: "udemy.com", categories: []string{CatEducation}, safetyScore: 0.97},
		{domainSuffix: "wikipedia.org", categories: []string{CatEducation, CatScience}, safetyScore: 0.98},
	}
}
