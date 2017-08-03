PB_PACKAGE=pb
PROTO_SRC_DIR=proto/monitor
PROTOC_BIN=$(GOPATH)/bin/protoc

proto:
	$(PROTOC_BIN) -I ./$(PROTO_SRC_DIR) --go_out=plugins=grpc,import_path=$(PB_PACKAGE):$(PB_PACKAGE) ./$(PROTO_SRC_DIR)/*.proto
