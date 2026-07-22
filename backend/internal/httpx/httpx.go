// Package httpx holds small HTTP helpers: a consistent JSON envelope, typed
// errors that map to status codes, and the middleware chain.
package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// --- errors -----------------------------------------------------------------

type Error struct {
	Status  int
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

func NewError(status int, msg string, err error) *Error {
	return &Error{Status: status, Message: msg, Err: err}
}

func BadRequest(msg string) *Error   { return &Error{Status: http.StatusBadRequest, Message: msg} }
func NotFound(msg string) *Error     { return &Error{Status: http.StatusNotFound, Message: msg} }
func Conflict(msg string) *Error     { return &Error{Status: http.StatusConflict, Message: msg} }
func Unauthorized(msg string) *Error { return &Error{Status: http.StatusUnauthorized, Message: msg} }

// ErrNotFound is returned by repositories when a row is missing.
var ErrNotFound = errors.New("data tidak ditemukan")

// --- responses --------------------------------------------------------------

type errorBody struct {
	Error string `json:"error"`
}

// JSON writes v as JSON with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Response already started; nothing useful left to do but log upstream.
		return
	}
}

// Fail maps an error to a status code + JSON body.
func Fail(w http.ResponseWriter, err error) {
	var he *Error
	switch {
	case errors.As(err, &he):
		JSON(w, he.Status, errorBody{Error: he.Message})
	case errors.Is(err, ErrNotFound):
		JSON(w, http.StatusNotFound, errorBody{Error: ErrNotFound.Error()})
	default:
		JSON(w, http.StatusInternalServerError, errorBody{Error: "terjadi kesalahan pada server"})
	}
}

// Decode reads a JSON body into dst, rejecting unknown fields so typos surface
// instead of being silently ignored.
func Decode(r *http.Request, dst any) error {
	if r.Body == nil {
		return BadRequest("body JSON wajib diisi")
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("body JSON tidak valid: " + err.Error())
	}
	return nil
}

// --- query helpers ----------------------------------------------------------

func Query(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

// QueryInt returns the query value as an int, or fallback when absent/invalid.
// The result is clamped to [min, max].
func QueryInt(r *http.Request, key string, fallback, min, max int) int {
	raw := Query(r, key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
