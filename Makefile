# Lorehallow command runner. `make help` lists every target.
# GNU Make 4+ required (on macOS, Homebrew's make with gnubin on PATH).

SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
MAKEFLAGS += --no-print-directory

ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
include $(ROOT)/toolchain.env

# --- Build identity ---------------------------------------------------------
VERSION ?= $(shell git -C $(ROOT) describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git -C $(ROOT) rev-parse HEAD 2>/dev/null || echo unknown)
GO_LDFLAGS := -s -w \
	-X github.com/dunamismax/lorehallow/server/internal/buildinfo.Version=$(VERSION) \
	-X github.com/dunamismax/lorehallow/server/internal/buildinfo.Commit=$(COMMIT)

# --- Tools --------------------------------------------------------------------
# Godot binary: override with `make client GODOT=/path/to/godot`.
GODOT ?= $(shell command -v godot 2>/dev/null || command -v godot4 2>/dev/null || \
	{ [ -x /Applications/Godot.app/Contents/MacOS/Godot ] && echo /Applications/Godot.app/Contents/MacOS/Godot; } || echo godot)
GODOT_HEADLESS := $(GODOT) --headless --path $(ROOT)/client
BIN := $(ROOT)/bin

# Local dev server address (documented in server/README.md).
SERVER_ADDR ?= 127.0.0.1:7420

# Client GDScript sources we own (addons/ and generated net/pb/ excluded).
GD_SOURCES = $(shell cd $(ROOT)/client && find . -name '*.gd' -not -path './addons/*' -not -path './net/pb/*' -not -path './.godot/*' | sort)

# Protobuf: buf breaking compares against this git ref (CI overrides it).
BUF_AGAINST_REF ?= main

# Fuzzing time per target for `make fuzz`.
FUZZTIME ?= 10s

# Art intake batch, e.g. `make art-intake BATCH=01`.
BATCH ?=

.PHONY: help
help: ## List every target with a one-line description
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# --- Setup ----------------------------------------------------------------------

.PHONY: doctor
doctor: ## Check every required tool is installed at the pinned version
	@GODOT="$(GODOT)" $(ROOT)/tools/scripts/doctor.sh

.PHONY: setup
setup: ## Install project-local dependencies (safe to re-run)
	@echo "Go modules:"; cd $(ROOT)/server && go mod download && echo "  ok"
	@$(ROOT)/tools/scripts/install-addons.sh
	@echo "Godot import:"; $(GODOT_HEADLESS) --import >/dev/null 2>&1 && echo "  ok"
	@echo "Art tools (uv):"; cd $(ROOT)/tools/art && uv sync --frozen --quiet && echo "  ok"
	@echo "Website Go modules:"; cd $(ROOT)/web && go mod download && echo "  ok"

# --- Aggregate gates ------------------------------------------------------------

.PHONY: fmt
fmt: ## Format all code in place
	cd $(ROOT)/server && gofmt -w $$(find . -name '*.go' -not -path './internal/pb/*')
	cd $(ROOT)/web && gofmt -w .
	cd $(ROOT)/client && gdformat $(GD_SOURCES)
	cd $(ROOT)/tools/art && uv run --frozen ruff format . && uv run --frozen ruff check --fix .
	buf format -w $(ROOT)/proto

.PHONY: lint
lint: server-lint proto-lint client-lint art-lint web-lint buildmd-lint content-lint ## Run every formatter check and linter

.PHONY: test
test: server-test client-test art-test web-test ## Run every test suite

.PHONY: check
check: lint proto-check test ## Everything CI runs: format checks, linters, codegen checks, tests
	@echo "make check: all green"

# --- Server -----------------------------------------------------------------------

.PHONY: server
server: ## Build and run the game server on SERVER_ADDR (default 127.0.0.1:7420)
	cd $(ROOT)/server && go build -ldflags "$(GO_LDFLAGS)" -o $(BIN)/lorehallow-server ./cmd/lorehallow-server
	$(BIN)/lorehallow-server -addr $(SERVER_ADDR) -map $(ROOT)/content/maps/starter.tmj

.PHONY: server-build
server-build: ## Build server binaries into bin/
	cd $(ROOT)/server && go build -ldflags "$(GO_LDFLAGS)" -o $(BIN)/ ./cmd/...

