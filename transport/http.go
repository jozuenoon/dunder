package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
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

	token := r.Header.Get("authorization")
	if token == "" {
		h.writeError(unauthorized, w)
		return
	}
	user, err := h.userFromBearerToken(token)
	if err != nil {
		h.writeError(err, w)
		return
	}
	ctx := context.Background()
	resp, err := h.dunder.CreateMessage(ctx, user, &req)
	if err != nil {
		h.writeError(err, w)
		return
	}
	buf, err := h.prepareResponse(resp)
	if err != nil {
		h.writeError(err, w)
		return
	}
	w.WriteHeader(http.StatusCreated)
	h.writeResponse(buf, w)
}

func (h Http) MessageQuery(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.writeError(err, w)
	}

	// Unary query from path
	vars := mux.Vars(r)
	if ulid, ok := vars["ulid"]; ok {
		h.unaryMessageQuery(ulid, w)
		return
	}

	// Unary query parameters
	if mulid := r.Form.Get("ulid"); mulid != "" {
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
	buf, err := h.prepareResponse(resp)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(buf, w)
}

func (h Http) unaryMessageQuery(mulid string, w http.ResponseWriter) {
	resp, err := h.dunder.GetMessage(context.Background(), &model.GetMessageRequest{ID: mulid})
	if err != nil {
		h.writeError(err, w)
		return
	}
	buf, err := h.prepareResponse(resp)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(buf, w)
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
	buf, err := h.prepareResponse(resp)
	if err != nil {
		h.writeError(err, w)
		return
	}
	h.writeResponse(buf, w)
}


func (h *Http) prepareResponse(resp interface{}) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&Response{
		Data: resp,
	}); err != nil {
		return nil, err
	}
	return &buf, nil
}

func (h *Http) writeResponse(buf *bytes.Buffer, w http.ResponseWriter) {
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
