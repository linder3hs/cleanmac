.PHONY: build install clean test bench

BINARY := cleanmac
PREFIX := /usr/local/bin

build:
	go build -ldflags="-s -w" -o $(BINARY) .

install: build
	cp $(BINARY) $(PREFIX)/$(BINARY)
	@echo "Installed to $(PREFIX)/$(BINARY)"

clean:
	rm -f $(BINARY)

test:
	go test ./...

bench:
	go run -tags bench . bench

# Universal binary for distribution (requires both arch builds)
build-universal:
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-arm64 .
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-amd64 .
	mkdir -p dist
	lipo -create -output dist/$(BINARY) dist/$(BINARY)-arm64 dist/$(BINARY)-amd64
	@echo "Universal binary: dist/$(BINARY)"
