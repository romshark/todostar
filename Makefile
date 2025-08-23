checkvuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

dist:
	bun run build:css

checkfmt:
	@unformatted=$$(go run mvdan.cc/gofumpt@latest -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "Files not gofumpt formatted:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

checkgen:
	@go generate ./...
	@if ! git diff --quiet --exit-code; then \
		echo "Generated files are not up to date. Run 'go generate ./...'"; \
		git --no-pager diff; \
		exit 1; \
	fi

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...

test: checkfmt checkgen
	go test ./... -v

templier:
	go run github.com/romshark/templier@latest

scc:
	go run github.com/boyter/scc/v3@latest . \
		--exclude-dir node_modules \
		--exclude-dir server/static \
		--not-match '_templ\.go$$' \
		--not-match '_gen\.go$$'
