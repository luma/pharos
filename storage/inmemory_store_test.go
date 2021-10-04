package storage_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/luma/pharos/storage"
)

var _ = Describe("storage / InmemoryStore", func() {
	Describe("Close()", func() {
		It("does not panic when closed twice", func() {
			store := storage.NewInmemoryStore()
			defer store.Close()

			Expect(func() { store.Close() }).NotTo(Panic())
			Expect(func() { store.Close() }).NotTo(Panic())
		})
	})

	It("an empty inmemory store equals {}", func() {
		store := storage.NewInmemoryStore()
		defer store.Close()

		value, err := store.Backup()
		Expect(err).To(Succeed())
		Expect(string(value)).To(Equal(`{}`))
	})

	Describe("Set() / Get()", func() {
		It("can read a key that is written", func() {
			store := storage.NewInmemoryStore()
			defer store.Close()

			err := store.Set(context.Background(), []byte("foo"), "bar")
			Expect(err).To(Succeed())

			Expect(store.Get(context.Background(), []byte("foo"))).To(Equal([]byte(`"bar"`)))

			value, err := store.Backup()
			Expect(err).To(Succeed())
			Expect(string(value)).To(Equal(`{"foo":"bar"}`))
		})

		It("sends on the update channel when values are set", func() {
			store := storage.NewInmemoryStore()
			defer store.Close()

			updateChan := store.ListenToUpdates()
			err := store.Set(context.Background(), []byte("foo"), "bar")
			Expect(err).To(Succeed())

			update, ok := <-updateChan
			Expect(ok).To(BeTrue())
			Expect(update).To(Equal(&storage.Update{
				Key:   []byte("foo"),
				Value: []byte(`"bar"`),
			}))
		})
	})
})
