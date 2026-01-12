package model

// Pager 分页结构
type Pager struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// Order 排序结构
type Order struct {
	OrderAsc bool   `json:"order_asc" form:"order_asc"` // 是否升序，eg: false
	OrderBy  string `json:"order_by" form:"order_by"`   // 排序字段，eg: "id"
}

// GetDocChunksCondition 查询条件（带分页和排序）
type GetDocChunksCondition struct {
	DocID   *int64  `json:"doc_id"`
	DocIDs  []int64 `json:"doc_ids"`
	Content *string `json:"content"` // like 查询
	*Pager
	*Order
}

func (g *GetDocChunksCondition) GetPager() *Pager {
	return g.Pager
}

func (g *GetDocChunksCondition) GetOrder() *Order {
	return g.Order
}
