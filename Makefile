GO ?= ~/opt/go-tip/bin/go
PKG := github.com/db47h/ngaro
SRC := vm/*.go cmd/retro/*.go

.PHONY: all install clean test bench qbench

all:
	$(GO) build $(PKG)/cmd/retro

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
	$(GO) test -v $(PKG)/vm

bench:
	$(GO) test -v $(PKG)/vm -run DONOTRUNTESTS -bench .

qbench: retro
	/usr/bin/time -f '%Uu %Ss %er %MkB %C' ./retro <vm/testdata/core.rx >/dev/null
