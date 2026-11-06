# automqttion System Architecture

This document describes the architectural layout, package dependency graph, reactive message processing pipeline, publisher polymorphism, and container deployment strategy of `automqttion`.

---

## Package DAG & Boundary Decoupling

The repository compiles a single binary from [`cmd/automqttion`](file:///home/kasch/projects/go/automqttion/cmd/automqttion/automqttion.go).
The architecture strictly enforces package boundaries: generic utility packages located under [`pkg/`](file:///home/kasch/projects/go/automqttion/pkg) never import application packages under [`cmd/`](file:///home/kasch/projects/go/automqttion/cmd), [`config/`](file:///home/kasch/projects/go/automqttion/config), or [`internal/`](file:///home/kasch/projects/go/automqttion/internal). This rule is enforced at build and lint time via `.golangci.yml` `depguard` settings.

```mermaid
flowchart TD
    subgraph CLI["CLI Entrypoint (cmd/)"]
        cmdEntry["cmd/automqttion"]
    end

    subgraph Internal["Application Internals (internal/ & config/)"]
        cfg["config (Config, LoadConfig, ParseConfig)"]
        cmdInit["internal/cmd (Init, Signals, Logger setup)"]
        auto["internal/automation (Automation, matcher, match, publisher)"]
    end

    subgraph Pkg["Reusable Platform Libraries (pkg/)"]
        brokerPkg["pkg/broker (Broker & Topic interfaces)"]
        mqttPkg["pkg/broker/mqtt (Paho MQTT Client & Options)"]
        nullPkg["pkg/broker/null (No-op NullBroker)"]
        logPkg["pkg/log (Zap Logger wrapper)"]
        logCfg["pkg/log/config (Logger config struct)"]
    end

    cmdEntry --> cmdInit
    cmdEntry --> cfg
    cmdEntry --> auto
    cmdEntry --> brokerPkg
    cmdEntry --> mqttPkg
    cmdEntry --> logPkg

    cmdInit --> cfg
    cmdInit --> logPkg

    auto --> cfg
    auto --> brokerPkg
    auto --> logPkg

    cfg --> logCfg

    mqttPkg --> brokerPkg
    mqttPkg --> logPkg

    nullPkg --> brokerPkg

    logPkg --> logCfg

    classDef pkgClass fill:#e1f5fe,stroke:#0288d1,stroke-width:2px;
    classDef internalClass fill:#fff3e0,stroke:#f57c00,stroke-width:2px;
    classDef cliClass fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px;

    class cmdEntry cliClass;
    class cfg,cmdInit,auto internalClass;
    class brokerPkg,mqttPkg,nullPkg,logPkg,logCfg pkgClass;
```

---

## Reactive Message Processing Sequence

[`cmd/automqttion`](file:///home/kasch/projects/go/automqttion/cmd/automqttion/automqttion.go) acts as an event-driven automation hub. When started, it registers an [`automation`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L23) instance for each topic defined in `cfg.Automate`.

The sequence below illustrates the end-to-end lifecycle when an MQTT message arrives on a monitored topic:

```mermaid
sequenceDiagram
    autonumber
    participant External as External MQTT Publisher
    participant Broker as MQTT Broker (Mosquitto)
    participant MQTTTopic as mqttTopic (pkg/broker/mqtt)
    participant Auto as automation (internal/automation)
    participant Matcher as match() (internal/automation)
    participant Pub as publisher (internal/automation)
    participant TargetTopic as Target mqttTopic (pkg/broker/mqtt)

    External->>Broker: Publish message to topic (e.g. lcn/segment/0/target/4/)
    Broker->>MQTTTopic: Inbound MQTT packet
    MQTTTopic->>MQTTTopic: SubscribeRawJSON callback (unmarshal JSON payload)
    MQTTTopic->>Auto: callback(topic, rawPayload)
    Auto->>Auto: dereference(rawPayload)
    loop For each configured matcher
        Auto->>Matcher: match(matcher.match, dereferencedPayload)
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

Pattern matching in [`internal/automation/match.go`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go) is designed for declarative JSON tree inspection:

```
Incoming JSON Data                Target Rule Pattern
{                                {
  "Src": 34,         <=======>     "Src": 34,
  "Cmd": 104,        <=======>     "Cmd": 104,
  "Payload": "MAQ=", <=======>     "Payload": "MAQ="
  "Timestamp": 16999               // Unmatched extra fields ignored
}                                }
```

### 1. Structural Subset Matching
When the target pattern is a `map[string]any`, the matcher iterates across all keys present in the target. It requires each target key to exist in the data map and recursively evaluates [`match(v, dat[k])`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L13). Additional fields in the incoming payload are ignored, allowing broad events to trigger targeted automations.

### 2. Cross-Type Numeric Equality
In IoT architectures, numbers often undergo type shifts between producers, brokers, and parsers (e.g., an integer `104` in YAML vs `float64(104)` parsed from JSON).
The function [`numericEqual`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L36) avoids false negatives by:
- Verifying both values are numeric (`int`, `uint`, or `float` variants via [`isNumeric`](file:///home/kasch/projects/go/automqttion/internal/automation/match.go#L26)).
- Converting operands bidirectionally via `reflect.Value.Convert`.
- Asserting equality in both directions to prevent loss of precision during narrowing conversions.

### 3. YAML Pre-Compilation (`$yaml`)
During automation initialization in [`NewAutomation`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L87), the helper [`replaceIfYamlMatcher`](file:///home/kasch/projects/go/automqttion/internal/automation/automation.go#L42) scans each rule's match field. If a map contains `$yaml`, the inline string block is unmarshaled into `map[string]any` upfront. Rule evaluation at runtime occurs purely between in-memory Go maps with zero string-parsing overhead.

---

## Action Publisher Polymorphism

Publishers implement the [`publisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L10) interface and dispatch commands to an assigned [`broker.Topic`](file:///home/kasch/projects/go/automqttion/pkg/broker/broker.go#L10):

```mermaid
classDiagram
    class publisher {
        <<interface>>
        +Publish()
    }

    class commonPublisher {
        #topic broker.Topic
    }

    class jsonPublisher {
        -content any
        +Publish()
    }

    class plainPublisher {
        -content any
        +Publish()
    }

    publisher <|.. jsonPublisher
    publisher <|.. plainPublisher
    commonPublisher <|-- jsonPublisher
    commonPublisher <|-- plainPublisher
```

- **[`plainPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L39)**: Selected when configuration contains `$plain`. Formats content with `fmt.Sprintf("%v", content)` and emits an unencoded string (ideal for device commands like `on`, `off`, `toggle`).
- **[`jsonPublisher`](file:///home/kasch/projects/go/automqttion/internal/automation/publisher.go#L29)**: Selected when configuration contains `$json` or as default fallback. Marshals the Go structure into JSON before sending.

---

## Container Build & Kubernetes Deployment Strategy

### Multi-Stage `FROM scratch` Build
The production container is defined in [`build/automqttion/Dockerfile`](file:///home/kasch/projects/go/automqttion/build/automqttion/Dockerfile):

1. **Build Stage (`golang:1.27-alpine`)**:
   - Downloads modules and copies source code.
   - Compiles a static Go binary using `-ldflags "-X main.Version=$APP_VERSION -extldflags '-static'"`.
   - Uses build cache mounts (`--mount=type=cache,target=/root/.cache/go-build`) for rapid compilation.
2. **Runtime Stage (`FROM scratch`)**:
   - Zero OS bloat, minimal attack surface, zero shell binaries.
   - Copies CA certificates from builder (`/etc/ssl/certs/ca-certificates.crt`) for secure TLS connections.
   - Copies timezone database (`/usr/share/zoneinfo`) for timezone-aware operations.
   - Copies the statically compiled binary `/dist/automqttion` and default configuration `/config/config.yml`.
   - Sets entrypoint to `["/automqttion"]`.

### Kubernetes Helm Chart
The repository includes a production-ready Helm chart under [`helm/charts/automqttion`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion):
- [`Chart.yaml`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion/Chart.yaml): Versioned application chart (`version: 0.0.2`, `appVersion: "main"`).
- [`values.yaml`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion/values.yaml): Default values specifying image repository `ghcr.io/mychaos87/automqttion`, resource limits, pod security contexts, and replicas.
- [`templates/deployment.yaml`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion/templates/deployment.yaml): Standard Kubernetes Deployment manifest with readiness and liveness lifecycle definitions.
- [`templates/serviceaccount.yaml`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion/templates/serviceaccount.yaml): Dedicated ServiceAccount for RBAC isolation.
- [`templates/hpa.yaml`](file:///home/kasch/projects/go/automqttion/helm/charts/automqttion/templates/hpa.yaml): Optional HorizontalPodAutoscaler for high-throughput MQTT workloads.