.PHONY: server-lint
server-lint: ## gofmt check, go vet, golangci-lint and govulncheck
	@cd $(ROOT)/server && out="$$(gofmt -l .)" && if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	cd $(ROOT)/server && go vet ./...
	cd $(ROOT)/server && golangci-lint run ./...
	cd $(ROOT)/server && go tool govulncheck ./...

.PHONY: server-test
server-test: ## go test -race ./... (includes fuzz seed corpora and allocation tests)
	cd $(ROOT)/server && go test -race -count=1 ./...

.PHONY: loadtest
loadtest: server-build ## Run a scaling.md scenario, write a draft to docs/perf/: SCENARIO=idle|spread N=100 DURATION=2m
	cd $(ROOT) && $(BIN)/lorehallow-loadtest -scenario $(or $(SCENARIO),idle) -n $(or $(N),100) \
		-duration $(or $(DURATION),2m) $(if $(WARMUP),-warmup $(WARMUP)) -commit $(COMMIT)

.PHONY: install-host
install-host: ## Install or update Lorehallow's pieces on the host (unit, Caddy site, layout): [DRY=1]
	@$(ROOT)/deploy/bootstrap/app.sh $(if $(DRY),--dry-run)

.PHONY: deploy-server
deploy-server: ## Build for the host, upload, switch, restart, check /healthz, roll back on failure
	@$(ROOT)/deploy/server/deploy.sh deploy

.PHONY: rollback-server
rollback-server: ## Switch the host back to the previous server release
	@$(ROOT)/deploy/server/deploy.sh rollback

.PHONY: rehearse-rollback-server
rehearse-rollback-server: ## Deploy a release that can't start and check the automatic rollback (game down ~20 s)
	@$(ROOT)/deploy/server/deploy.sh rehearse-rollback

.PHONY: content-lint
content-lint: ## Validate game content (maps and tilesets) in content/
	cd $(ROOT)/server && go run ./cmd/lorehallow-content validate -root $(ROOT)

.PHONY: fuzz
fuzz: ## Run every Go fuzz target for FUZZTIME each (default 10s)
	@FUZZTIME="$(FUZZTIME)" $(ROOT)/tools/scripts/fuzz.sh

.PHONY: bench
bench: ## Run server benchmarks with allocations: [BENCH=regex] [PKG=./internal/...]
	cd $(ROOT)/server && go test -run='^$$' -bench='$(or $(BENCH),.)' -benchmem $(or $(PKG),./...)

# --- Protocol ---------------------------------------------------------------------

$(BIN)/protoc-gen-go: $(ROOT)/server/go.mod $(ROOT)/server/go.sum
	cd $(ROOT)/server && go build -o $(BIN)/protoc-gen-go google.golang.org/protobuf/cmd/protoc-gen-go

.PHONY: proto
proto: $(BIN)/protoc-gen-go ## Regenerate Go and GDScript protocol code and golden vectors
	@$(ROOT)/tools/scripts/gen-proto.sh "$(GODOT)"

.PHONY: proto-lint
proto-lint: ## buf lint and buf format check
	buf lint $(ROOT)/proto
	buf format --diff --exit-code $(ROOT)/proto

.PHONY: proto-check
proto-check: proto ## Fail if generated code is stale or buf breaking reports an unplanned break
	@# Stale = regeneration changed files relative to the git index (staged or committed).
	@cd $(ROOT) && gen="server/internal/pb client/net/pb proto/testdata" && \
		if ! git diff --quiet -- $$gen || [ -n "$$(git ls-files --others --exclude-standard -- $$gen)" ]; then \
		echo "Generated protocol code is stale. Run 'make proto' and commit (or stage) the result:"; \
		git status --short -- $$gen; exit 1; fi
	@$(ROOT)/tools/scripts/proto-breaking.sh "$(BUF_AGAINST_REF)"

# --- Client -------------------------------------------------------------------------

.PHONY: client
client: ## Launch the Godot client (set GODOT=... to pick the binary)
	@$(ROOT)/tools/scripts/write-client-buildinfo.sh "$(VERSION)" "$(COMMIT)"
	$(GODOT) --path $(ROOT)/client

