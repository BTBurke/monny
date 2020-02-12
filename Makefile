PROTO_SRC_DIR=proto/**
PROTOC_BIN=$(shell which protoc)
SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
ifeq ($(origin .RECIPEPREFIX), undefined)
  $(error This Make does not support .RECIPEPREFIX. Please use GNU Make 4.0 or later)
endif
.RECIPEPREFIX = >

.make-proto: $(shell find . -name '*.proto' -type f)
> ${PROTOC_BIN} -I ./${PROTO_SRC_DIR} --go_out=plugins=grpc:. ./${PROTO_SRC_DIR}/*.proto
> touch $@

test:
> go test -v -cover -race ./...

dist/monny: $(shell find . -name '*.go' -type f) .make-proto
> mkdir -p $(@D)
> go build -o dist/monny ./cmd/monny/main.go

pkg/stat/kconst_gen.go:
> go generate ./...

.PHONY: test
