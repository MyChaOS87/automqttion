# automqttion Ubiquitous Language & Domain Glossary

This document defines the ubiquitous language, architectural terminology, and core domain abstractions across the `automqttion` codebase.

---

## Automation & Reactive Rule Domain

### [`Automation`](./internal/automation/automation.go)
The top-level execution interface for a topic-bound automation rule set:
```go
type Automation interface {
    Start()
}
```
An `automation` instance listens to a single MQTT inbound topic via `SubscribeRaw`, evaluating each registered `matcher` sequentially (direct string comparison for `$plain` rules, or JSON decode with plain-text string fallback) and triggering associated actions.

### [`matcher`](./internal/automation/automation.go)
An internal binding pairing a target pattern (`match any`), an optional raw-text flag (`plain bool`), and an array of action dispatchers (`publishers []publisher`):
```go
type matcher struct {
    match      any
    plain      bool
    publishers []publisher
}
```
When a message matches `matcher` (either exact string equality when `plain` is `true`, or `match(matcher.match, payload)` otherwise), all associated publishers are executed in order.

### Subset Pattern Matching ([`match`](./internal/automation/match.go))
The core matching algorithm implements **declarative subset matching**:
- If `target` is a `map[string]any`: The incoming data must also be a `map[string]any`. Every key in `target` must exist in `data` and recursively match its corresponding value. The incoming `data` may contain extra keys not specified in `target` (subset matching).
- If `target` is a `[]any`: The incoming data must be a `[]any` of equal length, and each element is recursively compared via `match(tgt[i], dat[i])` (supporting cross-type numeric equality inside arrays).
- If `target` and `data` have identical Go types: Evaluated via `reflect.DeepEqual(target, data)`.
- If types differ: Evaluated via `numericEqual(target, data)` to handle cross-type numeric equality.

### Numeric Coercion & Equality ([`numericEqual`](./internal/automation/match.go))
In JSON and YAML unmarshaling, numbers frequently arrive with differing concrete Go types (e.g., `int` from YAML literals vs `float64` from standard JSON decoders). The [`numericEqual`](./internal/automation/match.go) helper validates that both operands are numeric (via [`isNumeric`](./internal/automation/match.go)), checks bidirectional type convertibility, and ensures equality after bidirectional conversion so that precision is not lost:
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
During initialization, [`compileMatcher`](./internal/automation/automation.go) detects `$yaml` (unmarshaling the multi-line string into Go values via `go.yaml.in/yaml/v3`) or `$plain` (validating scalar values for raw UTF-8 payload comparison).

### Pointer Dereferencing ([`dereference`](./internal/automation/automation.go))
Incoming payloads from subscriber callbacks may arrive as pointers or raw values. The [`dereference`](./internal/automation/automation.go) helper safely returns `nil` for typed nil pointers and unwraps non-nil `reflect.Pointer` instances into their underlying values before passing them to the matcher.

---

## Action Dispatch & Publisher Domain

### [`publisher`](./internal/automation/publisher.go)
An abstraction encapsulating reactive command dispatch to a destination MQTT topic:
```go
type publisher interface {
    Publish() error
}
```

### [`newPublisher`](./internal/automation/publisher.go) Factory & Content Operators
When instantiating action publishers from configuration, the content object is inspected to select the appropriate publisher implementation:

1. **`$plain` Operator ([`plainPublisher`](./internal/automation/publisher.go))**:
   Used for simple string or binary switch commands (e.g., sending `on` or `off` to smart plugs):
   ```yaml
   content:
     $plain: on
   ```
   Invokes [`topic.PublishString(fmt.Sprintf("%v", p.content))`](./internal/automation/publisher.go).

2. **`$json` Operator ([`jsonPublisher`](./internal/automation/publisher.go))**:
   Used for structured JSON payload dispatching:
   ```yaml
   content:
     $json:
       state: "active"
       brightness: 100
   ```
   Invokes [`topic.Publish(j.content)`](./internal/automation/publisher.go), serializing the data via `json.Marshal`.

3. **Default Fallback**: If no operator key is detected, content is dispatched through `jsonPublisher` by default.

### [`ActionConfig`](./config/config.go)
A configuration struct defining a target topic and the content payload to be published:
```go
type ActionConfig struct {
    Topic   string
    Content any
}
```

---

## Pub/Sub Messaging Domain

### [`mqtt.Broker`](./pkg/broker/mqtt/mqttBroker.go)
The concrete MQTT broker client backed by the Eclipse Paho MQTT client (`github.com/eclipse/paho.mqtt.golang`):
- `Run(ctx context.Context) error`: Connects to the broker, re-subscribes all registered handlers on every connection/reconnection, blocks until `ctx` is cancelled, and disconnects cleanly before returning.
- `Topic(topicName string) *Topic`: Returns a handle (`*mqtt.Topic`) for pub/sub operations on a named topic.

