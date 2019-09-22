package repository

import (
	"context"
	"time"

	"github.com/jozuenoon/dunder/model"
)

type Service interface {
	Message(ctx context.Context, ulid string) (*Message, error)

	// Returns message ulid
	CreateMessage(ctx context.Context, message *CreateMessageRequest) (string, error)
	Messages(ctx context.Context, filter Filter) ([]*Message, error)
	Trends(ctx context.Context, filter Filter) (*MessagesAggregate, error)
}

const (
	defaultLimit uint = 100
)

var _ Filter = (*FilterImpl)(nil)

type FilterImpl struct {
	model.QueryRequest
}

func (f *FilterImpl) IsUserQuery() bool {
	return len(f.Rules.UserName) > 0
}

func (f *FilterImpl) GetUserName() string {
	return f.Rules.UserName[0]
}

func (f *FilterImpl) IsHashtagsQuery() bool {
	return len(f.Rules.Hashtag) > 0
}

func (f *FilterImpl) GetHashtag() string {
	return f.Rules.Hashtag[0]
}

func (f *FilterImpl) IsAggregateQuery() bool {
	return len(f.Rules.Aggregation) > 0 && f.Rules.Aggregation[0] >= time.Minute
}

func (f *FilterImpl) GetAggregationPeriod() time.Duration {
	return f.Rules.Aggregation[0]
}

func (f *FilterImpl) IsCursorQuery() bool {
	return len(f.Cursor) > 0
}

func (f *FilterImpl) GetCursor() string {
	return f.Cursor[0]
}

func (f *FilterImpl) IsDateRangeQuery() bool {
	return len(f.FromDate) > 0 && len(f.ToDate) > 0 && f.GetFromDate().Before(f.GetToDate())
}

func (f *FilterImpl) GetFromDate() time.Time {
	return f.FromDate[0].Truncate(time.Minute)
}

func (f *FilterImpl) GetToDate() time.Time {
	return f.ToDate[0].Truncate(time.Minute)
}

func (f *FilterImpl) GetLimit() uint {
	if len(f.Limit) == 0 {
		return defaultLimit
	}
	return f.Limit[0]
}
