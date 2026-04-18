# adortb-brand-safety

> adortb 平台品牌安全服务，基于 IAB Content Taxonomy 3.0 对广告位页面进行分类和安全评分，支持广告主自定义黑名单。

## 快速理解

- **本项目做什么**：接收页面 URL + 标题，通过域名规则匹配 IAB 类别、关键词严重级别检测，输出综合安全评分（0~1.0）和 allowed/blocked 决策
- **架构位置**：adortb-adx 竞拍前调用，过滤品牌不安全的广告位
- **核心入口**：
  - 服务启动：`cmd/brand-safety/main.go`（端口 8092）
  - HTTP 入口：`internal/api/handler.go:Handler.RegisterRoutes`
  - 综合评分：`internal/scorer/safety_scorer.go:Scorer.Score`

## 目录结构

```
adortb-brand-safety/
├── cmd/brand-safety/main.go    # 主程序：组件初始化，优雅关闭
└── internal/
    ├── api/handler.go          # HTTP 路由（/v1/classify, /v1/check, /v1/blocklists）
    ├── classifier/
    │   ├── url_classifier.go   # URLClassifier：域名后缀 → IAB 类别 + 安全基准分
    │   ├── keyword_classifier.go # KeywordClassifier：关键词 → 严重级别（block/warn/flag）
    │   └── taxonomy.go         # IAB Content Taxonomy 3.0 类别定义 + 敏感类别标记
    ├── scorer/
    │   └── safety_scorer.go    # Scorer：URL分 + 关键词惩罚 → 最终评分 + blocked 标记
    └── blocklist/
        ├── advertiser.go       # AdvertiserBlocklist（category/domain/keyword 黑名单，内存存储）
        └── platform.go         # PlatformBlocklist（平台全局黑名单）
```

## 核心概念

### 评分流程（`scorer/safety_scorer.go:Scorer.Score`）

```
Score(rawURL, titleText)
    1. urlClassifier.Classify(rawURL)          → (urlCats, urlScore 0~1)
    2. keywordClassifier.Classify(url+title)   → []KeywordMatch{keyword, severity}
    3. 合并类别集合（去重）
    4. 计算惩罚：
       severity=block → penalty+=0.50, blocked=true
       severity=warn  → penalty+=0.15
       severity=flag  → penalty+=0.05
    5. taxonomy.IsSensitive(cat) → blocked=true, penalty+=0.30
    6. finalScore = urlScore - penalty（下界 0，上界 1）
```

### URLClassifier（`classifier/url_classifier.go`）

基于 `domainRule` 列表匹配（域名后缀 + 可选路径关键词）：
- 新闻：cnn.com/bbc.com → `IAB-12`，score 0.80~0.90
- 科技：techcrunch.com/github.com → `IAB-13`，score 0.95~0.98
- 赌博：bet365.com → `IAB-9`，score 0.10
- 成人：pornhub.com → `IAB-25-3`，score 0.01

### 广告主黑名单（`blocklist/advertiser.go:AdvertiserBlocklist`）

内存存储（`map[advertiserID][]Rule`），线程安全（`sync.RWMutex`）：
- `type: category`：拦截特定 IAB 类别的页面
- `type: domain`：拦截特定域名
- `type: keyword`：拦截包含特定关键词的内容
- `action: block/warn`

## 开发指南

### Go 版本

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 本地运行

```bash
export PORT=8092
go run cmd/brand-safety/main.go

# 分类测试
curl -X POST http://localhost:8092/v1/classify \
  -d '{"url":"https://www.techcrunch.com/ai","title":"AI startup raises $10M"}'

# 广告主级检查
curl -X POST http://localhost:8092/v1/check \
  -d '{"advertiser_id":1001,"url":"https://bet365.com","title":"Sports odds"}'
```

### 测试

```bash
go test ./... -cover -race
go test ./internal/scorer/... -v       # 评分器测试
go test ./internal/classifier/... -v  # 分类器测试
```

### 添加新的域名规则

在 `classifier/url_classifier.go:defaultDomainRules()` 末尾追加：
```go
{domainSuffix: "newsite.com", categories: []string{CatNews}, safetyScore: 0.85},
```

### 代码约定

- 评分是**纯内存计算**，无 DB 依赖，响应 < 1ms
- AdvertiserBlocklist 完全在内存中，服务重启后需重新加载（生产环境应接入 DB）
- URL 解析时自动补全协议（无 `://` 时加 `https://`）并去除 `www.`

## 依赖关系

- **上游**：adortb-adx（在竞拍前调用 /v1/check）
- **下游**：无（纯内存计算，无 DB）
- **依赖的库**：无外部运行时依赖

## 深入阅读

- IAB 类别常量定义：`internal/classifier/taxonomy.go`
- 关键词黑名单列表：`internal/classifier/keyword_classifier.go`
- 敏感类别判断逻辑：`taxonomy.go:TaxonomyIndex.IsSensitive`
- 平台黑名单初始化：`blocklist/platform.go:NewPlatformBlocklist`
