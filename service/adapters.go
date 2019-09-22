package service

import (
	"github.com/jozuenoon/dunder/model"
	"github.com/jozuenoon/dunder/repository"
)

func RepositoryMessagesAdapter(in []*repository.Message) []*model.Message {
	var out []*model.Message
	for _, m := range in {
		out = append(out, &model.Message{
			ID: *m.Ulid,
			User: model.User{
				ID:   m.User.ID,
				Name: *m.User.Name,
			},
			Text:     m.Text,
			Hashtags: extractTagText(m.Hashtags),
		})
	}
	return out
}

func extractTagText(tags []*repository.Hashtag) []string {
	var t []string
	for _, h := range tags {
		tt := *h.Text
		t = append(t, tt)
	}
	return t
}
