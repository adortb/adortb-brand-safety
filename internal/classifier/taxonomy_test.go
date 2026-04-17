package classifier

import (
	"testing"
)

func TestTaxonomyIndex(t *testing.T) {
	cats := DefaultTaxonomy()
	idx := NewTaxonomyIndex(cats)

	t.Run("Get 已知类别", func(t *testing.T) {
		c, ok := idx.Get(CatNews)
		if !ok {
			t.Fatal("expected CatNews to exist")
		}
		if c.Name == "" {
			t.Error("expected non-empty name")
		}
	})

	t.Run("Get 不存在类别返回 false", func(t *testing.T) {
		_, ok := idx.Get("IAB-999")
		if ok {
			t.Error("expected not found")
		}
	})

	t.Run("IsSensitive 成人内容", func(t *testing.T) {
		if !idx.IsSensitive(CatAdult) {
			t.Error("adult should be sensitive")
		}
	})

	t.Run("IsSensitive 新闻不敏感", func(t *testing.T) {
		if idx.IsSensitive(CatNews) {
			t.Error("news should not be sensitive")
		}
	})

	t.Run("DefaultTaxonomy 包含所有 Tier-1 类别", func(t *testing.T) {
		tier1Count := 0
		for _, c := range cats {
			if c.Tier == 1 {
				tier1Count++
			}
		}
		if tier1Count < 10 {
			t.Errorf("expected at least 10 Tier-1 categories, got %d", tier1Count)
		}
	})
}
