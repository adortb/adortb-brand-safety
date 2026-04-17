// Package scorer 计算综合品牌安全评分（0~100，越高越安全）。
package scorer

import (
	"github.com/adortb/adortb-brand-safety/internal/classifier"
)

// SafetyScore 表示综合评分结果。
type SafetyScore struct {
	Score      float64  // 0.0 ~ 1.0
	Categories []string // 命中的 IAB 类别
	Blocked    bool     // 是否命中平台硬封锁
	Warnings   []string // 告警信息
}

// Scorer 综合 URL 分类 + 关键词分类结果，产出最终安全评分。
type Scorer struct {
	urlClassifier     *classifier.URLClassifier
	keywordClassifier *classifier.KeywordClassifier
	taxonomy          *classifier.TaxonomyIndex
}

// New 创建 Scorer。
func New() *Scorer {
	return &Scorer{
		urlClassifier:     classifier.NewURLClassifier(),
		keywordClassifier: classifier.NewKeywordClassifier(),
		taxonomy:          classifier.NewTaxonomyIndex(classifier.DefaultTaxonomy()),
	}
}

// Score 对 URL + 标题文本进行综合评分。
func (s *Scorer) Score(rawURL, titleText string) SafetyScore {
	// 1. URL 分类
	urlCats, urlScore := s.urlClassifier.Classify(rawURL)

	// 2. 关键词分类（标题 + URL 路径）
	combined := rawURL + " " + titleText
	kwMatches := s.keywordClassifier.Classify(combined)

	// 3. 合并类别（去重）
	catSet := make(map[string]bool, len(urlCats)+len(kwMatches))
	for _, c := range urlCats {
		catSet[c] = true
	}
	for _, m := range kwMatches {
		if m.CategoryID != "" {
			catSet[m.CategoryID] = true
		}
	}

	allCats := make([]string, 0, len(catSet))
	for c := range catSet {
		allCats = append(allCats, c)
	}

	// 4. 计算惩罚
	penalty := 0.0
	var warnings []string
	blocked := false

	for _, m := range kwMatches {
		switch m.Severity {
		case "block":
			blocked = true
			penalty += 0.5
			warnings = append(warnings, "block:keyword:"+m.Keyword)
		case "warn":
			penalty += 0.15
			warnings = append(warnings, "warn:keyword:"+m.Keyword)
		case "flag":
			penalty += 0.05
			warnings = append(warnings, "flag:keyword:"+m.Keyword)
		}
	}

	// 检测是否命中敏感 IAB 类别
	for _, c := range allCats {
		if s.taxonomy.IsSensitive(c) {
			blocked = true
			warnings = append(warnings, "sensitive:category:"+c)
			penalty += 0.3
		}
	}

	// 5. 最终评分 = URL基础分 - 惩罚（下界 0）
	finalScore := urlScore - penalty
	if finalScore < 0 {
		finalScore = 0
	}
	if finalScore > 1 {
		finalScore = 1
	}

	return SafetyScore{
		Score:      finalScore,
		Categories: allCats,
		Blocked:    blocked,
		Warnings:   warnings,
	}
}
