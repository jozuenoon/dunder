package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"net/http"
)

var (
	unauthorized = fmt.Errorf("unauthorized")
)

func (h *Http) writeError(err error, w http.ResponseWriter) {
	switch err {
	case unauthorized:
		w.WriteHeader(http.StatusUnauthorized)
	case gorm.ErrRecordNotFound:
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
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
}