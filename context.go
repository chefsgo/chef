package chef

import (
	"time"
)

type (
	Context interface {
		Zone() *time.Location
	}
	emptyContext struct{}
)

func newContext() Context {
	return &emptyContext{}
}

func (c *emptyContext) Zone() *time.Location {
	return time.Local
}
