package shutdown

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Func func(context.Context) error

// Group runs registered shutdown functions in reverse order with a shared
// context and reports all failures.
func Group(ctx context.Context, funcs []Func) error {
	var errs []error
	for i := len(funcs) - 1; i >= 0; i-- {
		if err := funcs[i](ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown step %d: %w", i, err))
		}
	}
	return errors.Join(errs...)
}

// Component coordinates a bounded background goroutine lifecycle.
type Component struct {
	name    string
	start   func(context.Context) error
	stop    func(context.Context) error
	mu      sync.Mutex
	started bool
}

func New(name string, start, stop func(context.Context) error) *Component {
	return &Component{name: name, start: start, stop: stop}
}

func (c *Component) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started {
		return fmt.Errorf("%s already started", c.name)
	}
	if err := c.start(ctx); err != nil {
		return fmt.Errorf("start %s: %w", c.name, err)
	}
	c.started = true
	return nil
}

func (c *Component) Stop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started {
		return nil
	}
	c.started = false
	if err := c.stop(ctx); err != nil {
		return fmt.Errorf("stop %s: %w", c.name, err)
	}
	return nil
}
