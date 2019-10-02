package transport

import (
	"encoding/base64"
	"strings"
)

// userFromBearerToken is simplistic authentication example, read base64 encoded user name in bearer token.
// Example "Authorization: Bearer <base64 encoded username>"
func (h *Http) userFromBearerToken(authHeader string) (string, error) {
	sp := strings.SplitN(authHeader, " ", 2)
	if len(sp) != 2 {
		h.log.Debug().Msgf("invalid length of bearer token: %s", authHeader)
		return "", unauthorized
	}
	if sp[0] != "Bearer" {
		h.log.Debug().Msgf("got: %s, expected: Bearer", sp[0])
		return "", unauthorized
	}
	u, err := base64.StdEncoding.DecodeString(sp[1])
	if err != nil {
		h.log.Debug().Msgf("failed to decode user: %s", sp[1])
		return "", unauthorized
	}
	return string(u), nil
}