package repository

import "time"

type Filter interface {
	IsUserQuery() bool
	IsHashtagsQuery() bool
	GetHashtag() string
	IsAggregateQuery() bool
	GetAggregationPeriod() time.Duration
	IsCursorQuery() bool
	GetCursor() string
	IsDateRangeQuery() bool
	GetFromDate() time.Time
	GetToDate() time.Time
	GetLimit() uint
	GetUserName() string
}
