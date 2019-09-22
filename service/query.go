package service

import (
	"context"

	"github.com/jozuenoon/dunder/model"

	"github.com/jozuenoon/dunder/repository"
	"github.com/rs/zerolog"
)

type DunderSearch interface {
	Messages(context.Context, *model.QueryRequest) (*model.QueryResponse, error)
	Trends(context.Context, *model.QueryRequest) (*model.QueryResponse, error)
}

var _ DunderSearch = (*DunderSearchImpl)(nil)

func NewDunderSearch(repo repository.Service, log *zerolog.Logger) *DunderSearchImpl {
	return &DunderSearchImpl{
		repo: repo,
		log:  log,
	}
}

type DunderSearchImpl struct {
	repo repository.Service
	log  *zerolog.Logger
}

func (d *DunderSearchImpl) Messages(ctx context.Context, req *model.QueryRequest) (*model.QueryResponse, error) {
	msgs, err := d.repo.Messages(ctx, &repository.FilterImpl{QueryRequest: *req})
	if err != nil {
		return nil, err
	}
	smsgs := RepositoryMessagesAdapter(msgs)

	nextCursor := func() string {
		if len(smsgs) == 0 {
			return ""
		}
		return smsgs[len(smsgs)-1].ID
	}

	return &model.QueryResponse{
		Messages:   smsgs,
		NextCursor: nextCursor(),
	}, nil
}

func (d *DunderSearchImpl) Trends(ctx context.Context, req *model.QueryRequest) (*model.QueryResponse, error) {
	trends, err := d.repo.Trends(ctx, &repository.FilterImpl{QueryRequest: *req})
	if err != nil {
		return nil, err
	}
	return &model.QueryResponse{
		Trends:     trends.Trends,
		NextCursor: "",
	}, nil
}
