package mqtt

import (
	"fmt"

	"github.com/MyChaOS87/automqttion/pkg/log"
)

type Op string

const (
	OpSubscribe Op = "subscribe"
	OpPublish   Op = "publish"
	OpUnmarshal Op = "unmarshal"
)

type Error struct {
	Op      Op
	Topic   string
	Payload []byte
	Err     error
}

func (e *Error) Error() string {
	return fmt.Sprintf("mqtt %s on topic %q: %v", e.Op, e.Topic, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

type ErrorHandler func(err *Error)

func defaultErrorHandler(err *Error) {
	log.Errorf("%s", err)
}
