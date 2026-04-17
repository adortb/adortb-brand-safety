package classifier

import (
	"testing"
)

func TestURLClassifier(t *testing.T) {
	cls := NewURLClassifier()

	tests := []struct {
		name          string
		url           string
		wantCatContains string
		wantMinScore  float64
		wantMaxScore  float64
	}{
		{
			name:          "新闻站点 cnn.com",
			url:           "https://www.cnn.com/news/world",
			wantCatContains: CatNews,
			wantMinScore:  0.7,
			wantMaxScore:  1.0,
		},
		{
			name:          "科技站点 techcrunch.com",
			url:           "https://techcrunch.com/2026/ai-news",
			wantCatContains: CatTech,
			wantMinScore:  0.9,
			wantMaxScore:  1.0,
		},
		{
			name:          "赌博站点 bet365.com",
			url:           "https://bet365.com",
			wantCatContains: CatGambling,
			wantMinScore:  0.0,
			wantMaxScore:  0.2,
		},
		{
			name:          "成人站点",
			url:           "https://pornhub.com",
			wantCatContains: CatAdult,
			wantMinScore:  0.0,
			wantMaxScore:  0.05,
		},
		{
			name:          "未知站点返回 Uncategorized",
			url:           "https://unknown-xyz-12345.io/page",
			wantCatContains: CatUncategorized,
			wantMinScore:  0.5,
			wantMaxScore:  0.9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cats, score := cls.Classify(tc.url)

			found := false
			for _, c := range cats {
				if c == tc.wantCatContains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected category %q in %v", tc.wantCatContains, cats)
			}

			if score < tc.wantMinScore || score > tc.wantMaxScore {
				t.Errorf("score %.2f not in [%.2f, %.2f]", score, tc.wantMinScore, tc.wantMaxScore)
			}
		})
	}
}

func TestExtractDomainPath(t *testing.T) {
	tests := []struct {
		url        string
		wantDomain string
	}{
		{"https://www.cnn.com/news", "cnn.com"},
		{"http://bbc.com/sport", "bbc.com"},
		{"example.com/path", "example.com"},
		{"", "", },
	}

	for _, tc := range tests {
		domain, _ := extractDomainPath(tc.url)
		if domain != tc.wantDomain {
			t.Errorf("extractDomainPath(%q) domain = %q, want %q", tc.url, domain, tc.wantDomain)
		}
	}
}
