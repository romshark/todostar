vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

dist:
	bun run build:css

fmtcheck:
	@unformatted=$$(go run mvdan.cc/gofumpt@latest -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files not gofumpt formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

gencheck:
	@go generate ./...
	@if ! git diff --quiet --exit-code; then \
		echo "Generated files are not up to date. Run 'go generate ./...'"; \
		git --no-pager diff; \
		exit 1; \
	fi

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

test: fmtcheck gencheck
	go test ./... -v

templier:
	go run github.com/romshark/templier/@latest
