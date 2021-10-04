##
### Environment variables
##

# Default to debug builds, override this to disable debug builds. E.g. BUILD_DEBUG=0 make build
BUILD_DEBUG			?= 1

# Defaults to enabling the race detector (to match the behaviour of BUILD_DEBUG).
# Override this to disable the race detector. E.g. BUILD_RACE=0 make build
BUILD_RACE			?= 1

# Override this to specify additional Go build flags. Note, we append to this below.
BUILD_FLAGS			?=

BUILD_ENV 			?=

##
### Build metadata
### This is injected into the binary by `go build`. See `build/version.go` for where they end up
###
### It's also made to be overridable by the user of the Makefile. This allows
### this data to be injected in when Concordia is being built in environments where
### the git repository or history isn't available. This is frequently true in CI
### for example.
##

BUILD							?= $(shell git rev-parse --short HEAD)
BUILD_BRANCH			?= $(shell git rev-parse --abbrev-ref HEAD)
BUILD_VERSION			?= $(shell git describe --always --tags)
BUILD_TIMEUTC			?= $(shell date -u '+%Y/%m/%d-%H:%M:%S')

# Override this to tag your build. E.g. BUILD_GO_TAGS=dev make build.
#
# Note: these are Go build tags, not git tags. See the following references for more info.
#
# * https://golang.org/pkg/go/build/#hdr-Build_Constraints
# * https://www.digitalocean.com/community/tutorials/customizing-go-binaries-with-build-tags
# * https://dave.cheney.net/2013/10/12/how-to-use-conditional-compilation-with-the-go-build-tool
#
BUILD_GO_TAGS		?= -tags=jsoniter

BUILD_META	:=	-X github.com/luma/pharos/internal/meta.Build=${BUILD} \
								-X github.com/luma/pharos/internal/meta.Branch=${BUILD_BRANCH} \
								-X github.com/luma/pharos/internal/meta.Version=${BUILD_VERSION} \
								-X github.com/luma/pharos/internal/meta.BuildTimeUTC=${BUILD_TIMEUTC} \
								-X github.com/luma/pharos/internal/meta.GoTag=${BUILD_GO_TAGS}

##
### Contruct our build flags from the build environment flags and meta
##

ifneq ($(strip $(BUILD_TAGS)),)
	BUILD_FLAGS += -tags 'jsoniter $(BUILD_TAGS)'
else
	BUILD_FLAGS += -tags 'jsoniter'
endif

ifeq ($(strip $(BUILD_DEBUG)),1)
	# Build with compiler optimizations disabled, which will help debugging with dlv.
	BUILD_FLAGS += -ldflags '${BUILD_META}' -gcflags='all=-N -l'
else
	# Ensure freshness by forcing rebuilding of packages that are already up-to-date.
	BUILD_FLAGS += -a
	# Strip out debug info from the final build to reduce the final executable size
	BUILD_FLAGS += -ldflags '-w -s ${BUILD_META}'
endif

# Build with race detector enabled.
ifeq ($(strip $(BUILD_RACE)),1)
	BUILD_FLAGS += -race
else
	BUILD_ENV += CGO_ENABLED=0
endif

SRC := $(shell find . -name '*.go' | grep -v "^./vendor/")


##
### Make targets
##

build: bin/pharos

bin/pharos: $(SRC) go.mod go.sum
	GOPRIVATE=github.com/luma ${BUILD_ENV} go build -v ${BUILD_FLAGS} -o bin/pharos

# Verify that the tests pass.
.PHONY: test
test:
	go run github.com/onsi/ginkgo/ginkgo test -race -vv -r -ldflags '${BUILD_META}' -gcflags='all=-N -l' ./...

# Output test coverage stats to your terminal.
.PHONY: cover
cover: $(SRC) reports/coverage/coverage.out
	go tool cover -func=reports/coverage/coverage.out

# Generate and view a HTML report of test coverage.
.PHONY: coverHTML
coverHTML: reports/coverage/coverage.out
	go tool cover -html=reports/coverage/coverage.out

reports/coverage/coverage.out:
	mkdir -p reports/coverage/
	go run github.com/onsi/ginkgo/ginkgo -r -cover -covermode=count -outputdir=reports/coverage/ -coverprofile=coverage.out.tmp  -v -ldflags '${BUILD_META}' -gcflags='all=-N -l' ./...
	cat reports/coverage/coverage.out.tmp | grep -v -e "mode: count" > reports/coverage/coverage.out
	echo 'mode: count' | cat - reports/coverage/coverage.out > temp && mv temp reports/coverage/coverage.out
	find . -name "coverage.out.tmp" -delete

.PHONY: fmt
fmt:
	go fmt github.com/luma/pharos/...

# Verify that the code is lint free.
.PHONY: lint
lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run --issues-exit-code=1

# Verify that the code is lint free using the prod linters which are stricter but slower to run.
.PHONY: prodLint
prodLint:
	./bin/golangci-lint run --issues-exit-code=1

# Remove all build artifacts.
.PHONY: clean
clean:
	rm -fr bin reports/*
	go clean -i -x

