# https://github.com/amitsaha/golang-packaging-demo
# Binary paths which we will use for:
# 1. Running the commands
# 2. As Makefile targets to automatically install them
GOPATH := $(shell go env GOPATH)
GODEP_BIN := $(GOPATH)/bin/dep
GOLINT_BIN := $(GOPATH)/bin/golint

version := $(shell cat VERSION)-$(shell git rev-parse --short HEAD)
build_date := $(shell date +%FT%T%z)

packages = $$(go list ./... | egrep -v '/vendor/')
files = $$(find . -name '*.go' | egrep -v '/vendor/')

BINARY_NAME = 'prometheus-artifactory-tests-exporter'

.phony: all
all: lint vet test build build-deb

$(GODEP_BIN):
	go get -u github.com/golang/dep/cmd/dep

gopkg.toml: $(GODEP_BIN)
	$(GODEP_BIN) init

version:
	@ echo $(VERSION)

build:          ## Build the binary
build: vendor
	go build -o $(BINARY_NAME) -ldflags "-X main.Version=$(version) -X main.BuildDate=$(build_date)"

build-deb:      ## Build DEB package (needs other tools)
	exec ./build-deb-docker.sh

build-deb-trusty:      ## Build DEB package for Ubuntu Trusty
	exec ./build-deb-docker.sh trusty

test: vendor
	go test -race $(packages)

vet:            ## Run go vet
vet: vendor
	go tool vet -printfuncs=Debug,Debugf,Debugln,Info,Infof,Infoln,Error,Errorf,Errorln $(files)

$(GOLINT_BIN):
	go get -u golang.org/x/lint/golint

lint:           ## Run go lint
lint: vendor $(GOLINT_BIN)
	$(GOLINT_BIN) -set_exit_status $(packages)

clean:
	rm -rf deb-packages

help:           ## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
