# automqttion Test Philosophy & Compliance

This document describes the testing standards, test architecture tiers, test mechanics, testing compliance audit, and tracked backlog items for `automqttion`.

---

## Output-Oriented Tabular Testing Philosophy

Event-driven automation rules and payload matchers must be verified through **output-oriented assertions** rather than mock interaction choreography:

1. **Assert Concrete Outputs & States**: Supply input payloads and target patterns, then assert the exact boolean match result, the published message string, or the marshaled JSON bytes.
2. **Avoid Callback Mock Choreography**: Do not use `testify/mock` to verify internal callback invocation counts. Instead, pass a lightweight test closure or channel that records dispatched messages, and assert directly against the captured collection.
3. **Table-Driven Structure**: Organize tests in tabular format:
   ```go
   tests := []struct {
       name     string
       target   any
       data     any
       expected bool
   }{ ... }
   ```

---

## 3-Tier Testing Architecture

```
+----------------------------------------------------------------------+
| Tier 3: Containerized Integration Tests (Optional)                   |
| Real Mosquitto MQTT broker via Podman / Testcontainers               |
+----------------------------------------------------------------------+
| Tier 2: In-Memory Component & Pipeline Fakes                         |
| null.NewBroker(), in-memory broker channel fixtures, context cancel  |
+----------------------------------------------------------------------+
| Tier 1: Pure Matcher, Expander, & Config Tabular Unit Tests          |
| match(), numericEqual(), replaceIfYamlMatcher(), newPublisher()      |
+----------------------------------------------------------------------+
```

