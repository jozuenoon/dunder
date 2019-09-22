package repository

import (
	// Import GORM-related packages.

	"time"

	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// User has and belongs to many languages, use `user_languages` as join table
type User struct {
	ID          uint `gorm:"primary_key"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `sql:"index"`
	Name        *string    `gorm:"unique;not null"`
	ScreenName  string
	Location    string
	URL         string
	Description string
}

// Trend provides minute granularity statistics.
type Trend struct {
	Bucket     uint    `gorm:"primary_key"`
	Hashtag    Hashtag `gorm:"foreignkey:HashtagRef"`
	HashtagRef uint    `gorm:"primary_key"`
	Count      uint
}

type Message struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
	Ulid      *string    `gorm:"unique;not null"`
	User      User       `gorm:"foreignkey:UserRef;association_autoupdate:false"`
	UserRef   uint
	Text      string
	Hashtags  []*Hashtag `gorm:"many2many:message_hashtags;association_autoupdate:false"`
}

type Hashtag struct {
	ID        uint `gorm:"primary_key"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `sql:"index"`
	Text      *string    `gorm:"unique;not null"`
	Messages  []*Message `gorm:"many2many:message_hashtags"`
}

type CreateMessageRequest struct {
	UserName string
	Text     string
	Hashtags []string
}
