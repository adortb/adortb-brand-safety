// Package classifier 实现 IAB Content Taxonomy 3.0 分类。
package classifier

// Category 表示 IAB Content Taxonomy 3.0 中的一个分类节点。
type Category struct {
	ID       string
	Name     string
	ParentID string // 空字符串表示顶级
	Tier     int
}

// IAB Tier-1 顶级类别 ID 常量
const (
	CatArts          = "IAB-1"
	CatAutomotive    = "IAB-2"
	CatBusiness      = "IAB-3"
	CatCareers       = "IAB-4"
	CatEducation     = "IAB-5"
	CatFamilyParent  = "IAB-6"
	CatHealthFitness = "IAB-7"
	CatFoodDrink     = "IAB-8"
	CatHobbies       = "IAB-9"
	CatHomeGarden    = "IAB-10"
	CatLaw           = "IAB-11"
	CatNews          = "IAB-12"
	CatPersonalFin   = "IAB-13"
	CatSociety       = "IAB-14"
	CatScience       = "IAB-15"
	CatPets          = "IAB-16"
	CatSports        = "IAB-17"
	CatStyle         = "IAB-18"
	CatTech          = "IAB-19"
	CatTravel        = "IAB-20"
	CatRealEstate    = "IAB-21"
	CatShopping      = "IAB-22"
	CatReligion      = "IAB-23"
	CatUncategorized = "IAB-24"
	CatNonStandard   = "IAB-25"
	// 敏感类别
	CatIllegal       = "IAB-26"
	CatAdult         = "IAB-25-3"
	CatGambling      = "IAB-9-7"
	CatWeapons       = "IAB-26-4"
	CatDrugs         = "IAB-26-2"
	CatHate          = "IAB-26-1"
)

// SensitiveCategories 是平台默认屏蔽的高风险类别集合。
var SensitiveCategories = map[string]bool{
	CatAdult:    true,
	CatGambling: true,
	CatWeapons:  true,
	CatDrugs:    true,
	CatHate:     true,
	CatIllegal:  true,
}

// DefaultTaxonomy 返回 IAB Tier-1 及常用 Tier-2 类别列表。
func DefaultTaxonomy() []Category {
	return []Category{
		{ID: CatArts, Name: "Arts & Entertainment", Tier: 1},
		{ID: "IAB-1-1", Name: "Books & Literature", ParentID: CatArts, Tier: 2},
		{ID: "IAB-1-2", Name: "Celebrity Fan/Gossip", ParentID: CatArts, Tier: 2},
		{ID: "IAB-1-5", Name: "Movies", ParentID: CatArts, Tier: 2},
		{ID: "IAB-1-6", Name: "Music", ParentID: CatArts, Tier: 2},
		{ID: "IAB-1-7", Name: "Television", ParentID: CatArts, Tier: 2},
		{ID: CatAutomotive, Name: "Automotive", Tier: 1},
		{ID: CatBusiness, Name: "Business", Tier: 1},
		{ID: "IAB-3-1", Name: "Advertising", ParentID: CatBusiness, Tier: 2},
		{ID: "IAB-3-5", Name: "Financial News", ParentID: CatBusiness, Tier: 2},
		{ID: CatCareers, Name: "Careers", Tier: 1},
		{ID: CatEducation, Name: "Education", Tier: 1},
		{ID: CatFamilyParent, Name: "Family & Parenting", Tier: 1},
		{ID: CatHealthFitness, Name: "Health & Fitness", Tier: 1},
		{ID: CatFoodDrink, Name: "Food & Drink", Tier: 1},
		{ID: CatHobbies, Name: "Hobbies & Interests", Tier: 1},
		{ID: CatGambling, Name: "Gambling", ParentID: CatHobbies, Tier: 2},
		{ID: CatHomeGarden, Name: "Home & Garden", Tier: 1},
		{ID: CatLaw, Name: "Law, Gov't & Politics", Tier: 1},
		{ID: CatNews, Name: "News / Weather / Information", Tier: 1},
		{ID: "IAB-12-1", Name: "International News", ParentID: CatNews, Tier: 2},
		{ID: "IAB-12-2", Name: "National News", ParentID: CatNews, Tier: 2},
		{ID: "IAB-12-3", Name: "Local News", ParentID: CatNews, Tier: 2},
		{ID: CatPersonalFin, Name: "Personal Finance", Tier: 1},
		{ID: CatSociety, Name: "Society", Tier: 1},
		{ID: CatScience, Name: "Science", Tier: 1},
		{ID: CatPets, Name: "Pets", Tier: 1},
		{ID: CatSports, Name: "Sports", Tier: 1},
		{ID: CatStyle, Name: "Style & Fashion", Tier: 1},
		{ID: CatTech, Name: "Technology & Computing", Tier: 1},
		{ID: CatTravel, Name: "Travel", Tier: 1},
		{ID: CatRealEstate, Name: "Real Estate", Tier: 1},
		{ID: CatShopping, Name: "Shopping", Tier: 1},
		{ID: CatReligion, Name: "Religion & Spirituality", Tier: 1},
		{ID: CatUncategorized, Name: "Uncategorized", Tier: 1},
		{ID: CatNonStandard, Name: "Non-Standard Content", Tier: 1},
		{ID: CatAdult, Name: "Adult Content", ParentID: CatNonStandard, Tier: 2},
		{ID: CatIllegal, Name: "Illegal Content", Tier: 1},
		{ID: CatHate, Name: "Hate Content", ParentID: CatIllegal, Tier: 2},
		{ID: CatDrugs, Name: "Drug-Related Content", ParentID: CatIllegal, Tier: 2},
		{ID: CatWeapons, Name: "Weapons", ParentID: CatIllegal, Tier: 2},
	}
}

// TaxonomyIndex 提供 O(1) 查找的类别索引。
type TaxonomyIndex struct {
	byID map[string]Category
}

// NewTaxonomyIndex 构建索引。
func NewTaxonomyIndex(cats []Category) *TaxonomyIndex {
	idx := &TaxonomyIndex{byID: make(map[string]Category, len(cats))}
	for _, c := range cats {
		idx.byID[c.ID] = c
	}
	return idx
}

// Get 按 ID 查找类别，found=false 表示不存在。
func (t *TaxonomyIndex) Get(id string) (Category, bool) {
	c, ok := t.byID[id]
	return c, ok
}

// IsSensitive 判断给定类别 ID 是否属于敏感类别。
func (t *TaxonomyIndex) IsSensitive(id string) bool {
	return SensitiveCategories[id]
}
