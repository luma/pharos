// +build tools

package tools

// Package tools tracks dependencies on binaries not otherwise referenced in the codebase.
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module
import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/onsi/ginkgo/ginkgo"
)
