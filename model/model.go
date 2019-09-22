package model

import (
	"time"
)

//go:generate gomodifytags -file model.go -struct User -add-tags json -add-options json=omitempty -w
type User struct {
	ID          uint   `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	ScreenName  string `json:"screen_name,omitempty"`
	Location    string `json:"location,omitempty"`
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

//go:generate gomodifytags -file model.go -struct CreateMessageRequest -add-tags json -add-options json=omitempty -w
type CreateMessageRequest struct {
	Text     string   `json:"text,omitempty"`
	Hashtags []string `json:"hashtags,omitempty"`
}

//go:generate gomodifytags -file model.go -struct CreateMessageResponse -add-tags json -add-options json=omitempty -w
type CreateMessageResponse struct {
	ID string `json:"id,omitempty"`
}

//go:generate gomodifytags -file model.go -struct GetMessageRequest -add-tags json -add-options json=omitempty -w
type GetMessageRequest struct {
	ID string `json:"id,omitempty"`
}

//go:generate gomodifytags -file model.go -struct GetMessageResponse -add-tags json -add-options json=omitempty -w
type GetMessageResponse struct {
	Message `json:"message,omitempty"`
}

//go:generate gomodifytags -file model.go -struct Message -add-tags json -add-options json=omitempty -w
type Message struct {
	ID        string    `json:"id,omitempty"`
	User      User      `json:"user,omitempty"`
	Text      string    `json:"text,omitempty"`
	Hashtags  []string  `json:"hashtags,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

//go:generate gomodifytags -file model.go -struct QueryRequest -add-tags json -add-options json=omitempty -w
type QueryRequest struct {
	FromDate []time.Time `json:"from_date,omitempty"`
	ToDate   []time.Time `json:"to_date,omitempty"`
	Limit    []uint      `json:"limit,omitempty"`
	Cursor   []string    `json:"cursor,omitempty"`
	Rules    QueryRules  `json:"rules,omitempty"`
}

//go:generate gomodifytags -file model.go -struct QueryRules -add-tags json -add-options json=omitempty -w
type QueryRules struct {
	UserName    []string        `json:"user_name,omitempty"`
	Hashtag     []string        `json:"hashtag,omitempty"`
	Aggregation []time.Duration `json:"aggregation,omitempty"`
}

//go:generate gomodifytags -file model.go -struct QueryResponse -add-tags json -add-options json=omitempty -w
type QueryResponse struct {
	Messages   []*Message `json:"messages,omitempty"`
	Trends     []*Trend   `json:"trends,omitempty"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

//go:generate gomodifytags -file model.go -struct Trend -add-tags json -add-options json=omitempty -w
type Trend struct {
	FromDate time.Time `json:"from_date,omitempty"`
	ToDate   time.Time `json:"to_date,omitempty"`
	Count    uint      `json:"count,omitempty"`
}
