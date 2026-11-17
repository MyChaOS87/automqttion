# automqttion System Architecture

This document describes the architectural layout, package dependency graph, reactive message processing pipeline, publisher polymorphism, and container deployment strategy of `automqttion`.

---

## Package DAG & Boundary Decoupling

The repository compiles a single binary from [`cmd/automqttion`](../cmd/automqttion/automqttion.go).
The architecture strictly enforces package boundaries: generic utility packages located under [`pkg/`](../pkg) never import application packages under [`cmd/`](../cmd), [`config/`](../config), or [`internal/`](../internal). This rule is enforced at build and lint time via `.golangci.yml` `depguard` settings.

```mermaid
flowchart TD
    subgraph CLI["CLI Entrypoint (cmd/)"]
        cmdEntry["cmd/automqttion"]
    end

    subgraph Internal["Application Internals (internal/ & config/)"]
        cfg["config (Config, LoadConfig, ParseConfig)"]
        cmdInit["internal/cmd (Init, Signals, NewMQTTBroker)"]
        auto["internal/automation (Automation, matcher, match, publisher)"]
    end

    subgraph Pkg["Reusable Platform Libraries (pkg/)"]
        mqttPkg["pkg/broker/mqtt (Paho MQTT Broker, Topic, TLS)"]
        hiddenPkg["pkg/hidden (Secret String & URL Masking)"]
        logPkg["pkg/log (Zap Logger wrapper)"]
        logCfg["pkg/log/config (Logger config struct)"]
    end

    cmdEntry --> cmdInit
    cmdEntry --> cfg
    cmdEntry --> auto
    cmdEntry --> mqttPkg
    cmdEntry --> logPkg

    cmdInit --> cfg
    cmdInit --> mqttPkg
    cmdInit --> logPkg

    auto --> cfg
    auto --> mqttPkg

    cfg --> hiddenPkg
    cfg --> logCfg

    mqttPkg --> logPkg

    logPkg --> logCfg

    classDef pkgClass fill:#e1f5fe,stroke:#0288d1,stroke-width:2px;
    classDef internalClass fill:#fff3e0,stroke:#f57c00,stroke-width:2px;
    classDef cliClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px;

    class cmdEntry cliClass;
    class cfg,cmdInit,auto internalClass;
    class mqttPkg,hiddenPkg,logPkg,logCfg pkgClass;
```

---

## Reactive Message Processing Sequence

[`cmd/automqttion`](../cmd/automqttion/automqttion.go) acts as an event-driven automation hub. When started, it registers an [`automation`](../internal/automation/automation.go) instance for each topic defined in `cfg.Automate`.

The sequence below illustrates the end-to-end lifecycle when an MQTT message arrives on a monitored topic:

```mermaid
sequenceDiagram
    autonumber
    participant External as External MQTT Publisher
    participant Broker as MQTT Broker (Mosquitto)
    participant MQTTTopic as Topic (pkg/broker/mqtt)
    participant Auto as automation (internal/automation)
    participant Matcher as match() (internal/automation)
    participant Pub as publisher (internal/automation)
    participant TargetTopic as Target Topic (pkg/broker/mqtt)

    External->>Broker: Publish message to topic (e.g. lcn/segment/0/target/4/ or switch/state)
    Broker->>MQTTTopic: Inbound MQTT packet
    MQTTTopic->>MQTTTopic: SubscribeRaw callback (raw []byte payload)
    MQTTTopic->>Auto: callback(topic, rawBytes)
    loop For each configured matcher
        alt $plain matcher
            Auto->>Matcher: rawStr == matcher.match
        else JSON / scalar matcher
            Auto->>Auto: Lazy json.Unmarshal(rawBytes) with string fallback
            Auto->>Matcher: match(matcher.match, dereference(payload))
        end
        alt Pattern Matches (subset & numeric equivalence)
            Matcher-->>Auto: true
            loop For each action publisher
                Auto->>Pub: Publish()
                alt $plain publisher
                    Pub->>TargetTopic: PublishString(content)
                    TargetTopic->>Broker: MQTT Publish (QoS 1, text string)
                else $json publisher
                    Pub->>TargetTopic: Publish(content)
                    TargetTopic->>Broker: MQTT Publish (QoS 1, JSON bytes)
                end
            end
        else No Match
            Matcher-->>Auto: false
        end
    end
```

---

## Declarative Pattern Matching Architecture

Pattern matching in [`internal/automation/match.go`](../internal/automation/match.go) is designed for declarative JSON tree inspection:

```
Incoming JSON Data                Target Rule Pattern
{                                {
  "Src": 34,         <=======>     "Src": 34,
  "Cmd": 104,        <=======>     "Cmd": 104,
  "Payload": "MAQ=", <=======>     "Payload": "MAQ="
  "Timestamp": 16999               // Unmatched extra fields ignored
}                                }
```

### 1. Structural Subset & Array Matching
- When the target pattern is a `map[string]any`, the matcher iterates across all keys present in the target. It requires each target key to exist in the data map (`val, exists := dat[k]`) and recursively evaluates [`match(v, val)`](../internal/automation/match.go). Additional fields in the incoming payload are ignored, allowing broad events to trigger targeted automations.
- When the target pattern is a `[]any`, the matcher requires an incoming `[]any` of identical length and evaluates [`match(tgt[i], dat[i])`](../internal/automation/match.go) element-wise.

