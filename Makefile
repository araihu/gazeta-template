export GOWORK := off
export WRANGLER_SEND_METRICS := false

.PHONY: install build check run dry-run
install:
	go mod download
build: install
	go tool gazeta build
check: build
	./scripts/check.sh
run: build
	go tool gazeta serve
dry-run: build
	go run ./internal/deployconfig -worker gazeta-check -allow-placeholder -out .build/wrangler.dry-run.json
	npx --no-install wrangler deploy --config .build/wrangler.dry-run.json --dry-run --outdir .build/wrangler-output
