# automqttion Developer Toolchain & Quality Gates

This document details the build targets, static analysis tools, vulnerability scanning pipelines, tool module isolation, and licensing compliance rules for `automqttion`.

---

## Makefile Target Reference

All repository automation is driven by [`Makefile`](file:///home/kasch/projects/go/automqttion/Makefile).
The makefile sets `export GOWORK=off` at the top level to guarantee build repeatability independent of any ambient workspace `go.work` file.

| Target | Description | Gate / Action |
|---|---|---|
| `make all` | Full local CI validation pipeline. | Runs `tidy`, `build`, `lint`, `test`, `notice`, and `sbom-check`. |
| `make build` | Compiles all packages and binary. | Runs `go build ./...`. |
| `make test` | Executes unit tests with race detection and generates coverage logs and JUnit XML. | Runs `go test -v -race -cover -coverprofile=coverage.out -covermode=atomic ./...` and translates logs via `go-junit-report`. |
| `make coverage` | Computes test coverage percentage and HTML visualizer. | Runs `go tool cover -func=coverage.out` and generates `coverage/coverage.html`. |
| `make lint` | Runs static analyzers and AST checks. | Runs `golangci-lint run ./...` and `go fix -diff ./...`. |
| `make lint.fix` | Automatically fixes auto-correctable linter issues and AST diffs. | Runs `golangci-lint run --fix ./...` and `go fix ./...`. |
| `make fix` | Alias for `make lint.fix`. | Shorthand for automated codebase repair. |
| `make tidy` | Tidies all isolated tool modules and root module. | Runs `tools-tidy` then `go mod tidy`. |
| `make tidy-check` | Verifies that `go.mod` and `go.sum` require no changes. | Runs `git diff --exit-code -- '*go.mod' '*go.sum'`. |
| `make notice` | Re-generates open source notice and dependencies list. | Runs `go-licence-detector` to output `NOTICE.txt` and `dependencies.asciidoc`. |
| `make sbom` | Generates a normalized CycloneDX Software Bill of Materials (SBOM). | Runs `syft scan dir:.` excluding tool caches and strips non-deterministic serial numbers/timestamps via `jq` into `automqttion.sbom.cdx.json`. |
| `make sbom-check` | Asserts that `automqttion.sbom.cdx.json` is clean and committed. | Runs `git diff --exit-code automqttion.sbom.cdx.json`. |
| `make vulncheck` | Analyzes code for known vulnerable Go functions via static call graph. | Runs `govulncheck ./...`. |
| `make audit-repo` | Scans the repository SBOM for known vulnerabilities using Grype. | Produces table output and markdown report in `reports/grype-repo.md`. |
| `make audit-image` | Builds the container image and scans its runtime filesystem SBOM. | Builds `automqttion:audit` via Podman/Docker, generates image SBOM, and writes `reports/grype-image.md`. |
| `make audit` | Combines Go source and repository artifact auditing. | Runs `vulncheck` and `audit-repo`. |
| `make github-actions-test` | Runs GitHub Actions workflows locally using `act`. | Runs `act -j test-and-build --artifact-server-path ~/.act/artifacts`. |
| `make docker-build` | Builds production container image. | Runs `podman build -t automqttion -f build/automqttion/Dockerfile .`. |
| `make tools-update` | Upgrades all tool dependencies in `tools/*/go.mod`. | Updates direct dependencies in each tool submodule. |
| `make tools-tidy` | Tidies all isolated `tools/*/go.mod` files. | Runs `go mod tidy` across all subdirectories in `tools/`. |
| `make upgrade-direct-dependencies` | Upgrades direct root module dependencies to latest versions. | Declares `tidy` as a prerequisite (`upgrade-direct-dependencies: tidy`), runs `go get -u` across direct dependencies, and tidies `go.mod`. |

---

## Tool Isolation Architecture (`tools/*/dummy.go`)

To avoid polluting the root [`go.mod`](file:///home/kasch/projects/go/automqttion/go.mod) with heavy CLI tools, every external tool runs inside its own dedicated module under `tools/<tool>/go.mod`.

### Tools Inventory
- `tools/golangci-lint`: Multi-linter framework for Go (`v2`).
- `tools/go-junit-report`: Converts `go test` output to standard JUnit XML reports.
- `tools/go-licence-detector`: Scans Go module licenses against compliance allowlists.
- `tools/syft`: Generates CycloneDX SBOMs for repository and container images.
- `tools/grype`: Vulnerability scanner for SBOM files.
- `tools/govulncheck`: Official Go vulnerability scanner.
- `tools/act`: Executes GitHub Actions workflows locally.

### Invocation Pattern
Tools are executed using the modern Go tool flag without modifying ambient environment state:
```bash
go tool -modfile=tools/golangci-lint/go.mod golangci-lint run ./...
```

### The `dummy.go` Mechanism
Standard `go mod tidy` behavior removes modules from `go.mod` if no Go source code in that package imports them.
To lock each tool as a direct dependency pinned to an exact version, every tool module contains a `dummy.go` file:
```go
package golangcilint

// This is a dummy file to make sure that `go mod tidy` keeps the golangci-lint dependency
import _ "github.com/golangci/golangci-lint/v2/cmd/golangci-lint"
```
**CRITICAL**: Never delete or strip `tools/<tool>/dummy.go`. Doing so causes `go mod tidy` to discard the tool dependency or relegate it to indirect.

---

## AST Modernization (`go fix`)

`automqttion` enforces automated AST modernization via `go fix`:
- Enforced during `make lint`:
  ```bash
  go fix -diff ./...
  ```
- If discrepancies exist, run `make fix` (or `make lint.fix`) to apply standard library updates automatically.

---

## Vulnerability Auditing & Remediation Policy

The security workflow combines static analysis, SBOM generation, and vulnerability scanning:

```
[Source Code] ---> govulncheck ./... (Call-graph reachability)
      |
      v
  syft scan ---> automqttion.sbom.cdx.json (Normalized CycloneDX SBOM)
      |
      v
  grype sbom ---> .grype.yaml policy ---> reports/grype-repo.md
```

### `.grype.yaml` Policy
Vulnerability suppressions in [`.grype.yaml`](file:///home/kasch/projects/go/automqttion/.grype.yaml) must follow strict standards:
- Every entry in `ignore:` must include an inline justification comment explaining why the vulnerability is not exploitable in `automqttion` (e.g., unreachable code path, CLI-only scope, or mitigated by runtime container boundaries).
- Never add blanket or uncommented ignores.

### Transitive CVE Remediation Rule
When a vulnerability is detected in a transitive (indirect) dependency:
1. **Rule**: Never promote the indirect dependency to a direct dependency in `go.mod`.
2. **Rule**: Add an explicit, version-pinned `replace` directive in `go.mod`:
   ```
   replace <module> <vulnerable-version> => <module> <fixed-version> // fix <CVE-ID>
   ```
   This ensures the workaround is self-documenting, pinned to a safe patch, and easily removed once the direct upstream dependency updates its requirements.

---

## Licensing & Compliance

### Project License
`automqttion` is licensed under the **GNU General Public License v3.0 (GPL-3.0)** as defined in [`LICENSE.MD`](file:///home/kasch/projects/go/automqttion/LICENSE.MD).

### Dependency Licensing
Dependencies are audited against [`tools/go-licence-detector/rules.yaml`](file:///home/kasch/projects/go/automqttion/tools/go-licence-detector/rules.yaml):
- Approved licenses include: `Apache-2.0`, `MIT`, `BSD-3-Clause`, `BSD-2-Clause`, `ISC`, `MPL-2.0`, `0BSD`, and `EPL-2.0`.
- The MQTT client dependency `github.com/eclipse/paho.mqtt.golang` is dual-licensed under **EPL-2.0 OR BSD-3-Clause**, both of which are approved.

### Workspace Licensing Rules
- **Never use `overrides.yaml` to fake a license**: Do not use override configurations to misrepresent a dependency's license type to bypass CI checks.
- **Whitelist approved licenses**: If a legitimate third-party dependency uses a permissible license not yet in `rules.yaml`, request approval from the user and add the license identifier to the `allowlist` in `tools/go-licence-detector/rules.yaml`.
