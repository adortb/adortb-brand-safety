// Package api 提供 Brand Safety HTTP API。
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/adortb/adortb-brand-safety/internal/blocklist"
	"github.com/adortb/adortb-brand-safety/internal/classifier"
	"github.com/adortb/adortb-brand-safety/internal/scorer"
)

// Handler 提供 Brand Safety HTTP 接口。
type Handler struct {
	scorer   *scorer.Scorer
	urlCls   *classifier.URLClassifier
	kwCls    *classifier.KeywordClassifier
	advBL    *blocklist.AdvertiserBlocklist
	platBL   *blocklist.PlatformBlocklist
}

// New 创建 Handler。
func New(
	sc *scorer.Scorer,
	urlCls *classifier.URLClassifier,
	kwCls *classifier.KeywordClassifier,
	advBL *blocklist.AdvertiserBlocklist,
	platBL *blocklist.PlatformBlocklist,
) *Handler {
	return &Handler{
		scorer: sc,
		urlCls: urlCls,
		kwCls:  kwCls,
		advBL:  advBL,
		platBL: platBL,
	}
}

// RegisterRoutes 注册所有路由到 mux。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/classify", h.handleClassify)
	mux.HandleFunc("POST /v1/check", h.handleCheck)
	mux.HandleFunc("GET /v1/blocklists", h.handleListBlocklist)
	mux.HandleFunc("POST /v1/blocklists", h.handleAddBlocklist)
	mux.HandleFunc("DELETE /v1/blocklists", h.handleDeleteBlocklist)
	mux.HandleFunc("GET /health", handleHealth)
}

// ClassifyRequest POST /v1/classify 请求体。
type ClassifyRequest struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// ClassifyResponse POST /v1/classify 响应。
type ClassifyResponse struct {
	Categories  []string `json:"categories"`
	SafetyScore float64  `json:"safety_score"`
	Blocked     bool     `json:"blocked"`
	Warnings    []string `json:"warnings,omitempty"`
}

func (h *Handler) handleClassify(w http.ResponseWriter, r *http.Request) {
	var req ClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	result := h.scorer.Score(req.URL, req.Title)
	writeJSON(w, http.StatusOK, ClassifyResponse{
		Categories:  result.Categories,
		SafetyScore: result.Score,
		Blocked:     result.Blocked,
		Warnings:    result.Warnings,
	})
}

// CheckRequest POST /v1/check 请求体。
type CheckRequest struct {
	AdvertiserID int64  `json:"advertiser_id"`
	URL          string `json:"url"`
	Title        string `json:"title"`
}

// CheckResponse POST /v1/check 响应。
type CheckResponse struct {
	Allowed     bool          `json:"allowed"`
	BlockedBy   []blocklist.Rule `json:"blocked_by,omitempty"`
	Categories  []string      `json:"categories"`
	SafetyScore float64       `json:"safety_score"`
}

func (h *Handler) handleCheck(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	// 1. 评分 + 分类
	result := h.scorer.Score(req.URL, req.Title)

	// 2. 平台级检查
	if result.Blocked {
		writeJSON(w, http.StatusOK, CheckResponse{
			Allowed:    false,
			Categories: result.Categories,
			SafetyScore: result.Score,
		})
		return
	}

	// 提取域名供广告主检查
	domain := extractDomainSimple(req.URL)

	// 3. 广告主级检查
	if req.AdvertiserID > 0 {
		checkResult := h.advBL.Check(req.AdvertiserID, result.Categories, domain, req.Title)
		if checkResult.Blocked {
			writeJSON(w, http.StatusOK, CheckResponse{
				Allowed:     false,
				BlockedBy:   checkResult.BlockedBy,
				Categories:  result.Categories,
				SafetyScore: result.Score,
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, CheckResponse{
		Allowed:     true,
		Categories:  result.Categories,
		SafetyScore: result.Score,
	})
}

// BlocklistEntryRequest 黑名单添加请求。
type BlocklistEntryRequest struct {
	AdvertiserID int64  `json:"advertiser_id"`
	Type         string `json:"type"`
	Value        string `json:"value"`
	Action       string `json:"action"`
}

func (h *Handler) handleAddBlocklist(w http.ResponseWriter, r *http.Request) {
	var req BlocklistEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AdvertiserID == 0 || req.Type == "" || req.Value == "" {
		writeError(w, http.StatusBadRequest, "advertiser_id, type, value are required")
		return
	}
	action := req.Action
	if action == "" {
		action = "block"
	}
	if action != "block" && action != "warn" {
		writeError(w, http.StatusBadRequest, "action must be block or warn")
		return
	}
	validTypes := map[string]bool{"category": true, "domain": true, "keyword": true}
	if !validTypes[req.Type] {
		writeError(w, http.StatusBadRequest, "type must be category, domain, or keyword")
		return
	}

	h.advBL.AddRule(blocklist.Rule{
		AdvertiserID: req.AdvertiserID,
		Type:         req.Type,
		Value:        req.Value,
		Action:       action,
	})
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

func (h *Handler) handleListBlocklist(w http.ResponseWriter, r *http.Request) {
	advIDStr := r.URL.Query().Get("advertiser_id")
	if advIDStr == "" {
		writeError(w, http.StatusBadRequest, "advertiser_id is required")
		return
	}
	advID, err := strconv.ParseInt(advIDStr, 10, 64)
	if err != nil || advID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid advertiser_id")
		return
	}
	rules := h.advBL.ListRules(advID)
	writeJSON(w, http.StatusOK, map[string]any{"rules": rules, "total": len(rules)})
}

func (h *Handler) handleDeleteBlocklist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdvertiserID int64  `json:"advertiser_id"`
		Type         string `json:"type"`
		Value        string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AdvertiserID == 0 || req.Type == "" || req.Value == "" {
		writeError(w, http.StatusBadRequest, "advertiser_id, type, value are required")
		return
	}
	h.advBL.DeleteRule(req.AdvertiserID, req.Type, req.Value)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func extractDomainSimple(rawURL string) string {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	// 简单提取：找到 "://" 后到第一个 "/" 之间的部分
	parts := strings.SplitN(rawURL, "://", 2)
	if len(parts) < 2 {
		return ""
	}
	host := strings.SplitN(parts[1], "/", 2)[0]
	host = strings.TrimPrefix(host, "www.")
	return strings.ToLower(host)
}
