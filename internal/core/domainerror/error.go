package domainerror

import "errors"

type Kind string

const (
	KindValidation Kind = "validation"
	KindConflict   Kind = "conflict"
	KindNotFound   Kind = "not_found"
	KindGone       Kind = "gone"
	KindBlocked    Kind = "blocked"
	KindInternal   Kind = "internal"
)

type Detail struct {
	Code    string
	Kind    Kind
	Message string
}

type detailedError interface {
	DomainErrorDetail() Detail
}

type Error struct {
	code    string
	kind    Kind
	message string
}

func New(code string, kind Kind, message string) *Error {
	return &Error{code: code, kind: kind, message: message}
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) DomainErrorDetail() Detail {
	return Detail{Code: e.code, Kind: e.kind, Message: e.message}
}

func DetailOf(err error) (Detail, bool) {
	var target detailedError
	if errors.As(err, &target) {
		return target.DomainErrorDetail(), true
	}

	return Detail{}, false
}
