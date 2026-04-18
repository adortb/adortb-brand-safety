# adortb-brand-safety

> adortb 平台的品牌安全服务，提供 IAB Content Taxonomy 3.0 分类、关键词黑名单检测、URL 安全评分和广告主级黑名单管理，确保广告不出现在品牌不适宜的页面上。

## 架构定位

```
┌─────────────────────────────────────────────────────────────────┐
│                      adortb 平台整体架构                         │
│                                                                  │
│  竞拍请求（含 page_url）                                         │
│       │                                                         │
│       ▼                                                         │
│  ★ adortb-brand-safety (Brand Safety Service)                  │
│       │                                                         │
│  ┌────┼──────────────────────────────────────────┐              │
│  │    ▼                                          │              │
│  │  [URLClassifier]    域名→IAB 类别映射           │              │
│  │       ↓                                       │              │
│  │  [KeywordClassifier] 关键词匹配（block/warn）   │              │
│  │       ↓                                       │              │
│  │  [Scorer]          综合安全评分（0~1.0）        │              │
│  │       ↓                                       │              │
│  │  [Blocklist]       广告主 + 平台级黑名单检查     │              │
│  └──────────────────────────────────────────────┘              │
│                                                                  │
│  adortb-adx 调用 /v1/check 决定是否允许广告投放                  │
└─────────────────────────────────────────────────────────────────┘
```

品牌安全服务在竞拍前被调用，对广告位页面进行**实时分类和安全评分**，支持广告主自定义黑名单规则。

## 目录结构

```
adortb-brand-safety/
├── go.mod                          # Go 1.25.3，无外部运行时依赖
├── cmd/brand-safety/
│   └── main.go                     # 主程序：服务初始化、优雅关闭（端口 8092）
├── client/                         # Go 客户端
├── migrations/                     # 数据库迁移（广告主黑名单持久化）
└── internal/
    ├── api/
    │   └── handler.go              # HTTP 路由：/v1/classify, /v1/check, /v1/blocklists
    ├── classifier/
    │   ├── url_classifier.go       # URLClassifier：域名/路径 → IAB 类别 + 安全分
    │   ├── keyword_classifier.go   # KeywordClassifier：关键词 → 严重级别（block/warn/flag）
    │   └── taxonomy.go             # IAB Content Taxonomy 3.0 类别树 + 敏感类别标记
    ├── scorer/
    │   └── safety_scorer.go        # Scorer：综合 URL 分 + 关键词惩罚 → 最终评分
    ├── blocklist/
    │   ├── advertiser.go           # AdvertiserBlocklist：广告主级别黑名单（category/domain/keyword）
    │   └── platform.go             # PlatformBlocklist：平台级全局黑名单
    └── repo/                       # 黑名单持久化（可选）
```

## 快速开始

### 环境要求

- Go 1.25.3（无数据库依赖，黑名单存储于内存）

```bash
export PATH="$HOME/.goenv/versions/1.25.3/bin:$PATH"
```

### 运行服务

```bash
cd adortb-brand-safety

export PORT=8092

go run cmd/brand-safety/main.go
```

### 运行测试

```bash
go test ./... -cover -race
```

## HTTP API

### POST /v1/classify — 页面分类 + 安全评分

```json
{
  "url": "https://www.bbc.com/news/technology",
  "title": "AI breakthrough in healthcare"
}
```

**响应**：

```json
{
  "categories": ["IAB-12", "IAB-12-1"],
  "safety_score": 0.85,
  "blocked": false,
  "warnings": []
}
```

### POST /v1/check — 广告主级安全检查

在 `/v1/classify` 基础上，额外检查广告主自定义黑名单。

```json
{
  "advertiser_id": 1001,
  "url": "https://www.bet365.com",
  "title": "Sports betting odds"
}
```

**响应**（被拦截时）：

```json
{
  "allowed": false,
  "blocked_by": [{"type": "category", "value": "IAB-9", "action": "block"}],
  "categories": ["IAB-9"],
  "safety_score": 0.10
}
```

### 黑名单管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/blocklists?advertiser_id=1001` | 查询广告主黑名单 |
| POST | `/v1/blocklists` | 添加黑名单规则 |
| DELETE | `/v1/blocklists` | 删除黑名单规则 |
| GET | `/v1/blocklist/advertiser/{id}` | RESTful 查询 |
| POST | `/v1/blocklist/advertiser/{id}` | RESTful 添加 |
| GET | `/v1/blocklist/global` | 平台全局黑名单 |
| GET | `/v1/categories` | IAB 分类体系列表 |

#### 添加黑名单规则

```json
{
  "advertiser_id": 1001,
  "type": "category",   // category / domain / keyword
  "value": "IAB-9",
  "action": "block"     // block / warn
}
```

## 配置说明

| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| `PORT` | `8092` | 监听端口 |

## 评分规则

### 安全评分（0.0 ~ 1.0，越高越安全）

| 分值范围 | 含义 |
|---------|------|
| 0.9 ~ 1.0 | 高安全（科技、教育、财经等）|
| 0.7 ~ 0.9 | 中等安全（新闻、娱乐）|
| 0.3 ~ 0.7 | 需关注（日报等）|
| 0.1 ~ 0.3 | 赌博类 |
| 0.0 ~ 0.1 | 成人内容（自动拦截）|

### 关键词严重级别

| 级别 | 惩罚 | 说明 |
|------|------|------|
| `block` | -0.50 | 赌博/成人/恶意软件关键词 |
| `warn` | -0.15 | 争议性内容 |
| `flag` | -0.05 | 轻微风险提示 |

## 相关项目

| 项目 | 说明 |
|------|------|
| [adortb-adx](https://github.com/adortb/adortb-adx) | 竞价引擎，调用品牌安全检查 |
| [adortb-supply-chain](https://github.com/adortb/adortb-supply-chain) | 供应链透明度 |
| [adortb-creative-review](https://github.com/adortb/adortb-creative-review) | 素材审核（内容安全） |