### 2. Cross-Type Numeric Equality
In IoT architectures, numbers often undergo type shifts between producers, brokers, and parsers (e.g., an integer `104` in YAML vs `float64(104)` parsed from JSON).
The function [`numericEqual`](../internal/automation/match.go) avoids false negatives by:
- Verifying both values are numeric (`int`, `uint`, or `float` variants via [`isNumeric`](../internal/automation/match.go)).
- Converting operands bidirectionally via `reflect.Value.Convert`.
- Asserting equality in both directions to prevent loss of precision during narrowing conversions.

### 3. Matcher Pre-Compilation (`$yaml` & `$plain`)
During automation initialization in [`NewAutomation`](../internal/automation/automation.go), the helper [`compileMatcher`](../internal/automation/automation.go) scans each rule's match field:
- `$yaml`: Unmarshals the inline YAML string into Go values (`any` — maps, slices, or scalars) via `go.yaml.in/yaml/v3` upfront.
- `$plain`: Validates scalar values and pre-formats them via `fmt.Sprintf("%v", raw)` for exact raw UTF-8 string comparison against inbound MQTT payloads (`on`, `off`, `online`, `"1"`).

---

## Action Publisher Polymorphism

Publishers implement the [`publisher`](../internal/automation/publisher.go) interface and dispatch commands to an assigned [`*mqtt.Topic`](../pkg/broker/mqtt/mqttBroker.go):

```mermaid
classDiagram
    class publisher {
        <<interface>>
        +Publish() error
    }

    class commonPublisher {
        #topic *mqtt.Topic
    }

    class jsonPublisher {
        -content any
        +Publish() error
    }

    class plainPublisher {
        -content any
        +Publish() error
    }

    publisher <|.. jsonPublisher
    publisher <|.. plainPublisher
    commonPublisher <|-- jsonPublisher
    commonPublisher <|-- plainPublisher
```

- **[`plainPublisher`](../internal/automation/publisher.go)**: Selected when configuration contains `$plain`. Formats content with `fmt.Sprintf("%v", content)` and emits an unencoded string (ideal for device commands like `on`, `off`, `toggle`).
- **[`jsonPublisher`](../internal/automation/publisher.go)**: Selected when configuration contains `$json` or as default fallback. Marshals the Go structure into JSON before sending.

---

## Container Build & Kubernetes Deployment Strategy

### Multi-Stage `FROM scratch` Build
The production container is defined in [`build/automqttion/Dockerfile`](../build/automqttion/Dockerfile):

1. **Build Stage (`--platform=$BUILDPLATFORM golang:1.27-alpine`)**:
   - Creates an unprivileged `appuser` (`UID 10001`, `/sbin/nologin`).
   - Downloads modules and copies source code.
   - Cross-compiles natively on `$BUILDPLATFORM` (`CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v}`) using `-ldflags "-w -s -X main.Version=${APP_VERSION}"` without QEMU emulation.
   - Uses build cache mounts (`--mount=type=cache,target=/root/.cache/go-build`) for rapid compilation.
2. **Runtime Stage (`FROM scratch`)**:
   - Zero OS bloat, minimal attack surface, zero shell binaries.
   - Copies CA certificates (`/etc/ssl/certs/ca-certificates.crt`), timezone database (`/usr/share/zoneinfo`), and `/etc/passwd` + `/etc/group` from builder.
   - Copies the statically compiled binary `/dist/automqttion` and default configuration `/config/config.yml`.
   - Runs as unprivileged `USER appuser:appuser` with entrypoint `["/automqttion"]`.

### Kubernetes Helm Chart
The repository includes a production-ready singleton Helm chart under [`helm/charts/automqttion`](../helm/charts/automqttion):
- [`Chart.yaml`](../helm/charts/automqttion/Chart.yaml): Versioned application chart (`version: 0.1.0`, `appVersion: "0.1.0"`).
- [`values.yaml`](../helm/charts/automqttion/values.yaml): Default values enforcing singleton `replicaCount: 1` (standard MQTT v3.1.1 subscriptions duplicate actions across multiple replicas), `strategy.type: Recreate`, `image.repository: ghcr.io/mychaos87/automqttion`, and hardened non-root security contexts (`UID 10001`, `readOnlyRootFilesystem: true`, `capabilities.drop: [ALL]`).
- [`templates/configmap.yaml`](../helm/charts/automqttion/templates/configmap.yaml): Optional ConfigMap supporting raw YAML strings or structured mappings mounted read-only at `/config` when `.Values.config` is provided.
- [`templates/deployment.yaml`](../helm/charts/automqttion/templates/deployment.yaml): Standard singleton Kubernetes Deployment manifest with config checksum pod annotation and `Recreate` rollout strategy.
- [`templates/serviceaccount.yaml`](../helm/charts/automqttion/templates/serviceaccount.yaml): Dedicated ServiceAccount for RBAC isolation.
