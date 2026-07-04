MODULES      := auth test files echo
ROOT         := $(abspath .)
RRPC_GEN_DIR := $(abspath ../gen)
RRPC_GEN     := RRPC_DEVEL=true go run ./cmd/rrpc-gen/

.PHONY: all gen gen-go gen-ts

all: gen

gen: gen-go gen-ts

gen-go:
	@for mod in $(MODULES); do \
		echo "  go  $$mod"; \
		cd $(RRPC_GEN_DIR) && $(RRPC_GEN) \
			--source $(ROOT)/modules/$$mod/def/module.rrpc \
			--target $(ROOT)/modules/$$mod \
			--type go || exit 1; \
	done

gen-ts:
	@for mod in $(MODULES); do \
		echo "  ts  $$mod"; \
		cd $(RRPC_GEN_DIR) && $(RRPC_GEN) \
			--source $(ROOT)/modules/$$mod/def/module.rrpc \
			--target $(ROOT)/web/src/lib \
			--type ts || exit 1; \
	done