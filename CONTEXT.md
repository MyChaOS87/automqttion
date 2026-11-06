# automqttion Ubiquitous Language & Domain Glossary

This document defines the ubiquitous language, architectural terminology, and core domain abstractions across the `automqttion` codebase.

---

## Automation & Reactive Rule Domain

### [`Automation`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L14)
The top-level execution interface for a topic-bound automation rule set:
```go
type Automation interface {
    Start(ctx context.Context, cancel context.CancelFunc)
}
```
An `automation` instance listens to a single MQTT inbound topic via `SubscribeRawJSON`. When a message arrives, it evaluates each registered `matcher` sequentially and triggers associated actions.

### [`matcher`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L18)
An internal binding pairing a target pattern (`match any`) with an array of action dispatchers (`publishers []publisher`):
```go
type matcher struct {
    match      any
    publishers []publisher
}
```
When `match(matcher.match, payload)` evaluates to `true`, all associated publishers are executed in order.

### Subset Pattern Matching ([`match`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L5))
The core matching algorithm implements **declarative subset matching**:
- If `target` is a `map[string]any`: The incoming data must also be a `map[string]any`. Every key-value pair in `target` must recursively match the corresponding key in `data`. The incoming `data` may contain extra keys not specified in `target` (subset matching).
- If `target` and `data` have identical Go types: Evaluated via `reflect.DeepEqual(target, data)`.
- If types differ: Evaluated via `numericEqual(target, data)` to handle cross-type numeric equality.

### Numeric Coercion & Equality ([`numericEqual`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L36))
In JSON and YAML unmarshaling, numbers frequently arrive with differing concrete Go types (e.g., `int` from YAML literals vs `float64` from standard JSON decoders). The [`numericEqual`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L36) helper validates that both operands are numeric (via [`isNumeric`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L26)), checks bidirectional type convertibility, and ensures equality after bidirectional conversion so that precision is not lost:
```go
return av.Convert(bv.Type()).Interface() == bv.Interface() &&
    bv.Convert(av.Type()).Interface() == av.Interface()
```

### Embedded YAML Matcher Syntax (`$yaml`)
To write structured multi-field matchers cleanly within the application configuration without escaping JSON quotes, `automqttion` supports the `$yaml` operator:
```yaml
match:
  $yaml: |
    Src: 34
    Cmd: 104
    Payload: "MAQ="
```
During initialization, [`replaceIfYamlMatcher`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L42) detects the `$yaml` key, parses the multi-line string into a native `map[string]any` via `go.yaml.in/yaml/v3`, and replaces the pattern with the parsed map.

### Pointer Dereferencing ([`dereference`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L78))
Incoming payloads from subscriber callbacks may arrive as pointers or raw values. The [`dereference`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L78) helper unwraps `reflect.Pointer` instances into their underlying values before passing them to the matcher.

---

## Action Dispatch & Publisher Domain

### [`publisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L10)
An abstraction encapsulating reactive command dispatch to a destination MQTT topic:
```go
type publisher interface {
    Publish()
}
```

### [`newPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L49) Factory & Content Operators
When instantiating action publishers from configuration, the content object is inspected to select the appropriate publisher implementation:

1. **`$plain` Operator ([`plainPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L39))**:
   Used for simple string or binary switch commands (e.g., sending `on` or `off` to smart plugs):
   ```yaml
   content:
     $plain: on
   ```
   Invokes [`topic.PublishString(fmt.Sprintf("%v", p.content))`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L46).

2. **`$json` Operator ([`jsonPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L29))**:
   Used for structured JSON payload dispatching:
   ```yaml
   content:
     $json:
       state: "active"
       brightness: 100
   ```
   Invokes [`topic.Publish(j.content)`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L36), serializing the data via `json.Marshal`.

3. **Default Fallback**: If no operator key is detected, content is dispatched through `jsonPublisher` by default.

### [`ActionConfig`](file:///home/kasch/projects/go/automqttion/config/config.go#L24)
A configuration struct defining a target topic and the content payload to be published:
```go
type ActionConfig struct {
    Topic   string
    Content any
}
```

