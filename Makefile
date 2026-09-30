VERSION?=$(shell cat VERSION)
LDFLAGS=-s -w -X main.version=$(VERSION)

.PHONY: test build package clean

test:
	go test ./...
	go vet ./...

build:
	mkdir -p build/bin/linux-amd64 build/bin/linux-arm64 build/bin/linux-arm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-amd64/ccu-modbusd ./cmd/ccu-modbusd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-arm64/ccu-modbusd ./cmd/ccu-modbusd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-arm/ccu-modbusd ./cmd/ccu-modbusd

package: test build
	rm -rf build/pkg && mkdir -p build/pkg/bin build/pkg/rc.d build/pkg/www build/pkg/examples
	cp -a build/bin/* build/pkg/bin/
	cp packaging/update_script build/pkg/
	cp packaging/rc.d/ccu-modbus build/pkg/rc.d/
	cp packaging/www/* build/pkg/www/
	cp examples/config.json build/pkg/examples/
	cp VERSION build/pkg/
	cd build/pkg && tar -czf ../ccu-modbus-$(VERSION).tar.gz .

clean:
	rm -rf build
