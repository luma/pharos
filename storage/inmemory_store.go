package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type InmemoryStore struct {
	values []byte

	mu          sync.Mutex
	updateChans []chan *Update

	// stop willl be closed when Close() is called
	stop chan struct{}
}

func NewInmemoryStore() *InmemoryStore {
	return &InmemoryStore{
		values:      []byte(""),
		stop:        make(chan struct{}),
		updateChans: make([]chan *Update, 0),
	}
}

func (i *InmemoryStore) Close() error {
	if i.isRunning() {
		close(i.stop)
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	for _, updateChan := range i.updateChans {
		close(updateChan)
	}

	return nil
}

func (i *InmemoryStore) Set(ctx context.Context, key []byte, value interface{}) (err error) {
	i.values, err = sjson.SetBytes(i.values, string(key), value)
	if err != nil {
		return err
	}

	if i.isRunning() {
		i.mu.Lock()

		for _, updateChan := range i.updateChans {
			updateChan <- &Update{
				Key:   key,
				Value: []byte(gjson.GetBytes(i.values, string(key)).Raw),
			}
		}

		i.mu.Unlock()
	}

	fmt.Printf("WUT %s = %v\n%s\n", string(key), value, string(i.values))

	return nil
}

func (i *InmemoryStore) Get(ctx context.Context, key []byte) ([]byte, error) {
	result := gjson.GetBytes(i.values, string(key))

	if result.Index == 0 {
		return []byte(result.Raw), nil
	}

	return i.values[result.Index : result.Index+len(result.Raw)], nil
}

func (i *InmemoryStore) ListenToUpdates() <-chan *Update {
	i.mu.Lock()
	defer i.mu.Unlock()

	updateChan := make(chan *Update, 255)
	i.updateChans = append(i.updateChans, updateChan)

	return updateChan
}

func (i *InmemoryStore) Restore(values []byte) error {
	i.values = values
	return nil
}

func (i *InmemoryStore) Backup() ([]byte, error) {
	if len(i.values) == 0 {
		return []byte("{}"), nil
	}

	return i.values, nil
}

// isRunning returns true if Close has not been called
func (i *InmemoryStore) isRunning() bool {
	select {
	case <-i.stop:
		return false

	default:
		return true
	}
}

var _ Store = (*InmemoryStore)(nil)
