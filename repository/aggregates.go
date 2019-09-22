package repository

import (
	"github.com/jozuenoon/dunder/model"
)

type MessagesAggregate struct {
	Trends []*model.Trend
}
