set dotenv-load := false

default:
    just --list

install:
    pnpm install

tidy:
    go mod tidy -v
    pnpm install --lockfile-only

fmt:
    gofmt -w *.go demo/backend/main.go

go-test:
    go test ./... --race -count=1

js-test:
    pnpm test

typecheck:
    pnpm typecheck

wasm:
    pnpm run prepare:demo

pages:
    pnpm run build:pages

e2e:
    pnpm test:e2e

test: fmt go-test js-test typecheck e2e wasm

demo:
    pnpm run demo

pack:
    pnpm pack --dry-run

clean:
    rm -rf test-results playwright-report
    rm -f demo/public/formular-demo.wasm demo/public/wasm_exec.js
