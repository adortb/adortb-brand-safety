package blocklist

import (
	"sync"
)

// Rule 是广告主的一条黑名单规则。
type Rule struct {
	AdvertiserID int64
	Type         string // category / domain / keyword
	Value        string
	Action       string // block / warn
}

// BlockResult 表示黑名单检查结果。
type BlockResult struct {
	Blocked   bool
	BlockedBy []Rule // 触发拦截的规则列表
}

// AdvertiserBlocklist 管理广告主级别黑名单。
// 使用分段锁按广告主 ID 分片，减少并发竞争。
type AdvertiserBlocklist struct {
	shards [256]shard
}

type shard struct {
	mu    sync.RWMutex
	rules map[int64][]Rule
}

// NewAdvertiserBlocklist 创建广告主黑名单实例。
func NewAdvertiserBlocklist() *AdvertiserBlocklist {
	bl := &AdvertiserBlocklist{}
	for i := range bl.shards {
		bl.shards[i].rules = make(map[int64][]Rule)
	}
	return bl
}

func (a *AdvertiserBlocklist) shardFor(advertiserID int64) *shard {
	return &a.shards[uint8(advertiserID)]
}

// SetRules 批量设置某广告主的所有规则（覆盖写）。
func (a *AdvertiserBlocklist) SetRules(advertiserID int64, rules []Rule) {
	s := a.shardFor(advertiserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	// 创建新切片避免共享底层数组
	copied := make([]Rule, len(rules))
	copy(copied, rules)
	s.rules[advertiserID] = copied
}

// AddRule 追加单条规则。
func (a *AdvertiserBlocklist) AddRule(rule Rule) {
	s := a.shardFor(rule.AdvertiserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.AdvertiserID] = append(s.rules[rule.AdvertiserID], rule)
}

// DeleteRule 删除指定广告主的特定规则（按 type+value 匹配）。
func (a *AdvertiserBlocklist) DeleteRule(advertiserID int64, ruleType, value string) {
	s := a.shardFor(advertiserID)
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.rules[advertiserID]
	filtered := existing[:0]
	for _, r := range existing {
		if !(r.Type == ruleType && r.Value == value) {
			filtered = append(filtered, r)
		}
	}
	s.rules[advertiserID] = filtered
}

// Check 检查广告主 advertiserID 在含有给定类别/域名/关键词的页面是否可投放。
func (a *AdvertiserBlocklist) Check(advertiserID int64, categories []string, domain string, pageText string) BlockResult {
	s := a.shardFor(advertiserID)
	s.mu.RLock()
	rules := s.rules[advertiserID]
	s.mu.RUnlock()

	if len(rules) == 0 {
		return BlockResult{Blocked: false}
	}

	catSet := toSet(categories)
	var blockedBy []Rule

	for _, rule := range rules {
		switch rule.Type {
		case "category":
			if catSet[rule.Value] {
				blockedBy = append(blockedBy, rule)
			}
		case "domain":
			if domain == rule.Value {
				blockedBy = append(blockedBy, rule)
			}
		case "keyword":
			if containsWord(pageText, rule.Value) {
				blockedBy = append(blockedBy, rule)
			}
		}
	}

	blocked := false
	for _, r := range blockedBy {
		if r.Action == "block" {
			blocked = true
			break
		}
	}

	return BlockResult{Blocked: blocked, BlockedBy: blockedBy}
}

// ListRules 返回广告主的所有规则（只读副本）。
func (a *AdvertiserBlocklist) ListRules(advertiserID int64) []Rule {
	s := a.shardFor(advertiserID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.rules[advertiserID]
	out := make([]Rule, len(src))
	copy(out, src)
	return out
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, v := range items {
		m[v] = true
	}
	return m
}

func containsWord(text, keyword string) bool {
	if text == "" || keyword == "" {
		return false
	}
	// 简单子串匹配；生产中可替换为更精确的词边界匹配
	return len(text) >= len(keyword) && containsSubstring(text, keyword)
}

func containsSubstring(s, substr string) bool {
	n := len(substr)
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return true
		}
	}
	return false
}