---

## Pub/Sub Messaging Domain

### [`broker.Broker`](file:///home/kasch/projects/go/automqttion/pkg/broker/broker.go#L5)
The high-level broker interface decoupling the automation core from specific transport implementations:
```go
type Broker interface {
    Run(ctx context.Context, cancel context.CancelFunc)
    Topic(topic string) Topic
}
```
- `Run`: Starts background connection management, blocking or tracking connection state until context cancellation.
- `Topic`: Returns a thread-safe handle for pub/sub operations on a named topic.

### [`broker.Topic`](file:///home/kasch/projects/go/automqttion/pkg/broker/broker.go#L10)
Defines publishing and subscription primitives:
```go
type Topic interface {
    PublishString(s string)
    Publish(i any)
    Subscribe(hint any, callback CallbackFunction)
    SubscribeRawJSON(callback CallbackFunction)
}
```
- `PublishString`: Sends an unencoded text message with QoS 1.
- `Publish`: Encodes `any` data to JSON and sends with QoS 1.
- `Subscribe`: Subscribes with QoS 0, unmarshals the payload into the type provided by `hint` (using reflection), and invokes `callback`.
- `SubscribeRawJSON`: Subscribes with QoS 0, unmarshals the payload into a generic `any` (`map[string]any` or slice), and invokes `callback`.

### [`mqttBroker`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/mqttBroker.go#L24) & [`mqttTopic`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/mqttBroker.go#L19)
The concrete implementation of `broker.Broker` backed by the Eclipse Paho MQTT client (`github.com/eclipse/paho.mqtt.golang`). Configured via functional options:
- [`mqtt.Broker(url)`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/options.go#L16): Sets target broker address (e.g. `tcp://mosquitto:1883`).
- [`mqtt.TLS(cfg)`](file:///home/kasch/projects/go/automqttion/pkg/broker/mqtt/options.go#L22): Attaches TLS credentials for secure connections.

### [`nullBroker`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go#L10) & [`nullTopic`](file:///home/kasch/projects/go/automqttion/pkg/broker/null/nullBroker.go#L11)
A no-op implementation of `broker.Broker` and `broker.Topic` that discards publishes and ignores subscriptions. Used for testing, offline verification, or dry runs.

---

## Configuration & Runtime Domain

### [`config.Config`](file:///home/kasch/projects/go/automqttion/config/config.go#L35)
The root configuration struct loaded via Viper:
```go
type Config struct {
    Logger   loggerConfig.Logger
    Mqtt     MqttConfig
    Automate map[string][]MatchConfig
}
```

### Environment Variable Mapping
The configuration loader sets `AUTOMQTTION` as the environment variable prefix and replaces dots with underscores:
- `AUTOMQTTION_MQTT_BROKER` maps to `mqtt.broker`
- `AUTOMQTTION_LOGGER_LEVEL` maps to `logger.level`

### Logging Infrastructure ([`pkg/log`](file:///home/kasch/projects/go/automqttion/pkg/log))
A structured logging wrapper around Uber Zap:
- Standard levels: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.
- Configurable encoders: `console` (human-readable) vs `json` (production telemetry).
- Development mode: Toggles caller reporting and stacktraces.

### Smart Home Protocol Bridging Scenarios
`automqttion` was designed to solve real-world heterogeneous smart home protocol bridging over MQTT.
For example, translating binary Local Control Network (LCN) status broadcasts into commands for Shelly WiFi smart relays:
- **Trigger**: An LCN module transmits a status report on topic `lcn/segment/0/target/4/` with payload `{"Src":34,"Cmd":104,"Payload":"MAQ="}`.
- **Evaluation**: The matcher checks that `Src == 34`, `Cmd == 104`, and `Payload == "MAQ="`.
- **Action**: Two separate actions fire, sending plain `on` strings to `shellyplusplugs-kaffeelicht/command/switch:0` and `shellyplusplugs-spühle/command/switch:0`.
