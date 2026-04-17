package classifier

import (
	"strings"
	"unicode"
)

// KeywordMatch 表示一次关键词命中结果。
type KeywordMatch struct {
	Keyword    string
	CategoryID string
	Severity   string // block / warn / flag
}

// KeywordClassifier 扫描文本，检测敏感/危险关键词。
// 使用 Aho-Corasick 的简化版本：预处理关键词小写并逐一检查。
// 生产环境可替换为真正的 Aho-Corasick 树以支持海量关键词。
type KeywordClassifier struct {
	rules []keywordRule
}

type keywordRule struct {
	keyword    string
	categoryID string
	severity   string
}

// NewKeywordClassifier 创建带默认规则集的关键词分类器。
func NewKeywordClassifier() *KeywordClassifier {
	return &KeywordClassifier{rules: defaultKeywordRules()}
}

// Classify 扫描文本，返回所有命中的关键词匹配结果。
func (k *KeywordClassifier) Classify(text string) []KeywordMatch {
	normalized := normalizeText(text)
	var matches []KeywordMatch
	seen := make(map[string]bool)

	for _, rule := range k.rules {
		if seen[rule.keyword] {
			continue
		}
		if strings.Contains(normalized, rule.keyword) {
			seen[rule.keyword] = true
			matches = append(matches, KeywordMatch{
				Keyword:    rule.keyword,
				CategoryID: rule.categoryID,
				Severity:   rule.severity,
			})
		}
	}
	return matches
}

// HasBlockKeyword 快速判断文本是否包含任何 block 级关键词。
func (k *KeywordClassifier) HasBlockKeyword(text string) bool {
	normalized := normalizeText(text)
	for _, rule := range k.rules {
		if rule.severity == "block" && strings.Contains(normalized, rule.keyword) {
			return true
		}
	}
	return false
}

func normalizeText(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func defaultKeywordRules() []keywordRule {
	return []keywordRule{
		// 赌博
		{keyword: "gambling", categoryID: CatGambling, severity: "block"},
		{keyword: "casino", categoryID: CatGambling, severity: "block"},
		{keyword: "betting", categoryID: CatGambling, severity: "warn"},
		{keyword: "poker", categoryID: CatGambling, severity: "warn"},
		{keyword: "slot machine", categoryID: CatGambling, severity: "block"},
		{keyword: "sports betting", categoryID: CatGambling, severity: "block"},
		{keyword: "online casino", categoryID: CatGambling, severity: "block"},
		// 成人内容
		{keyword: "pornography", categoryID: CatAdult, severity: "block"},
		{keyword: "adult content", categoryID: CatAdult, severity: "block"},
		{keyword: "explicit", categoryID: CatAdult, severity: "flag"},
		{keyword: "nsfw", categoryID: CatAdult, severity: "block"},
		// 武器
		{keyword: "buy gun", categoryID: CatWeapons, severity: "block"},
		{keyword: "weapon", categoryID: CatWeapons, severity: "warn"},
		{keyword: "firearm", categoryID: CatWeapons, severity: "warn"},
		{keyword: "assault rifle", categoryID: CatWeapons, severity: "block"},
		{keyword: "ammunition", categoryID: CatWeapons, severity: "warn"},
		// 毒品
		{keyword: "buy drugs", categoryID: CatDrugs, severity: "block"},
		{keyword: "cocaine", categoryID: CatDrugs, severity: "block"},
		{keyword: "heroin", categoryID: CatDrugs, severity: "block"},
		{keyword: "methamphetamine", categoryID: CatDrugs, severity: "block"},
		{keyword: "cannabis", categoryID: CatDrugs, severity: "flag"},
		{keyword: "marijuana", categoryID: CatDrugs, severity: "flag"},
		{keyword: "illegal drug", categoryID: CatDrugs, severity: "block"},
		// 仇恨内容
		{keyword: "hate speech", categoryID: CatHate, severity: "block"},
		{keyword: "racial slur", categoryID: CatHate, severity: "block"},
		{keyword: "terrorism", categoryID: CatHate, severity: "block"},
		{keyword: "extremist", categoryID: CatHate, severity: "warn"},
		{keyword: "white supremacy", categoryID: CatHate, severity: "block"},
		// 违法内容
		{keyword: "piracy", categoryID: CatIllegal, severity: "block"},
		{keyword: "counterfeit", categoryID: CatIllegal, severity: "block"},
		{keyword: "fraud", categoryID: CatIllegal, severity: "warn"},
		{keyword: "money laundering", categoryID: CatIllegal, severity: "block"},
	}
}
