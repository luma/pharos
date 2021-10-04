package storage

import "context"

type Store interface {
	Set(ctx context.Context, key []byte, value interface{}) error
	Get(ctx context.Context, key []byte) ([]byte, error)

	Restore(values []byte) error
	Backup() ([]byte, error)

	ListenToUpdates() <-chan *Update

	Close() error
}
