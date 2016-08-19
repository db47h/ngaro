GO ?= go
PKG := github.com/db47h/ngaro
SRC := vm/*.go cmd/retro/*.go asm/*.go

.PHONY: all install clean test bench qbench get-deps cover-asm cover-vm report

all: test

retro: $(SRC)
	$(GO) build $(PKG)/cmd/retro

install:
	$(GO) install $(PKG)/cmd/retro

clean:
	$(GO) clean -i $(PKG)/cmd/retro
	$(RM) retro

distclean:
	$(GO) clean -i -r $(PKG)/cmd/retro
	$(RM) retro

test:
ifeq ($(REPORT_COVERAGE),true)
	$(GO) test $(PKG)/vm -covermode=count -coverprofile=coverage0.cov
	$(GO) test $(PKG)/asm -covermode=count -coverprofile=coverage1.cov
	@echo "mode: count" > coverage.cov
	@grep -v ^mode coverage0.cov >> coverage.cov
	@grep -v ^mode coverage1.cov >> coverage.cov
	$$(go env GOPATH | awk 'BEGIN{FS=":"} {print $1}')/bin/goveralls -coverprofile=coverage.cov -service=travis-ci
	@$(RM) coverage0.cov coverage1.cov coverage.cov
else
	$(GO) test -v $(PKG)/...
endif


bench:
	$(GO) test -v $(PKG)/vm -run DONOTRUNTESTS -bench .

cover:
	$(GO) test $(PKG)/vm -covermode=count -coverprofile=coverage0.cov
	$(GO) test $(PKG)/asm -covermode=count -coverprofile=coverage1.cov
	@echo "mode: count" > coverage.cov
	@grep -v ^mode coverage0.cov >> coverage.cov
	@grep -v ^mode coverage1.cov >> coverage.cov
	$(GO) tool cover -html coverage.cov
	@$(RM) coverage0.cov coverage1.cov coverage.cov

qbench: retroImage
	/usr/bin/time -f '%Uu %Ss %er %MkB %C' ./retro <vm/testdata/core.rx >/dev/null

retroImage: retro _misc/kernel.rx _misc/meta.rx _misc/stage2.rx
	./retro -image vm/testdata/retroImage -ibits 32 -with _misc/meta.rx -with _misc/kernel.rx -o retroImage >/dev/null
	./retro -with _misc/stage2.rx >/dev/null

report: $(SRC)
	@echo "=== gocyclo ===\n"
	@gocyclo . | head
	@echo "\n\n=== misspell ===\n"
	@misspell -source go $^
	@misspell -source text README.md

get-deps:
	$(GO) get github.com/pkg/errors
	$(GO) get github.com/pkg/term