Configured via functional options:
- [`mqtt.BrokerURL(url)`](./pkg/broker/mqtt/options.go): Sets target broker address (e.g. `tcp://mosquitto:1883`).
- [`mqtt.ClientID(id)`](./pkg/broker/mqtt/options.go): Sets explicit MQTT client identifier when non-empty.
- [`mqtt.Credentials(user, pass)`](./pkg/broker/mqtt/options.go): Configures MQTT broker authentication credentials when non-empty.
- [`mqtt.TokenTimeout(d)`](./pkg/broker/mqtt/options.go): Overrides the default 10s wait timeout for publish/subscribe acknowledgement tokens.
- [`mqtt.TLS(cfg)`](./pkg/broker/mqtt/options.go) & [`mqtt.BuildTLSConfig(cfg)`](./pkg/broker/mqtt/tls.go): Attaches TLS 1.2+ / mTLS credentials (`Enabled`, `CAFile`, `CertFile`, `KeyFile`, `ServerName`, `InsecureSkipVerify`).
- [`mqtt.ClientFactory(fn)`](./pkg/broker/mqtt/options.go): Overrides Paho client creation for unit testing.
- [`mqtt.OnError(handler)`](./pkg/broker/mqtt/options.go): Registers an [`mqtt.ErrorHandler`](./pkg/broker/mqtt/error.go) for asynchronous [`*mqtt.Error`](./pkg/broker/mqtt/error.go) events (`OpSubscribe`, `OpPublish`, `OpUnmarshal`, including `ErrTokenTimeout` / `context.DeadlineExceeded`).

### [`mqtt.Topic`](./pkg/broker/mqtt/mqttBroker.go)
Provides publishing and subscription methods:
- `PublishString(s string)`: Sends an unencoded text message with QoS 1 (bounded by a 10s token wait timeout).
- `Publish(data any) error`: Encodes `any` data to JSON synchronously and sends with QoS 1.
- `Subscribe[T any](callback CallbackFunction[T])`: Subscribes with QoS 0, unmarshals the JSON payload into `T`, and invokes `callback(topic, payload)`.
- `SubscribeRaw(callback CallbackFunction[[]byte])`: Subscribes with QoS 0 and delivers the raw byte payload without JSON decoding.

---

## Configuration & Runtime Domain

### [`config.Config`](./config/config.go)
The root configuration struct loaded using a hybrid Viper (`KeyDelimiter("::")`, `ExperimentalBindStruct()`) + `yaml.v3` decoder:
```go
type Config struct {
    Logger   loggerConfig.Logger
    Mqtt     MqttConfig
    Automate map[string][]MatchConfig `mapstructure:"-"`
}
```
Viper decodes `Logger` and `Mqtt` (including `Mqtt.TLS`) with full environment-variable override support while avoiding dot-splitting; `Automate` is decoded directly via `go.yaml.in/yaml/v3` so mixed-case and dotted MQTT topics and inline `$json`/`match` keys remain intact.

### [`hidden.String`](./pkg/hidden/string.go) & [`hidden.URL`](./pkg/hidden/url.go)
Type-safe secret wrappers under [`pkg/hidden`](./pkg/hidden/string.go) prevent credential leakage across `fmt` (`%v`, `%+v`, `%#v`) and JSON logs (`MqttConfig.Password` and `MqttConfig.Broker`), exposing plaintext exclusively via `.Reveal()`.

### Environment Variable Mapping
The configuration loader sets `AUTOMQTTION` as the environment variable prefix and replaces the `::` key delimiter with underscores:
- `AUTOMQTTION_MQTT_BROKER`, `AUTOMQTTION_MQTT_CLIENTID`, `AUTOMQTTION_MQTT_USERNAME`, `AUTOMQTTION_MQTT_PASSWORD`
- `AUTOMQTTION_MQTT_TLS_ENABLED`, `AUTOMQTTION_MQTT_TLS_CAFILE`, `AUTOMQTTION_MQTT_TLS_CERTFILE`, `AUTOMQTTION_MQTT_TLS_KEYFILE`, `AUTOMQTTION_MQTT_TLS_SERVERNAME`, `AUTOMQTTION_MQTT_TLS_INSECURESKIPVERIFY`
- `AUTOMQTTION_LOGGER_LEVEL`, `AUTOMQTTION_LOGGER_ENCODING`, `AUTOMQTTION_LOGGER_DEVELOPMENT`

### Logging Infrastructure ([`pkg/log`](./pkg/log))
A structured logging wrapper around Uber Zap:
- Standard levels: `debug`, `info`, `warn`, `error`, `dpanic`, `panic`, `fatal`.
- Configurable encoders: `console` (human-readable) vs `json` (production telemetry).
- Development mode: Toggles caller reporting and stacktraces.

### Smart Home Protocol Bridging Scenarios
`automqttion` was designed to solve real-world heterogeneous smart home protocol bridging over MQTT.
For example, translating binary Local Control Network (LCN) status broadcasts into commands for Shelly WiFi smart relays:
- **Trigger**: An LCN module transmits a status report on topic `lcn/segment/0/target/4/` with payload `{"Src":34,"Cmd":104,"Payload":"MAQ="}`.
- **Evaluation**: The matcher checks that `Src == 34`, `Cmd == 104`, and `Payload == "MAQ="`.
- **Action**: Two separate actions fire, sending plain `on` strings to `shellyplug-lamp/command/switch:0` and `shellyplug-heater/command/switch:0`.
