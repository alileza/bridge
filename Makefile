.PHONY: build test e2e ui

TOMATO ?= tomato

## build: build the bridge binary into ./bin
build:
	go build -o bin/bridge .

## test: run unit tests
test:
	go test ./...

## e2e: run black-box tests with tomato (github.com/tomatool/tomato)
e2e: build
	mkdir -p e2e/.data && rm -f e2e/.data/*.json e2e/.data/*.jsonl
	$(TOMATO) run --config e2e/tomato.yml
	$(TOMATO) run --config e2e/tomato.auth.yml

## ui: rebuild the embedded portal UI
ui:
	cd portal/ui && pnpm install --frozen-lockfile && pnpm build
