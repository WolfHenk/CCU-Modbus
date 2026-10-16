VERSION?=$(shell cat VERSION)
LDFLAGS=-s -w -X main.version=$(VERSION)

.PHONY: test build package clean

test:
	go mod tidy
	go test ./...
	go vet ./...
	python3 packaging/test_refresh_addon_versions.py
	python3 packaging/test_lifecycle.py
	python3 packaging/test_webui_security.py
	node --check packaging/www/app.js
	node packaging/test_inbox_navigation.js
	! grep -Eq 'v=0\.1\.[0-9]+|class="version">0\.1\.[0-9]+' packaging/www/index.cgi

build:
	mkdir -p build/bin/linux-amd64 build/bin/linux-arm64 build/bin/linux-arm
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-amd64/ccu-modbusd ./cmd/ccu-modbusd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-arm64/ccu-modbusd ./cmd/ccu-modbusd
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags "$(LDFLAGS)" -o build/bin/linux-arm/ccu-modbusd ./cmd/ccu-modbusd
	@for arch in amd64 arm64; do CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -o build/bin/linux-$$arch/ccu-modbus-update ./cmd/ccu-modbus-update || exit 1; done
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -o build/bin/linux-arm/ccu-modbus-update ./cmd/ccu-modbus-update

package: test build
	rm -rf build/pkg && mkdir -p build/pkg/bin build/pkg/rc.d build/pkg/www build/pkg/lib build/pkg/examples build/pkg/docs
	cp -a build/bin/* build/pkg/bin/
	cp packaging/bin/update_addon build/pkg/bin/update_addon
	cp packaging/bin/register_devtype build/pkg/bin/register_devtype
	chmod 755 build/pkg/bin/update_addon build/pkg/bin/register_devtype
	cp packaging/update_script build/pkg/
	cp packaging/rc.d/ccu-modbus build/pkg/rc.d/
	cp -a packaging/www/. build/pkg/www/
	cp -a packaging/lib/. build/pkg/lib/
	cp examples/config.json build/pkg/examples/
	cp docs/CCU-Modbus_0.2.0_Testprotokoll_2026-10-15.md build/pkg/docs/
	cp VERSION build/pkg/
	cd build/pkg && tar -czf ../ccu-modbus-$(VERSION).tar.gz .

clean:
	rm -rf build
