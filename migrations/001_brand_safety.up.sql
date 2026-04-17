-- IAB Content Taxonomy 3.0 类别树
CREATE TABLE IF NOT EXISTS content_categories (
    id         VARCHAR(20)  PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    parent_id  VARCHAR(20)  REFERENCES content_categories(id),
    tier       INT          NOT NULL,
    created_at TIMESTAMPTZ  DEFAULT NOW()
);

-- URL 分类缓存
CREATE TABLE IF NOT EXISTS url_classifications (
    id             BIGSERIAL    PRIMARY KEY,
    domain         VARCHAR(255) NOT NULL,
    url_pattern    VARCHAR(512),
    categories     VARCHAR(20)[] NOT NULL DEFAULT '{}',
    safety_score   DECIMAL(3,2) NOT NULL DEFAULT 1.00,
    classified_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_url_domain ON url_classifications(domain);

-- 广告主黑名单
CREATE TABLE IF NOT EXISTS advertiser_blocklists (
    id             BIGSERIAL    PRIMARY KEY,
    advertiser_id  BIGINT       NOT NULL,
    type           VARCHAR(20)  NOT NULL CHECK (type IN ('category','domain','keyword')),
    value          VARCHAR(512) NOT NULL,
    action         VARCHAR(20)  NOT NULL DEFAULT 'block' CHECK (action IN ('block','warn')),
    created_at     TIMESTAMPTZ  DEFAULT NOW(),
    UNIQUE (advertiser_id, type, value)
);
CREATE INDEX IF NOT EXISTS idx_abl_advertiser ON advertiser_blocklists(advertiser_id);

-- 关键词库
CREATE TABLE IF NOT EXISTS content_keywords (
    id          BIGSERIAL    PRIMARY KEY,
    keyword     VARCHAR(128) NOT NULL,
    category_id VARCHAR(20)  REFERENCES content_categories(id),
    severity    VARCHAR(20)  NOT NULL DEFAULT 'block' CHECK (severity IN ('block','warn','flag')),
    language    VARCHAR(10)  NOT NULL DEFAULT 'en',
    UNIQUE (keyword, language)
);
