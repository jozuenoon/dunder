package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/araddon/dateparse"

	"github.com/jozuenoon/dunder/model"

	"github.com/rs/zerolog"

	"github.com/jozuenoon/dunder/service"
)

func NewHttp(dunder service.Dunder, search service.DunderSearch, log *zerolog.Logger) *Http {
	return &Http{
		dunder: dunder,
		search: search,
		log:    log,
	}
}

type Http struct {
	dunder service.Dunder
	search service.DunderSearch
	log    *zerolog.Logger
}

func (h *Http) CreateMessage(w http.ResponseWriter, r *http.Request) {
	var req model.CreateMessageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		h.writeError(err, w)
		return
	}

	user := r.Header.Get("user")
	if user == "" {
		h.writeError(fmt.Errorf("expected `User` header"), w)
		return
	}
	ctx := context.Background()
	resp, err := h.dunder.CreateMessage(ctx, user, &req)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(resp, w)
}

func (h Http) MessageQuery(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.writeError(err, w)
	}
	// Unary query
	mulid := r.Form.Get("ulid")
	if mulid != "" {
		h.unaryMessageQuery(mulid, w)
		return
	}

	q, err := parseQuery(r.Form)
	if err != nil {
		h.writeError(err, w)
		return
	}

	resp, err := h.search.Messages(context.Background(), q)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(resp, w)
}

func (h Http) unaryMessageQuery(mulid string, w http.ResponseWriter) {
	resp, err := h.dunder.GetMessage(context.Background(), &model.GetMessageRequest{ID: mulid})
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(resp, w)
}

func (h Http) Trends(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.writeError(err, w)
	}

	q, err := parseQuery(r.Form)
	if err != nil {
		h.writeError(err, w)
		return
	}

	resp, err := h.search.Trends(context.Background(), q)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(resp, w)
}

func (h *Http) writeError(err error, w http.ResponseWriter) {
	msg := &Response{
		Error: err.Error(),
	}
	var buf bytes.Buffer
	err1 := json.NewEncoder(&buf).Encode(msg)
	if err1 != nil {
		h.log.Error().Err(err1).Msg("failed to marshall error")
	}
	_, err = w.Write(buf.Bytes())
	if err != nil {
		h.log.Error().Err(err).Msg("writeResponse: failed to write")
	}
	w.WriteHeader(http.StatusBadRequest)
}

func (h *Http) writeResponse(resp interface{}, w http.ResponseWriter) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&Response{
		Data: resp,
	}); err != nil {
		h.writeError(err, w)
		return
	}
	_, err := w.Write(buf.Bytes())
	if err != nil {
		h.log.Error().Err(err).Msg("writeResponse: failed to write")
	}
}

//go:generate gomodifytags -file http.go -struct Response -add-tags json -add-options json=omitempty -w
type Response struct {
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

func parseQuery(vals url.Values) (*model.QueryRequest, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(vals); err != nil {
		return nil, err
	}
	fmt.Println(vals.Encode())
	fmt.Println(buf.String())
	var flat flatQuery
	if err := json.NewDecoder(&buf).Decode(&flat); err != nil {
		return nil, err
	}
	return &model.QueryRequest{
		FromDate: toTime(flat.FromDate),
		ToDate:   toTime(flat.ToDate),
		Limit:    flat.Limit,
		Cursor:   flat.Cursor,
		Rules: model.QueryRules{
			UserName:    flat.UserName,
			Hashtag:     flat.Hashtag,
			Aggregation: toDuration(flat.Aggregation),
		},
	}, nil
}

func toTime(in []DateTime) []time.Time {
	var tt []time.Time
	for _, t := range in {
		tt = append(tt, time.Time(t))
	}
	return tt
}

func toDuration(in []Duration) []time.Duration {
	var tt []time.Duration
	for _, t := range in {
		tt = append(tt, time.Duration(t))
	}
	return tt
}

type DateTime time.Time

func (d *DateTime) UnmarshalJSON(b []byte) error {
	s := strings.Replace(string(b), "\"", "", -1)
	t, err := dateparse.ParseAny(s)
	*d = DateTime(t)
	return err
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type flatQuery struct {
	FromDate    []DateTime `json:"from_date,omitempty"`
	ToDate      []DateTime `json:"to_date,omitempty"`
	Limit       []uint     `json:"limit,omitempty"`
	Cursor      []string   `json:"cursor,omitempty"`
	UserName    []string   `json:"user_name,omitempty"`
	Hashtag     []string   `json:"hashtag,omitempty"`
	Aggregation []Duration `json:"aggregation,omitempty"`
}
