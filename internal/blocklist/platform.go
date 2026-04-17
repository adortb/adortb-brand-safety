// Package blocklist 管理平台级和广告主级黑名单规则。
package blocklist

import (
	"github.com/adortb/adortb-brand-safety/internal/classifier"
)

// PlatformBlocklist 维护平台级黑名单（所有广告主统一遵守）。
// 采用不可变设计，运行时通过 Reload() 替换整个列表。
type PlatformBlocklist struct {
	categories map[string]bool
	domains    map[string]bool
}

// NewPlatformBlocklist 使用默认平台规则初始化黑名单。
func NewPlatformBlocklist() *PlatformBlocklist {
	bl := &PlatformBlocklist{
		categories: make(map[string]bool),
		domains:    make(map[string]bool),
	}
	bl.loadDefaults()
	return bl
}

func (p *PlatformBlocklist) loadDefaults() {
	// 平台默认屏蔽的敏感类别
	for cat := range classifier.SensitiveCategories {
		p.categories[cat] = true
	}

	// 平台默认屏蔽的域名
	defaultBlockedDomains := []string{
		"pornhub.com",
		"xvideos.com",
		"xnxx.com",
		"bet365.com",
	}
	for _, d := range defaultBlockedDomains {
		p.domains[d] = true
	}
}

// IsBlockedCategory 判断某个 IAB 类别是否被平台屏蔽。
func (p *PlatformBlocklist) IsBlockedCategory(cat string) bool {
	return p.categories[cat]
}

// IsBlockedDomain 判断某个域名是否被平台屏蔽。
func (p *PlatformBlocklist) IsBlockedDomain(domain string) bool {
	return p.domains[domain]
}

// CheckCategories 检查类别列表，返回第一个被屏蔽的类别（空字符串=无命中）。
func (p *PlatformBlocklist) CheckCategories(cats []string) string {
	for _, c := range cats {
		if p.categories[c] {
			return c
		}
	}
	return ""
}

// BlockedCategories 返回平台屏蔽的类别列表（用于 API 展示）。
func (p *PlatformBlocklist) BlockedCategories() []string {
	result := make([]string, 0, len(p.categories))
	for cat := range p.categories {
		result = append(result, cat)
	}
	return result
}

// BlockedDomains 返回平台屏蔽的域名列表（用于 API 展示）。
func (p *PlatformBlocklist) BlockedDomains() []string {
	result := make([]string, 0, len(p.domains))
	for d := range p.domains {
		result = append(result, d)
	}
	return result
}