.PHONY: client-art-check
client-art-check: ## Render the dev art scene in a window and check it is pixel-perfect: [ART=all]
	$(GODOT) --path $(ROOT)/client -s res://scenes/dev/art_verify.gd -- $(if $(ART),--art=$(ART))

.PHONY: client-lint
client-lint: ## gdformat --check and gdlint on client scripts
	cd $(ROOT)/client && gdformat --check $(GD_SOURCES)
	cd $(ROOT)/client && gdlint $(GD_SOURCES)

.PHONY: client-test
client-test: ## Run GUT tests headless
	@$(GODOT_HEADLESS) --import >/dev/null 2>&1 || true
	@cd $(ROOT)/client && rc=0 && out="$$($(GODOT) --headless --path . -s addons/gut/gut_cmdln.gd -gconfig=.gutconfig.json 2>&1)" || rc=$$?; \
		echo "$$out" | grep -vE '^\s*$$|is now available' || true; \
		if echo "$$out" | grep -qE 'SCRIPT ERROR|Parse Error'; then echo "client-test: GDScript errors above"; exit 1; fi; \
		exit $$rc

# --- Art ------------------------------------------------------------------------------

.PHONY: art-lint
art-lint: ## ruff check, format check and art manifest validation
	cd $(ROOT)/tools/art && uv run --frozen ruff check . && uv run --frozen ruff format --check .
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art validate --root $(ROOT)

.PHONY: art-test
art-test: ## pytest for tools/art
	cd $(ROOT)/tools/art && uv run --frozen pytest -q

.PHONY: art-draw
art-draw: ## Render hand-drawn art from art/src: make art-draw [BATCH=01] [REVIEW=dir]
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art draw --root $(ROOT) $(if $(BATCH),--batch $(BATCH)) $(if $(REVIEW),--review $(abspath $(REVIEW)))

.PHONY: art-intake
art-intake: ## Process one art batch: make art-intake BATCH=01
	@if [ -z "$(BATCH)" ]; then echo "usage: make art-intake BATCH=01"; exit 2; fi
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art intake --root $(ROOT) --batch $(BATCH)

.PHONY: art-generate
art-generate: ## Rebuild palette.png, palette.gpl and every batch prompts.md from palette.hex
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art generate --root $(ROOT)

.PHONY: art-placeholders
art-placeholders: ## Write placeholder art for every manifest asset not processed yet
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art placeholders --root $(ROOT)

.PHONY: art-favicons
art-favicons: ## Export the favicon set and client app icon from the emblem
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art favicons --root $(ROOT)

.PHONY: sync
sync: ## Mirror art/processed into client/ and web/, and content/ into client/assets/content
	cd $(ROOT)/tools/art && uv run --frozen lorehallow-art sync --root $(ROOT)

# --- Website and BUILD.md ----------------------------------------------------------

.PHONY: web-dev
web-dev: ## Preview lorehallow.com at http://127.0.0.1:4321 with drafts, rebuilt on change
	cd $(ROOT)/web && go run ./cmd/lorehallow-web serve -addr $(or $(WEB_ADDR),127.0.0.1:4321)

.PHONY: web-build
web-build: ## Build lorehallow.com into web/dist/ (no drafts)
	cd $(ROOT)/web && go run ./cmd/lorehallow-web build

.PHONY: web-lint
web-lint: ## Website (Go): gofmt check, go vet, golangci-lint and govulncheck
	@cd $(ROOT)/web && out="$$(gofmt -l .)" && if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	cd $(ROOT)/web && go vet ./...
	cd $(ROOT)/web && golangci-lint run ./...
	cd $(ROOT)/web && go tool govulncheck ./...

.PHONY: web-test
web-test: ## Website (Go) tests: go test -race ./...
	cd $(ROOT)/web && go test -race -count=1 ./...

.PHONY: buildmd-lint
buildmd-lint: ## Fail on any BUILD.md format-contract violation
	cd $(ROOT)/web && go run ./cmd/buildmd-lint ../BUILD.md

.PHONY: clean
clean: ## Remove build output (bin/, Godot import cache)
	rm -rf $(BIN) $(ROOT)/client/.godot
