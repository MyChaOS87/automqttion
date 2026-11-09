# automqttion Coding Idioms & Conventions

This document records the coding style, Go language idioms, declarative operator patterns, error handling rules, and commit standards across `automqttion`.

---

## Go 1.27 Modernization & Language Idioms

`automqttion` targets Go 1.27 and enforces modern standard library capabilities:

- **AST Modernization via `go fix`**: Run `go fix ./...` to adopt standard library improvements and AST migrations. `make lint` enforces `go fix -diff ./...`. Discrepancies must be fixed using `make lint.fix` (or `make fix`).
- **Standard Library Primitives**: Use standard library primitives over external packages:
  - Use stdlib `errors` (`errors.Is`, `errors.As`, `errors.AsType`, `fmt.Errorf`) instead of `github.com/pkg/errors` (blocked by `gomodguard_v2`).
  - Use stdlib `slices` and built-in `min`/`max` functions instead of hand-rolled loops.
  - Test context creation must use `t.Context()` instead of `context.Background()` or `context.TODO()` in tests (enforced by `usetesting`).
- **Forbidden Dependencies (`gomodguard_v2`)**:
  - `github.com/pkg/errors` is forbidden; use the standard library `errors` package.
  - `github.com/google/uuid` is forbidden; use Go 1.27 standard library or internal implementations.
- **Loop Variable Scoping**: Go 1.22+ per-iteration variable scoping is active. Do not add redundant `tt := tt` redeclarations in subtests.

---

## Declarative Operator Conventions

The configuration and automation pipeline uses dollar-prefixed keys (`$yaml`, `$plain`, `$json`) to denote transformation operators:

### 1. `$yaml` Pattern Expander
Used inside `match:` blocks in configuration to write structured payload matchers as readable YAML blocks rather than JSON strings:
```yaml
match:
  $yaml: |
    Src: 34
    Cmd: 104
    Payload: "MAQ="
```
- **Convention**: The value of `$yaml` must strictly be a string containing valid YAML.
- **Handling**: Unmarshaled into a `map[string]any` during automation initialization so that runtime matching operates on in-memory Go maps without runtime string parsing.

### 2. `$plain` Action Content Operator
Used inside `content:` blocks to signal unencoded string transmission:
```yaml
actions:
  - topic: home/livingroom/light/set
    content:
      $plain: on
```
- **Convention**: Emits the value converted to string via `fmt.Sprintf("%v", content)` through [`topic.PublishString`](../pkg/broker/mqtt/mqttBroker.go).

### 3. `$json` Action Content Operator
Used inside `content:` blocks to signal structured JSON marshaling:
```yaml
actions:
  - topic: home/livingroom/thermostat/set
    content:
      $json:
        target_temperature: 21.5
        mode: auto
```
- **Convention**: Dispatches the content structure through [`topic.Publish`](../pkg/broker/mqtt/mqttBroker.go) which serializes via `json.Marshal`.
- **Default Fallback**: If neither `$plain` nor `$json` is present in a map, the publisher defaults to JSON marshaling.

---

## Safe Reflection Standards

Reflection is used in [`internal/automation`](../internal/automation) for dynamic message matching and pointer dereferencing:

1. **Kind Validation Before Conversion**: Always verify that `reflect.Value.CanConvert` returns `true` before calling `.Convert()`.
2. **Bidirectional Numeric Checks**: When checking numeric equality between disparate types (e.g., `int` and `float64`), convert in both directions:
   ```go
   av.Convert(bv.Type()).Interface() == bv.Interface() &&
   bv.Convert(av.Type()).Interface() == av.Interface()
   ```
   This prevents lossy truncation (such as `1.2` truncating to `1`) from falsely reporting a match.
3. **Map Iteration**: Use `val.MapRange()` instead of allocating intermediate key slices via `.MapKeys()`.
4. **Pointer Dereferencing**: Always use [`dereference`](../internal/automation/automation.go) to unwrap pointers before inspecting interface values.

---

## Error Handling & Fatal Exit Elimination

### Sentinel Error Wrapping (`%w`)
All domain-level errors must be exposed as sentinel errors or wrap underlying errors with `%w` for deterministic inspection via `errors.Is`:
```go
var ErrConfigFileNotFound = errors.New("config file not found")
```
When wrapping lower-level I/O or parser failures:
```go
return nil, fmt.Errorf("failed to read config: %w", err)
```

### No `log.Fatal` in Library Code
**Rule**: Libraries and sub-packages must **never** call `log.Fatal`, `log.Fatalf`, or `os.Exit`. Constructors and parsers (such as [`NewAutomation`](../internal/automation/automation.go), [`replaceIfYamlMatcher`](../internal/automation/automation.go), and [`cmd.Init`](../internal/cmd/init.go)) return typed `error` values up the call stack so the CLI entrypoint in `main()` handles process termination cleanly.

---

## Functional Options Pattern

Component configuration in `pkg/` follows the functional options pattern for extensible, zero-allocation setup:

```go
type (
    Option func(*Config)
    Config struct {
        clientOptions *mqtt.ClientOptions
        clientFactory func(*mqtt.ClientOptions) mqtt.Client
        errorHandler  ErrorHandler
    }
)

func BrokerURL(broker string) Option {
    return func(c *Config) {
        c.clientOptions.AddBroker(broker)
    }
}

func TLS(tls *tls.Config) Option {
    return func(c *Config) {
        c.clientOptions.TLSConfig = tls
    }
}

func NewBroker(options ...Option) *Broker {
    config := newDefaultConfig()
    for _, opt := range options {
        opt(config)
    }
    b := &Broker{
        errorHandler: config.errorHandler,
    }
    config.clientOptions.SetOnConnectHandler(b.onConnect)
    config.clientOptions.SetConnectionLostHandler(b.onConnectionLost)
    b.client = config.clientFactory(config.clientOptions)
    return b
}
```

---

## Configuration & Viper Mapping

Configuration is loaded using Viper in [`config/config.go`](../config/config.go):
- **Environment Prefix**: `AUTOMQTTION`
- **Key Delimiter**: Standard dots (`.`) are mapped to underscores (`_`) using `strings.NewReplacer(".", "_")`.
- **Precedence**: Environment variables override configuration file values automatically via `v.AutomaticEnv()`.
- **Search Paths**: The loader searches `./config/config.{yml,yaml,json}` relative to working directory.

---

## Commit Message Conventions

All commits across the workspace must follow the standardized format:
```
feat[(<topic>)]: <description>
chore[(<topic>)]: <description>
bugfix[(<topic>)]: <description>
```
- Use active present tense ("add matcher test" instead of "added matcher test").
- One heading only.
- Use `*` for bullet points in commit bodies.
- Escape backticks in shell commands.
