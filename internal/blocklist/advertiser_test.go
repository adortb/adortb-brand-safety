package blocklist

import (
	"testing"
)

func TestAdvertiserBlocklist(t *testing.T) {
	bl := NewAdvertiserBlocklist()

	t.Run("空规则时允许", func(t *testing.T) {
		result := bl.Check(1, []string{"IAB-1"}, "example.com", "")
		if result.Blocked {
			t.Error("expected allowed with no rules")
		}
	})

	t.Run("类别黑名单拦截", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 2, Type: "category", Value: "IAB-9-7", Action: "block"})

		result := bl.Check(2, []string{"IAB-9-7", "IAB-1"}, "example.com", "")
		if !result.Blocked {
			t.Error("expected blocked")
		}
		if len(result.BlockedBy) == 0 {
			t.Error("expected BlockedBy to be populated")
		}
	})

	t.Run("域名黑名单拦截", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 3, Type: "domain", Value: "badnews.com", Action: "block"})

		result := bl.Check(3, []string{"IAB-12"}, "badnews.com", "")
		if !result.Blocked {
			t.Error("expected blocked by domain")
		}
	})

	t.Run("关键词黑名单拦截", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 4, Type: "keyword", Value: "gambling", Action: "block"})

		result := bl.Check(4, []string{"IAB-1"}, "example.com", "latest gambling news")
		if !result.Blocked {
			t.Error("expected blocked by keyword")
		}
	})

	t.Run("warn action 不拦截但记录", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 5, Type: "category", Value: "IAB-12", Action: "warn"})

		result := bl.Check(5, []string{"IAB-12"}, "news.com", "")
		if result.Blocked {
			t.Error("warn action should not block")
		}
		if len(result.BlockedBy) == 0 {
			t.Error("expected BlockedBy to record warning")
		}
	})

	t.Run("DeleteRule 后规则失效", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 6, Type: "domain", Value: "example.com", Action: "block"})
		bl.DeleteRule(6, "domain", "example.com")

		result := bl.Check(6, []string{}, "example.com", "")
		if result.Blocked {
			t.Error("expected not blocked after delete")
		}
	})

	t.Run("SetRules 覆盖写", func(t *testing.T) {
		bl.AddRule(Rule{AdvertiserID: 7, Type: "domain", Value: "old.com", Action: "block"})
		bl.SetRules(7, []Rule{
			{AdvertiserID: 7, Type: "domain", Value: "new.com", Action: "block"},
		})

		result1 := bl.Check(7, nil, "old.com", "")
		result2 := bl.Check(7, nil, "new.com", "")

		if result1.Blocked {
			t.Error("old.com should not be blocked after SetRules")
		}
		if !result2.Blocked {
			t.Error("new.com should be blocked")
		}
	})

	t.Run("ListRules 返回副本", func(t *testing.T) {
		bl.SetRules(8, []Rule{
			{AdvertiserID: 8, Type: "category", Value: "IAB-1", Action: "block"},
		})
		rules := bl.ListRules(8)
		if len(rules) != 1 {
			t.Errorf("expected 1 rule, got %d", len(rules))
		}
		// 修改返回值不影响原始数据
		rules[0].Action = "warn"
		rules2 := bl.ListRules(8)
		if rules2[0].Action != "block" {
			t.Error("ListRules should return a copy, not a reference")
		}
	})
}
