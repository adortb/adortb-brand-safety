package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adortb/adortb-brand-safety/internal/blocklist"
	"github.com/adortb/adortb-brand-safety/internal/classifier"
	"github.com/adortb/adortb-brand-safety/internal/scorer"
)

func newTestHandler() *Handler {
	return New(
		scorer.New(),
		classifier.NewURLClassifier(),
		classifier.NewKeywordClassifier(),
		blocklist.NewAdvertiserBlocklist(),
		blocklist.NewPlatformBlocklist(),
	)
}

func TestHandleClassify(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("新闻站点分类", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"url":   "https://reuters.com/business",
			"title": "Market Update",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp ClassifyResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Categories) == 0 {
			t.Error("expected non-empty categories")
		}
		if resp.SafetyScore <= 0 {
			t.Error("expected positive safety score")
		}
	})

	t.Run("缺少 URL 返回 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"title": "test"})
		req := httptest.NewRequest(http.MethodPost, "/v1/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", w.Code)
		}
	})
}

func TestHandleCheck(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("安全站点允许投放", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"advertiser_id": 1,
			"url":           "https://reuters.com",
			"title":         "Financial News",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp CheckResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if !resp.Allowed {
			t.Error("expected allowed=true for safe site")
		}
	})

	t.Run("成人站点不允许", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"advertiser_id": 1,
			"url":           "https://pornhub.com",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		var resp CheckResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Allowed {
			t.Error("adult site should not be allowed")
		}
	})

	t.Run("广告主自定义黑名单拦截", func(t *testing.T) {
		const advID = int64(100)
		h.advBL.AddRule(blocklist.Rule{
			AdvertiserID: advID,
			Type:         "domain",
			Value:        "competitor.com",
			Action:       "block",
		})

		body, _ := json.Marshal(map[string]any{
			"advertiser_id": advID,
			"url":           "https://competitor.com/news",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/check", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		var resp CheckResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Allowed {
			t.Error("should be blocked by advertiser blocklist")
		}
	})
}

func TestHandleBlocklistCRUD(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	t.Run("添加黑名单规则", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"advertiser_id": 200,
			"type":          "category",
			"value":         "IAB-9-7",
			"action":        "block",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/blocklists", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("查询黑名单规则", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/blocklists?advertiser_id=200", nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]any
		json.NewDecoder(w.Body).Decode(&resp)
		total, _ := resp["total"].(float64)
		if total < 1 {
			t.Errorf("expected at least 1 rule, got %.0f", total)
		}
	})
}

func TestHandleHealth(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