### Tier 1: Pure Matcher, Expander, & Config Tabular Tests
- **Scope**: Pattern matching ([`match`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L5)), numeric coercion ([`numericEqual`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L36)), YAML expander ([`replaceIfYamlMatcher`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L42)), publisher selection ([`newPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L49)), and configuration parsing ([`config.ParseConfig`](file:///home/kasch/projects/go/automqttion/config/config.go#L65)).
- **Characteristics**: 100% in-memory, deterministic, zero network or filesystem dependencies, executes in milliseconds.
- **Target Packages**: [`internal/automation`](file:///home/kasch/projects/go/automqttion/internal/automation), [`config`](file:///home/kasch/projects/go/automqttion/config), [`pkg/log`](file:///home/kasch/projects/go/automqttion/pkg/log).

### Tier 2: In-Memory Component Fakes
- **Scope**: Verifying [`automation.Automation`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L14) event loops, topic dispatch, and context cancellation without real network sockets.
- **Fixtures**: Use [`null.NewBroker()`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go#L37) or lightweight channel-based fakes implementing [`broker.Broker`](file:///home/kasch/projects/go/automqttion/pkg/broker/broker.go#L5) and [`broker.Topic`](file:///home/kasch/projects/go/automqttion/pkg/broker/broker.go#L10) to capture published messages.

### Tier 3: Container Integration Tests
- **Scope**: Live end-to-end MQTT connectivity using an external Mosquitto broker container.
- **Execution**: Triggered in CI workflows or on demand. Verifies Paho MQTT connection, QoS 0/1 handling, and network disconnect recovery.

---

## Mandatory Test Mechanics

1. **Black-Box Test Packages**: Always define tests in a separate black-box test package (`package <pkg>_test`, e.g., `package automation_test`, `package config_test`). Only export internal helpers via `export_test.go` if strictly necessary.
2. **Parallel Execution (`t.Parallel()`)**:
   - Every top-level test function must invoke `t.Parallel()`.
   - Every subtest within table loops must invoke `t.Parallel()`.
   ```go
   func TestMatch(t *testing.T) {
       t.Parallel()
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               t.Parallel()
               actual := automation.Match(tt.target, tt.data)
               assert.Equal(t, tt.expected, actual)
           })
       }
   }
   ```
3. **Standard Test Context (`t.Context()`)**:
   - When tests require a `context.Context`, always call `t.Context()`.
   - Never use `context.Background()` or `context.TODO()` inside tests (enforced by `usetesting.context-background: true` in `.golangci.yml`).
4. **Assertion Library Standards**:
   - Use `github.com/stretchr/testify/require` for fatal preconditions (e.g., verifying `require.NoError(t, err)` or `require.NotNil(t, obj)` before dereferencing).
   - Use `github.com/stretchr/testify/assert` for non-fatal value assertions.
   - Respect parameter order: `assert.Equal(t, expected, actual)`.

---

## Testing Compliance Audit

A comprehensive codebase audit reveals significant coverage gaps and convention violations that must be addressed:

| Package | Status | Finding & Compliance Gap |
|---|---|---|
| [`internal/automation/automation.go`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go) | 0% Coverage | Untested. [`NewAutomation`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L87), [`Start`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L29), [`replaceIfYamlMatcher`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L42), and [`dereference`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L78) have no automated tests. |
| [`internal/automation/publisher.go`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go) | 0% Coverage | Untested. [`newPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L49), [`jsonPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L29), and [`plainPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L39) have no automated tests. |
| [`config`](file:///home/kasch/projects/go/automqttion/config) | 0% Coverage | Untested. [`LoadConfig`](file:///home/kasch/projects/go/automqttion/config/config.go#L42), [`ParseConfig`](file:///home/kasch/projects/go/automqttion/config/config.go#L65), and environment variable overrides have no automated tests. |
| [`pkg/broker/mqtt`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt) | 0% Coverage | Untested. Functional options ([`Broker`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/options.go#L16), [`TLS`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/options.go#L22)), client constructor, and message publishing have no tests. |
| [`pkg/broker/null`](file:///home/kasch/projects/go/automqttion/pkg/broker/null) | 0% Coverage | Untested. No-op methods of [`nullBroker`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go#L10) and [`nullTopic`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go#L11) are not verified. |
| [`pkg/log`](file:///home/kasch/projects/go/automqttion/pkg/log) & [`pkg/log/config`](file:///home/kasch/projects/go/automqttion/pkg/log/config) | 0% Coverage | Untested. Logger initialization ([`NewLogger`](file:///home/kasch/projects/go/automqttion/pkg/log/zap_logger.go#L37), [`InitLogger`](file:///home/kasch/projects/go/automqttion/pkg/log/zap_logger.go#L52)), level parsing, and console/JSON encoding toggles have no tests. |
| [`internal/cmd`](file:///home/kasch/projects/go/automqttion/internal/cmd) | 0% Coverage | Untested. [`Init`](file:///home/kasch/projects/go/automqttion/internal/cmd/init.go#L18) signal handling and context cancellation have no tests. |

---

## Tracked Backlog Items

The following testing alignment and architectural backlog items are tracked privately on GitHub Project #1 (`MyChaOS87/.projects`):

### Issue #18: Add tabular unit tests for YAML matcher expansion, publisher dispatch, and automation wiring
- **Location**: [`internal/automation/automation.go`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go), [`internal/automation/publisher.go`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go)
- **Scope**:
  - Add unit tests for `replaceIfYamlMatcher`: valid YAML string expansion, non-map inputs, and non-string errors.
  - Add unit tests for `newPublisher`: verify `$plain` creates `plainPublisher`, `$json` creates `jsonPublisher`, and untagged map falls back to `jsonPublisher`.
  - Add fake `broker.Topic` test fixture to capture `PublishString` and `Publish` invocations and assert dispatched payload bytes.
  - Add unit tests for `dereference`: pointer to int, pointer to struct, and non-pointer primitives.
  - Add unit test for `automation.Start`: verify inbound topic message invokes matching publishers.

### Issue #19: Add unit tests for config loader, logger factory, and MQTT options
- **Location**: [`config/config.go`](file:///home/kasch/projects/go/automqttion/config/config.go), [`pkg/log/zap_logger.go`](file:///home/kasch/projects/go/automqttion/pkg/log/zap_logger.go), [`pkg/broker/mqtt/options.go`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/options.go)
- **Scope**:
  - Tabular tests for `LoadConfig` with valid YAML file, non-existent file (`ErrConfigFileNotFound`), and invalid YAML syntax.
  - Test `AUTOMQTTION_*` environment variable overrides via Viper.
  - Test `NewLogger` with `development: true/false`, console vs JSON encoding, and invalid log level fallbacks.
  - Test functional options `mqtt.Broker()` and `mqtt.TLS()` on `mqtt.Config`.

### Issue #20: Add unit and boundary tests for null and MQTT message brokers
- **Location**: [`pkg/broker/null/nullBroker.go`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go), [`pkg/broker/mqtt/mqttBroker.go`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/mqttBroker.go)
- **Scope**:
  - Add unit tests verifying `nullBroker` and `nullTopic` methods execute without panic.
  - Add unit tests for `mqttTopic.Publish` and `mqttTopic.PublishString` with disconnected client handling.
  - Verify context cancellation halts `mqttBroker.Run` cleanly.

### Issue #21: Refactor CLI init and YAML matcher parsing to return errors instead of calling log.Fatal
- **Location**: [`internal/automation/automation.go`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L63), [`internal/cmd/init.go`](file:///home/kasch/projects/go/automqttion/internal/cmd/init.go#L28)
- **Scope**:
  - Refactor `replaceIfYamlMatcher(m any) (any, error)` to return errors instead of calling `log.Fatalf`.
  - Refactor `NewAutomation(..., matchersConfig []config.MatchConfig) (Automation, error)` to propagate configuration errors.
  - Refactor `cmd.Init()` to return `(context.Context, context.CancelFunc, *config.Config, error)` and remove `log.Printf("FATAL: ...")`.
  - Update `cmd/automqttion/automqttion.go` to handle returned errors and exit gracefully.
