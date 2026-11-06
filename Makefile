export GOWORK=off
PWD := $(shell pwd)

SBOM_FILE := automqttion.sbom.cdx.json
REPORTS_DIR := $(PWD)/reports
GRYPE_TEMPLATE := $(PWD)/tools/grype/report.md.tmpl
CONTAINER_ENGINE ?= podman

.PHONY: all
all: tidy build lint test notice sbom-check

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	rm -rf $(PWD)/coverage
	mkdir -p $(PWD)/coverage
	go test -v -race -cover -coverprofile=coverage.out -covermode=atomic ./... 2>&1 | tee coverage/unit_test.log
	go tool -modfile=tools/go-junit-report/go.mod go-junit-report --set-exit-code < coverage/unit_test.log > coverage/unit_test.xml

.PHONY: coverage
coverage: test
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o=coverage/coverage.html

.PHONY: lint
lint:
	go tool -modfile=tools/golangci-lint/go.mod golangci-lint run ./...
	go fix -diff ./...

.PHONY: lint.fix
lint.fix:
	go tool -modfile=tools/golangci-lint/go.mod golangci-lint run --fix ./...
	go fix ./...

.PHONY: fix
fix: lint.fix

.PHONY: tidy
tidy: tools-tidy
	go mod tidy

.PHONY: tidy-check
tidy-check: tidy
	git diff --exit-code -- '*go.mod' '*go.sum'

.PHONY: notice
notice:
	go list -m -json all | \
	go tool -modfile=tools/go-licence-detector/go.mod go-licence-detector \
	    -rules tools/go-licence-detector/rules.yaml \
	    -includeIndirect \
	    -overrides tools/go-licence-detector/overrides.yaml \
	    -depsOut=dependencies.asciidoc -depsTemplate tools/go-licence-detector/dependencies.asciidoc.tmpl \
	    -noticeOut=NOTICE.txt -noticeTemplate tools/go-licence-detector/NOTICE.txt.tmpl

.PHONY: sbom
sbom:
	go tool -modfile=tools/syft/go.mod syft scan dir:. \
		--source-name automqttion \
		--exclude './.git/**' \
		--exclude './tools/**' \
		--exclude './coverage/**' \
		--exclude './reports/**' \
		--exclude './.act/**' \
		--exclude './.vscode/**' \
		--exclude './$(SBOM_FILE)' \
		-o cyclonedx-json | \
	jq 'del(.serialNumber, .metadata.timestamp) | if .components then .components |= map(if .properties then .properties |= map(if (.name == "syft:location:0:path" and (.value | startswith("/"))) then .value |= ltrimstr("/") else . end) else . end) else . end' \
		> $(SBOM_FILE)

.PHONY: sbom-check
sbom-check: sbom
	git diff --exit-code $(SBOM_FILE)

.PHONY: vulncheck
vulncheck:
	go tool -modfile=tools/govulncheck/go.mod govulncheck ./...

.PHONY: audit-repo
audit-repo: sbom
	mkdir -p $(REPORTS_DIR)
	go tool -modfile=tools/grype/go.mod grype sbom:$(SBOM_FILE) --config .grype.yaml -o table
	go tool -modfile=tools/grype/go.mod grype sbom:$(SBOM_FILE) --config .grype.yaml -o template -t $(GRYPE_TEMPLATE) --file $(REPORTS_DIR)/grype-repo.md

.PHONY: audit-image
audit-image:
	mkdir -p $(REPORTS_DIR)
	$(CONTAINER_ENGINE) build -t automqttion:audit -f build/automqttion/Dockerfile .
	go tool -modfile=tools/syft/go.mod syft scan automqttion:audit -o cyclonedx-json=$(REPORTS_DIR)/automqttion-image.sbom.cdx.json
	go tool -modfile=tools/grype/go.mod grype sbom:$(REPORTS_DIR)/automqttion-image.sbom.cdx.json --config .grype.yaml -o table
	go tool -modfile=tools/grype/go.mod grype sbom:$(REPORTS_DIR)/automqttion-image.sbom.cdx.json --config .grype.yaml -o template -t $(GRYPE_TEMPLATE) --file $(REPORTS_DIR)/grype-image.md

.PHONY: audit
audit: vulncheck audit-repo

.PHONY: github-actions-test
github-actions-test:
	go tool -modfile=tools/act/go.mod act -j test-and-build --artifact-server-path ~/.act/artifacts

.PHONY: docker-build
docker-build:
	$(CONTAINER_ENGINE) build -t automqttion -f build/automqttion/Dockerfile .

.PHONY: tools-update
tools-update:
	for d in tools/* ; do \
		if [ -d $$d ] && [ -f $$d/go.mod ] ; then \
			echo "Updating $$d" ; \
			cd $$d ; \
			go get -u $$(go list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all) ; \
			go mod tidy ; \
			cd - >/dev/null; \
		fi ; \
	done

.PHONY: tools-tidy
tools-tidy:
	for d in tools/* ; do \
		if [ -d $$d ] && [ -f $$d/go.mod ] ; then \
			echo "tidy $$d" ; \
			cd $$d ; \
			go mod tidy ; \
			cd - >/dev/null; \
		fi ; \
	done

.PHONY: upgrade-direct-dependencies
upgrade-direct-dependencies: tidy
	echo "Upgrading direct dependencies..."
	go get -u $$(go list -m -f '{{if and (not .Indirect) (not .Main) (not .Replace)}}{{.Path}}@latest{{end}}' all)
	go mod tidy
