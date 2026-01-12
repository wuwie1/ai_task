package xormimplement

import (
	"ai_task/model"
	"strings"

	"xorm.io/xorm"
)

// 分页查询
type PaginationOrderCondition interface {
	GetPager() *model.Pager
	GetOrder() *model.Order
}

type pagerOrderCondition struct {
	DefaultOrderField string
	DefaultOrderAsc   bool
}

func WithDefaultOrderField(field string) func(*pagerOrderCondition) {
	return func(condition *pagerOrderCondition) {
		condition.DefaultOrderField = field
	}
}

// nolint
func pagerOrder(session xorm.Interface, condition PaginationOrderCondition, arr ...func(*pagerOrderCondition)) {
	pagerOrderCon := &pagerOrderCondition{}
	for _, f := range arr {
		f(pagerOrderCon)
	}
	pagination := condition.GetPager()
	if pagination != nil {
		if pagination.Limit > 0 {
			session.Limit(pagination.Limit, pagination.Offset)
		}
	}
	order := condition.GetOrder()
	orderField := pagerOrderCon.DefaultOrderField
	orderAsc := pagerOrderCon.DefaultOrderAsc
	if order != nil && !strings.EqualFold(order.OrderBy, "") {
		orderField = order.OrderBy
		orderAsc = order.OrderAsc
	}
	if orderField != "" {
		if orderAsc {
			session.OrderBy(orderField + " asc")
		} else {
			session.OrderBy(orderField + " desc")
		}
	}
}
