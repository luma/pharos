package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/luma/pharos/cmd"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	// See https://github.com/dgraph-io/badger#are-there-any-go-specific-settings-that-i-should-use
	// Setting a higher number here allows more disk I/O calls to be scheduled, hence considerably
	// improving throughput. The extra CPU overhead is almost negligible in comparison.
	runtime.GOMAXPROCS(128)

	cmd.Execute()
}
