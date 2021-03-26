package socketio

import "github.com/davidmz/debug-log"

// Option allows to configure Connection on create.
type Option interface {
	apply(*Connection)
}

type optionFn func(*Connection)

func (f optionFn) apply(l *Connection) { f(l) }

// Options is a list of Option. It satisfies the Option interface itself.
type Options []Option

func (o Options) apply(l *Connection) {
	for _, opt := range o {
		opt.apply(l)
	}
}

// WithLogger allows to configure log of the connection.
func WithLogger(log debug.Logger) Option { return optionFn(func(i *Connection) { i.log = log }) }
