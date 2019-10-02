package cockroach

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/jozuenoon/dunder/model"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"github.com/jozuenoon/dunder/repository"
	"github.com/oklog/ulid"
	"github.com/rs/zerolog"
)

const (
	defaultDatabase = "dunder"
	defaultUser     = "api_service"
)

type Config struct {
	Host          string
	ShouldMigrate bool
	Debug         bool
	Database      *string
	User          *string
}

func New(cfg *Config) (*ServiceImpl, error) {
	t := time.Now()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)

	db, err := newDatabase(cfg.Host, cfg.ShouldMigrate, cfg.Debug, cfg.Database, cfg.User)
	if err != nil {
		return nil, err
	}

	return &ServiceImpl{
		DB:          db,
		ulidEntropy: entropy,
	}, nil
}

func newDatabase(host string, shouldMigrate, debug bool, database, user *string) (*gorm.DB, error) {
	dbName := defaultDatabase
	if database != nil {
		dbName = *database
	}
	dbUser := defaultUser
	if user != nil {
		dbUser = *user
	}

	addr := fmt.Sprintf("postgresql://%s@%s:26257/%s?sslmode=disable", dbUser, host, dbName)
	db, err := gorm.Open("postgres", addr)
	if err != nil {
		return nil, err
	}

	db.LogMode(debug)

	if shouldMigrate {
		db.AutoMigrate(&repository.User{})
		db.AutoMigrate(&repository.Message{})
		db.AutoMigrate(&repository.Hashtag{})
		db.AutoMigrate(&repository.Trend{})
	}

	return db, nil
}

var _ repository.Service = (*ServiceImpl)(nil)

type ServiceImpl struct {
	DB          *gorm.DB
	ulidEntropy io.Reader
	log         *zerolog.Logger
}

func (s *ServiceImpl) Message(ctx context.Context, ulid string) (*repository.Message, error) {
	var resp repository.Message
	if result := s.DB.Where("ulid = ?", ulid).Preload("User").Preload("Hashtags").First(&resp); result.Error != nil {
		return nil, result.Error
	}
	return &resp, nil
}

