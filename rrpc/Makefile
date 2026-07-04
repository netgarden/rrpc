ROOT         := $(abspath .)
RRPC_GEN_DIR := $(abspath ../rrpc-gen)
RRPC_GEN     := RRPC_DEVEL=true go run ./cmd/rrpc-gen/

.PHONY: gen test

gen:
	@echo "  go  testmodule"
	@cd $(RRPC_GEN_DIR) && $(RRPC_GEN) \
		--source $(ROOT)/internal/testmodule/def/module.rrpc \
		--target $(ROOT)/internal/testmodule \
		--type go

test: gen
	@go test ./...