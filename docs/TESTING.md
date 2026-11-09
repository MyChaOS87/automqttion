# automqttion Test Philosophy & Compliance

This document describes the testing standards, test architecture tiers, and mandatory test mechanics for `automqttion`.

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
| mqtttest.NewBroker() & mqtttest.Simulator in-memory Paho fakes       |
+----------------------------------------------------------------------+
| Tier 1: Pure Matcher, Expander, & Config Tabular Unit Tests          |
| match(), numericEqual(), replaceIfYamlMatcher(), newPublisher()      |
+----------------------------------------------------------------------+
```

### Tier 1: Pure Matcher, Expander, & Config Tabular Tests
- **Scope**: Pattern matching ([`match`](../internal/automation/match.go)), numeric coercion ([`numericEqual`](../internal/automation/match.go)), YAML expander ([`replaceIfYamlMatcher`](../internal/automation/automation.go)), publisher selection ([`newPublisher`](../internal/automation/publisher.go)), and configuration parsing ([`config.ParseConfig`](../config/config.go)).
- **Characteristics**: 100% in-memory, deterministic, zero network or filesystem dependencies, executes in milliseconds.
- **Target Packages**: [`internal/automation`](../internal/automation), [`config`](../config), [`pkg/log`](../pkg/log).

### Tier 2: In-Memory Component Fakes
- **Scope**: Verifying [`automation.Automation`](../internal/automation/automation.go) event loops, topic dispatch, and context cancellation without real network sockets.
- **Fixtures**: Use [`mqtttest.NewBroker`](../pkg/broker/mqtt/mqtttest/simulator.go) and [`mqtttest.Simulator`](../pkg/broker/mqtt/mqtttest/simulator.go) to inject an in-memory `paho.Client` fake into [`mqtt.NewBroker`](../pkg/broker/mqtt/mqttBroker.go) and capture published messages.

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