func (s *ServiceImpl) getUserByName(db *gorm.DB, name string) (*repository.User, error) {
	user := &repository.User{
		Name: &name,
	}
	if err := db.Where("name = ?", name).FirstOrCreate(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (s *ServiceImpl) getHashtagsByText(db *gorm.DB, texts []string) ([]*repository.Hashtag, error) {
	var hashtags []*repository.Hashtag
	if err := db.Where("text IN (?)", texts).Find(&hashtags).Error; err != nil {
		return nil, err
	}
	if len(hashtags) == len(texts) {
		return hashtags, nil
	}

	found := func(txt string) bool {
		for _, h := range hashtags {
			if *h.Text == txt {
				return true
			}
		}
		return false
	}

	var missingTags []*repository.Hashtag
	for _, txt := range texts {
		if !found(txt) {
			tt := txt
			ctag := &repository.Hashtag{
				Text: &tt,
			}
			if err := db.Create(ctag).Error; err != nil {
				return nil, err
			}
			missingTags = append(missingTags, ctag)
		}
	}

	return append(hashtags, missingTags...), nil
}

func (s *ServiceImpl) CreateMessage(ctx context.Context, req *repository.CreateMessageRequest) (mID string, err error) {
	t := time.Now()
	u, err := ulid.New(ulid.Timestamp(t), s.ulidEntropy)
	if err != nil {
		return "", err
	}
	tx := s.DB.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	user, err := s.getUserByName(tx, req.UserName)
	if err != nil {
		return "", err
	}

	hashtags, err := s.getHashtagsByText(tx, req.Hashtags)
	if err != nil {
		return "", err
	}

	if err := trendsUpdate(tx, t, hashtags); err != nil {
		return "", err
	}

	us := u.String()
	message := &repository.Message{
		CreatedAt: t,
		Ulid:      &us,
		UserRef:   user.ID,
		Text:      req.Text,
		Hashtags:  hashtags,
	}

	if result := tx.Create(message); result.Error != nil {
		return "", result.Error
	}
	if result := tx.Save(message); result.Error != nil {
		return "", result.Error
	}
	tx.Commit()
	return *message.Ulid, nil
}

func (s *ServiceImpl) Messages(ctx context.Context, filter repository.Filter) ([]*repository.Message, error) {
	var resp []*repository.Message

	if filter.IsAggregateQuery() {
		return nil, fmt.Errorf("can't handle aggregate query")
	}

	query := s.DB.Limit(filter.GetLimit()).Order("ulid desc")

	switch {
	case filter.IsCursorQuery():
		query = query.Where("ulid < ?", filter.GetCursor())
	case filter.IsDateRangeQuery():
		query = query.Where("created_at > ?", filter.GetFromDate()).
			Where("created_at < ?", filter.GetToDate())
	}

	if filter.IsUserQuery() {
		var user repository.User
		s.DB.Where("name = ?", filter.GetUserName()).First(&user)
		query = query.Where("user_ref = ?", user.ID)
	}

	if filter.IsHashtagsQuery() {
		var tag repository.Hashtag
		s.DB.Where("text = ?", filter.GetHashtag()).Find(&tag)
		query = query.Joins("JOIN message_hashtags on message_hashtags.message_id = messages.id").
			Where("message_hashtags.hashtag_id = ?", tag.ID)
	}

	return resp, query.Preload("User").Preload("Hashtags").Find(&resp).Error
}

const (
	minute = 60
)

func (s *ServiceImpl) Trends(ctx context.Context, filter repository.Filter) (*repository.MessagesAggregate, error) {
	if !filter.IsAggregateQuery() {
		return nil, fmt.Errorf("expected aggregate filter query, possibly missing `aggregate` query option")
	}
	if !filter.IsDateRangeQuery() {
		return nil, fmt.Errorf("aggregated query requires valid date range")
	}

	bucketSize := int64(filter.GetAggregationPeriod().Seconds()) / minute
	fromBoundary := filter.GetFromDate().Unix() / minute
	toBoundary := filter.GetToDate().Unix() / minute

	query := s.DB.Table("trends").
		Select("floor(bucket/?) as bbucket,sum(count)", bucketSize).
		Where("bucket > ?", fromBoundary).
		Where("bucket < ?", toBoundary).
		Order("bbucket").
		Group("1")

	if filter.IsHashtagsQuery() {
		var tag repository.Hashtag
		s.DB.Where("text = ?", filter.GetHashtag()).Find(&tag)
		query = query.Where("hashtag_ref = ?", tag.ID)
	}

	rows, err := query.Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trends []*model.Trend
	for rows.Next() {
		var raw rawTrend
		if err := rows.Scan(&raw.Bucket, &raw.Count); err != nil {
			return nil, err
		}
		fromDate := time.Unix(raw.Bucket*bucketSize*minute, 0)
		toDate := fromDate.Add(time.Second * time.Duration(bucketSize*minute))
		trends = append(trends, &model.Trend{
			FromDate: fromDate,
			ToDate:   toDate,
			Count:    raw.Count,
		})
	}

	return &repository.MessagesAggregate{Trends: trends}, nil
}

type rawTrend struct {
	Bucket int64
	Count  uint
}

// trendsUpdate - creates or updates bucket_hashtag entry.
func trendsUpdate(db *gorm.DB, t time.Time, tags []*repository.Hashtag) error {
	bucket := uint(t.Unix() / 60)
	for _, tag := range tags {
		trend := &repository.Trend{
			Bucket:     bucket,
			HashtagRef: tag.ID,
			Count:      1,
		}
		if err := db.Model(&repository.Trend{}).
			Set("gorm:insert_option",
				"ON CONFLICT (bucket,hashtag_ref) DO UPDATE SET count = trends.count + 1").
			Create(trend).Error; err != nil {
			return err
		}
	}
	return nil
}
