SOURCES := $(shell find . 2>&1 | grep -E '.*\.(c|h|go)$$')
VERSION := ${shell cat ./VERSION}
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

.DEFAULT: cc-runtime
cc-runtime: $(SOURCES)
	go build -i -ldflags "-X main.commit=${COMMIT} -X main.version=${VERSION}" -o $@ .
