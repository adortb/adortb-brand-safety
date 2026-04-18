# adortb-brand-safety 内部架构

## 内部架构图

```
┌──────────────────────────────────────────────────────────────┐
│                  adortb-brand-safety 内部架构                 │
│                                                              │
│  HTTP 请求                                                   │
│      │                                                       │
│      ▼                                                       │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  internal/api/handler.go  Handler                    │   │
│  │                                                      │   │
│  │  POST /v1/classify    → Scorer.Score()               │   │
│  │  POST /v1/check       → Scorer.Score() + Blocklist   │   │
│  │  GET  /v1/categories  → TaxonomyIndex                │   │
│  │  GET/POST/DELETE /v1/blocklists                      │   │
│  │  GET  /v1/blocklist/global                           │   │
│  └──────────┬────────────────────────────────────────── ┘   │
│             │                                               │
│             ▼                                               │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  internal/scorer/safety_scorer.go  Scorer            │   │
│  │                                                      │   │
│  │  Score(rawURL, titleText) → SafetyScore              │   │
│  │   1. urlCls.Classify(rawURL) → (cats, baseScore)     │   │
│  │   2. kwCls.Classify(url+title) → []KeywordMatch      │   │
│  │   3. 合并类别集合                                     │   │
│  │   4. 计算惩罚（block/warn/flag/sensitive）            │   │
│  │   5. finalScore = baseScore - penalty                │   │
│  └──────┬──────────────────────┬───────────────────────┘   │
│         │                      │                           │
│         ▼                      ▼                           │
│  ┌──────────────┐    ┌──────────────────────────────┐      │
│  │  classifier/ │    │  classifier/                 │      │
│  │  url_         │    │  keyword_classifier.go       │      │
│  │  classifier  │    │                              │      │
│  │  .go         │    │  KeywordClassifier           │      │
│  │              │    │  Classify(text)              │      │
│  │  URLClassifier│    │  → []Match{kw, sev, catID}  │      │
│  │  Classify()  │    └──────────────────────────────┘      │
│  │  domainRules │                                          │
│  └──────────────┘                                          │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  blocklist/advertiser.go  AdvertiserBlocklist        │   │
│  │  map[advertiserID][]Rule  (sync.RWMutex)             │   │
│  │  AddRule / DeleteRule / ListRules / Check            │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  blocklist/platform.go  PlatformBlocklist            │   │
│  │  硬编码：赌博/成人/恶意软件类别 + 全局域名黑名单       │   │
│  └──────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

## 数据流

### classify 请求数据流

```
POST /v1/classify {url, title}
    │
    ▼
Handler.handleClassify()
    │
    ▼
Scorer.Score(rawURL, titleText)
    │
    ├─[1]─► URLClassifier.Classify(rawURL)
    │        extractDomainPath(rawURL) → (domain, path)
    │        for rule in defaultDomainRules():
    │          strings.HasSuffix(domain, rule.domainSuffix)?
    │          len(rule.pathKeywords)==0 OR path 含 keyword?
    │          → (rule.categories, rule.safetyScore)
    │        未命中 → (["Uncategorized"], 0.7)
    │
    ├─[2]─► KeywordClassifier.Classify(url + " " + title)
    │        for kw in bannedKeywords:
    │          strings.Contains(lower(text), kw)?
    │          → Match{keyword, severity, categoryID}
    │
    ├─[3]─► 合并类别集合（URL类别 ∪ 关键词类别，set 去重）
    │
    ├─[4]─► 计算惩罚：
    │        severity=block   → penalty+=0.50, blocked=true
    │        severity=warn    → penalty+=0.15
    │        severity=flag    → penalty+=0.05
    │        taxonomy.IsSensitive(cat) → penalty+=0.30, blocked=true
    │
    └─[5]─► finalScore = max(0, min(1, urlScore - penalty))
             返回 SafetyScore{Score, Categories, Blocked, Warnings}
```

### check 请求数据流（广告主级检查）

```
POST /v1/check {advertiser_id, url, title}
    │
    ▼
1. Scorer.Score() → SafetyScore
   │
   ├── Blocked=true → 立即返回 allowed=false
   │
   └── Blocked=false
       │
       ▼
2. AdvertiserBlocklist.Check(advertiserID, categories, domain, title)
   for rule in rules[advertiserID]:
     type=category → categories contains rule.Value?
     type=domain   → domain == rule.Value?
     type=keyword  → title contains rule.Value?
   │
   ├── blocked → allowed=false + blocked_by
   └── ok      → allowed=true
```

## 时序图

```
adortb-adx       Handler      Scorer      URLClassifier  KeywordClassifier
    │               │             │              │               │
    │─POST /check──►              │              │               │
    │               │──Score()───►│              │               │
    │               │             │──Classify()─►│               │
    │               │             │◄─(cats,score)│               │
    │               │             │──Classify()─────────────────►│
    │               │             │◄─[]Match──────────────────── │
    │               │             │  计算惩罚    │               │
    │               │◄─SafetyScore│              │               │
    │               │  blocked?   │              │               │
    │               │  AdvBL.Check│              │               │
    │◄─200 response─│             │              │               │
```

## 状态机

### SafetyScore 决策树

```
Scorer.Score(url, title)
    │
    ├── urlScore（域名基准分）
    │    0.95~0.99: 科技/教育
    │    0.80~0.94: 新闻/财经/娱乐
    │    0.70~0.79: 中立内容
    │    0.10~0.19: 赌博
    │    0.01~0.09: 成人
    │
    ├── keyword penalty（-0.05 ~ -0.50 per keyword）
    │
    ├── sensitive category penalty（-0.30）
    │
    └── finalScore = urlScore - Σ(penalties)
         │
         ├── Blocked=true → /v1/check returns allowed=false (立即拒绝)
         ├── Score < 0.3  → 高风险，广告主可配置拦截
         └── Score >= 0.7 → 相对安全
```

### AdvertiserBlocklist 规则类型

```
Rule.Type:
    "category" → 匹配 IAB 类别 ID（如 "IAB-9" 赌博）
    "domain"   → 精确匹配域名（如 "competitor.com"）
    "keyword"  → 文本包含匹配（如 "gambling"）

Rule.Action:
    "block" → CheckResult.Blocked=true
    "warn"  → 记录到 CheckResult.Warnings（不拦截）
```
