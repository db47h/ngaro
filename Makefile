PKG := github.com/db47h/ngaro
SRC := vm/*.go cmd/retro/*.go

.PHONY: all install clean test bench qbench get-deps

all:
	go test -v $(PKG)/vm

retro: $(SRC)
	go build $(PKG)/cmd/retro

install:
	go install $(PKG)/cmd/retro

clean:
	go clean -i $(PKG)/cmd/retro
	$(RM) retro

distclean:
	go clean -i -r $(PKG)/cmd/retro
	$(RM) retro

bench:
	go test -v $(PKG)/vm -run DONOTRUNTESTS -bench .

qbench: retro
	/usr/bin/time -f '%Uu %Ss %er %MkB %C' ./retro <vm/testdata/core.rx >/dev/null

get-deps:
	go get github.com/pkg/errors
	go get github.com/pkg/term
