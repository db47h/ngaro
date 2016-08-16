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
	$(GO) test -v $(PKG)/...

bench:
	$(GO) test -v $(PKG)/vm -run DONOTRUNTESTS -bench .

cover-asm:
	$(GO) test $(PKG)/asm -coverprofile=cover.out && go tool cover -html=cover.out

cover-vm:
	$(GO) test $(PKG)/vm -coverprofile=cover.out && go tool cover -html=cover.out

qbench: retro
	/usr/bin/time -f '%Uu %Ss %er %MkB %C' ./retro <vm/testdata/core.rx >/dev/null

retroImage: retro _misc/kernel.rx _misc/meta.rx _misc/stage2.rx
	./retro -image vm/testdata/retroImage -with _misc/meta.rx -o retroImage <_misc/kernel.rx
	./retro -with _misc/stage2.rx
	
report: $(SRC)
	@echo "=== gocyclo ===\n"
	@gocyclo . | head
	@echo "\n\n=== misspell ===\n"
	@misspell -source go $^
	@misspell -source text README.md

get-deps:
	$(GO) get github.com/pkg/errors
	$(GO) get github.com/pkg/term
