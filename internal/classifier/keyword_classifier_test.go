package classifier

import (
	"testing"
)

func TestKeywordClassifier(t *testing.T) {
	cls := NewKeywordClassifier()

	t.Run("命中 block 级赌博关键词", func(t *testing.T) {
		matches := cls.Classify("best online casino games")
		if len(matches) == 0 {
			t.Fatal("expected matches, got none")
		}
		foundBlock := false
		for _, m := range matches {
			if m.Severity == "block" {
				foundBlock = true
				break
			}
		}
		if !foundBlock {
			t.Error("expected block severity match")
		}
	})

	t.Run("命中毒品关键词", func(t *testing.T) {
		matches := cls.Classify("buy cocaine online")
		if len(matches) == 0 {
			t.Fatal("expected matches")
		}
		if matches[0].CategoryID != CatDrugs {
			t.Errorf("expected CatDrugs, got %q", matches[0].CategoryID)
		}
	})

	t.Run("HasBlockKeyword 返回 true", func(t *testing.T) {
		if !cls.HasBlockKeyword("illegal slot machine operations") {
			t.Error("expected HasBlockKeyword = true")
		}
	})

	t.Run("安全文本不命中", func(t *testing.T) {
		matches := cls.Classify("latest technology news from silicon valley")
		for _, m := range matches {
			if m.Severity == "block" {
				t.Errorf("unexpected block match on safe text: %+v", m)
			}
		}
	})

	t.Run("空文本无命中", func(t *testing.T) {
		matches := cls.Classify("")
		if len(matches) != 0 {
			t.Errorf("expected 0 matches for empty text, got %d", len(matches))
		}
	})

	t.Run("大小写不敏感", func(t *testing.T) {
		matches := cls.Classify("GAMBLING IS FORBIDDEN HERE")
		if len(matches) == 0 {
			t.Error("expected case-insensitive match for GAMBLING")
		}
	})
}
