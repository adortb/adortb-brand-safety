package scorer

import (
	"testing"
)

func TestScorer(t *testing.T) {
	sc := New()

	t.Run("新闻站点高安全分", func(t *testing.T) {
		result := sc.Score("https://www.reuters.com/business/finance", "Stock Market News")
		if result.Score < 0.7 {
			t.Errorf("expected high safety score for news, got %.2f", result.Score)
		}
		if result.Blocked {
			t.Error("news site should not be blocked")
		}
	})

	t.Run("赌博站点低安全分且拦截", func(t *testing.T) {
		result := sc.Score("https://bet365.com/sports", "bet on sports")
		if result.Score > 0.3 {
			t.Errorf("expected low safety score for gambling, got %.2f", result.Score)
		}
		if !result.Blocked {
			t.Error("gambling site should be blocked")
		}
	})

	t.Run("成人站点被拦截", func(t *testing.T) {
		result := sc.Score("https://pornhub.com", "")
		if !result.Blocked {
			t.Error("adult site should be blocked")
		}
		if result.Score > 0.1 {
			t.Errorf("adult site score should be near 0, got %.2f", result.Score)
		}
	})

	t.Run("包含赌博关键词的页面", func(t *testing.T) {
		result := sc.Score("https://example.com/news", "online casino offers")
		if result.Score >= 0.8 {
			t.Errorf("page with gambling keywords should have reduced score, got %.2f", result.Score)
		}
	})

	t.Run("类别不为空", func(t *testing.T) {
		result := sc.Score("https://techcrunch.com/ai", "AI news")
		if len(result.Categories) == 0 {
			t.Error("expected non-empty categories")
		}
	})

	t.Run("评分始终在 0~1 范围内", func(t *testing.T) {
		urls := []string{
			"https://cnn.com",
			"https://bet365.com",
			"https://unknown-xyz.io",
			"https://pornhub.com",
			"",
		}
		for _, u := range urls {
			result := sc.Score(u, "test content with gambling keywords")
			if result.Score < 0 || result.Score > 1 {
				t.Errorf("score %.4f out of [0, 1] for URL %q", result.Score, u)
			}
		}
	})
}
