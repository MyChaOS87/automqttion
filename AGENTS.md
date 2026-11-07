# automqttion Agent Guide

`automqttion` is a lightweight, event-driven Go automation engine that enables declarative MQTT message routing, structural pattern matching, and automated multi-topic action dispatching across IoT and smart home environments.

The repository compiles a single core CLI binary:
- [`cmd/automqttion`](./cmd/automqttion/automqttion.go): An event-driven daemon that connects to an MQTT broker, subscribes to configured input topics, applies recursive payload matchers (including nested JSON maps, numeric type coercion, and embedded YAML expressions), and dispatches reactive actions (plain string state commands or structured JSON messages) to target MQTT topics.

---

## Engineering Principles

- **Pre-v1 Engineering Integrity**: `automqttion` is currently pre-v1 software. Never introduce backwards-compatibility shims, deprecated legacy wrappers, or speculative abstraction layers. Refactor callers in-place and remove superseded code cleanly.
- **Declarative MQTT-Native Simplicity**: Keep rule evaluation deterministic, fast, and simple. Favor straightforward, idiomatic Go standard library code over heavy external workflow frameworks. Rules must remain expressible directly in configuration.
- **Strict Boundary Decoupling**: Generic packages under [`pkg/`](./pkg) must remain completely independent of application concerns. They must never depend on [`cmd/`](./cmd), [`config/`](./config), or [`internal/`](./internal) (enforced by `depguard`).

---

## Knowledgebase Routing Table

Consult these dedicated documents for detailed technical specifications and patterns:

| Document | Scope & When to Read |
|---|---|
| [`CONTEXT.md`](./CONTEXT.md) | **Domain Glossary & Ubiquitous Language**: Definitions of declarative automation, [`Automation`](./internal/automation/automation.go#L14), [`matcher`](./internal/automation/automation.go#L18), recursive subset matching ([`match`](./internal/automation/match.go#L5)), numeric coercion ([`numericEqual`](./internal/automation/match.go#L36)), YAML expansion (`$yaml`), [`publisher`](./internal/automation/publisher.go#L10) (`$plain` vs `$json`), [`broker.Broker`](./pkg/broker/broker.go#L5), [`broker.Topic`](./pkg/broker/broker.go#L10), and [`config.Config`](./config/config.go#L35). |
| [`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) | **System Architecture**: Package DAG (`pkg/` vs `internal/` boundary), reactive pub/sub message flow sequence, declarative pattern matching pipeline, publisher polymorphism, and multi-stage `FROM scratch` container builds with Helm chart deployment. |
| [`docs/CONVENTIONS.md`](./docs/CONVENTIONS.md) | **Coding Idioms & Standards**: Go 1.27 conventions, AST modernization (`go fix`), declarative operators (`$yaml`, `$plain`, `$json`), reflection conventions, sentinel error wrapping (`%w`), functional options for MQTT, Viper environment mapping (`AUTOMQTTION_*`), and commit conventions. |
| [`docs/TESTING.md`](./docs/TESTING.md) | **Test Philosophy & Compliance**: Output-oriented tabular testing for declarative matchers and publishers, 3-tier testing model, `t.Parallel()` and `t.Context()` compliance, test suite audit, and the 5 tracked backlog items (#17–#21). |
| [`docs/TOOLCHAIN.md`](./docs/TOOLCHAIN.md) | **Developer Toolchain**: Complete [`Makefile`](./Makefile) target reference, `export GOWORK=off`, isolated tool modules with `dummy.go`, `upgrade-direct-dependencies: tidy` prerequisite, vulnerability scanning (`syft`, `grype`, `govulncheck`), and GPL-3.0 / EPL-2.0 licensing policies. |

---

## Git & Solo Linear-`main` Workflow

- **Linear-`main` Development**: Solo maintainer mode is active. Work directly on `main` for cohesive changes, or create an isolated feature branch / worktree and merge back to `main` using fast-forward only (`git merge --ff-only`).
- **Remote Mutation Safeguard**: Never run `git push`, `git push --force`, or any command that mutates git remotes without explicit, affirmative user confirmation.

---

## Sandbox Execution Matrix (`BypassSandbox`)

Prefer running commands in the default sandboxed environment (`BypassSandbox: false`). Escalate to `BypassSandbox: true` only when encountering genuine environment boundaries:

| Operation | `BypassSandbox` | Reason |
|---|---|---|
| `make test`, `make coverage`, `go test ./...` | `false` | Runs entirely in-memory using local workspace mounts and stdlib tests. |
| `make lint`, `make build`, `make tidy`, `make sbom`, `make notice` | `false` | Offline static analysis, AST verification, and local file generation using pre-installed toolchains. |
| `make audit-image`, `make docker-build` | `true` | Requires Podman or Docker container engine socket access (`/run/user/...` or `/var/run/docker.sock`). |
| `make audit-repo`, `make vulncheck`, `make audit` | `true` | Requires outbound network access to fetch remote vulnerability databases (Grype DB and Go vulnerability database). |
| `make upgrade-direct-dependencies`, `make tools-update`, `git fetch/pull/push` | `true` | Requires outbound network access to Go module proxies and remote git repositories. |

---

## Agent Hazards (Do Not Do)

1. **Never allow `pkg/` to import `cmd/`, `config/`, or `internal/`**: The `pkg/` tree provides reusable, generic primitives (broker abstraction, MQTT client, logging). Enforced strictly by `.golangci.yml` `depguard` rules.
2. **Never block the MQTT callback goroutine**: The Paho MQTT callback dispatches incoming messages on internal worker threads. Action execution in `automation` and publisher routines must be lightweight and non-blocking to prevent stalling the MQTT message loop.
3. **Never call `log.Fatal` in libraries or parsing functions**: Currently, [`replaceIfYamlMatcher`](./internal/automation/automation.go#L63) and [`cmd.Init`](./internal/cmd/init.go#L28) invoke `log.Fatalf` / `log.Printf` on configuration errors, terminating the process ungracefully. This is tracked in Issue #21 to be refactored into returning structured errors. Do not introduce new fatal exits.
4. **Never hand-edit generated license or SBOM assets**: Do not manually edit [`NOTICE.txt`](./NOTICE.txt), `dependencies.asciidoc`, or `automqttion.sbom.cdx.json`. Re-generate them via `make notice` and `make sbom`.
5. **Never delete or strip `tools/<tool>/dummy.go` files**: Each isolated tool module requires a `dummy.go` file with a blank import `import _ "<pkg>"` to pin direct tool versions and prevent `go mod tidy` from stripping tool dependencies.
6. **Never remediate transitive CVEs by promoting indirect dependencies**: Do not convert an `// indirect` dependency into a direct dependency in [`go.mod`](./go.mod). Use a version-pinned `replace` directive instead:
   ```
   replace <module> <vulnerable-version> => <module> <fixed-version> // fix <CVE-ID>
   ```
