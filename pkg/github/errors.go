package github

import (
	"errors"
	"fmt"
	"net/http"
)

func statusToErr(statusCode int) error {
	switch statusCode {
	case http.StatusOK: //noop
	case http.StatusNoContent: //noop
	case http.StatusNotFound:
		return errors.New("not found")
	case http.StatusBadRequest:
		return errors.New("bad request")
	case http.StatusUnauthorized:
		return errors.New("unauthorized")
	case http.StatusForbidden:
		return errors.New("forbidden")
	default:
		return fmt.Errorf("unknown response code: %d", statusCode)
	}

	return nil
}
