package service

import (
	"context"

	"github.com/jozuenoon/dunder/model"

	"github.com/rs/zerolog"

	"github.com/jozuenoon/dunder/repository"
)

type Dunder interface {
	CreateMessage(context.Context, string, *model.CreateMessageRequest) (*model.CreateMessageResponse, error)
	GetMessage(context.Context, *model.GetMessageRequest) (*model.GetMessageResponse, error)
}

var _ Dunder = (*DunderImpl)(nil)

func NewDunder(repo repository.Service, log *zerolog.Logger) *DunderImpl {
	return &DunderImpl{
		repo: repo,
		log:  log,
	}
}

type DunderImpl struct {
	repo repository.Service
	log  *zerolog.Logger
}

func (d *DunderImpl) CreateMessage(ctx context.Context, userName string, req *model.CreateMessageRequest) (*model.CreateMessageResponse, error) {
	msgID, err := d.repo.CreateMessage(ctx, &repository.CreateMessageRequest{
		UserName: userName,
		Text:     req.Text,
		Hashtags: req.Hashtags,
	})
	if err != nil {
		return nil, err
	}
	return &model.CreateMessageResponse{
		ID: msgID,
	}, nil
}

func (d *DunderImpl) GetMessage(ctx context.Context, req *model.GetMessageRequest) (*model.GetMessageResponse, error) {
	msg, err := d.repo.Message(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &model.GetMessageResponse{model.Message{
		ID: *msg.Ulid,
		User: model.User{
			ID:   msg.User.ID,
			Name: *msg.User.Name,
		},
		Text:     msg.Text,
		Hashtags: extractTagText(msg.Hashtags),
	}}, nil
}
