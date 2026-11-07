# automqttion

`automqttion` is an event-driven, declarative MQTT automation daemon engineered to bridge heterogeneous smart home devices and execute reactive message pipelines across MQTT brokers.

When smart home protocols (such as Local Control Network, Zigbee, or Matter) publish status telemetry to an MQTT broker, `automqttion` listens on specified topics, evaluates structural match conditions against the message payload, and dispatches reactive actions (such as turning smart plugs on or off) to target topics.

---

## Key Features

- **Declarative Rule Pipeline**: Configure multi-topic reactive automations directly in YAML without writing glue code.
- **Structural Subset Pattern Matching**: Incoming JSON payloads are matched against target patterns; incoming payloads can contain extra fields while matching required values.
- **Type-Tolerant Numeric Comparison**: Avoids false negatives between JSON `float64` numbers and YAML integers via bidirectional conversion checking.
- **Embedded YAML Matcher Blocks (`$yaml`)**: Write complex multi-field payload matchers as readable YAML blocks instead of escaped JSON strings.
- **Action Content Operators**:
  - `$plain`: Publishes unencoded plain-text commands (e.g., `on`, `off`, `toggle`) to device topics.
  - `$json`: Marshals Go objects into structured JSON packets.
- **Minimal `FROM scratch` Container**: Multi-stage build producing a self-contained, statically linked binary with CA certificates and zero shell overhead.
- **Kubernetes Helm Chart**: Production Helm chart under `helm/charts/automqttion` supporting Deployment, ServiceAccount, and HorizontalPodAutoscaler resources.

---

## Configuration Example

The daemon is configured via [`config/config.yml`](./config/config.yml). Below is an example translating LCN bus status telegrams into Shelly switch commands:

```yaml
---
mqtt:
  broker: tcp://mosquitto.internal:1883

automate:
  # Inbound topic to monitor:
  lcn/segment/0/target/4/:
    # Rule 1: Match light active status and switch appliances ON
    - match:
        $yaml: |
          Src: 34
          Cmd: 104
          Payload: "MAQ="
      actions:
        - topic: shellyplusplugs-kaffeelicht/command/switch:0
          content:
            $plain: on
        - topic: shellyplusplugs-spühle/command/switch:0
          content:
            $plain: on

    # Rule 2: Match light inactive status and switch appliances OFF
    - match:
        $yaml: |
          Src: 34
          Cmd: 104
          Payload: "MAA="
      actions:
        - topic: shellyplusplugs-kaffeelicht/command/switch:0
          content:
            $plain: off
        - topic: shellyplusplugs-spühle/command/switch:0
          content:
            $plain: off

logger:
  development: true
  encoding: console
  level: info
```

---

## Quickstart

### Running Locally
```bash
# Build the binary
make build

# Run the automation daemon
go run ./cmd/automqttion
```

### Running with Docker Compose
```bash
docker compose up -d
```

### Kubernetes Deployment via Helm
```bash
helm install automqttion ./helm/charts/automqttion \
  --set image.repository=ghcr.io/mychaos87/automqttion \
  --set image.tag=latest
```

---

## Disclaimer
`automqttion` is pre-v1 software undergoing active development. Configuration schemas and matching semantics are subject to refinement without backwards-compatibility shims.

---

## License
Licensed under the **GNU General Public License v3.0 (GPL-3.0)**. See [`LICENSE.MD`](./LICENSE.MD) for full details.

---

## Documentation & Development

For technical specifications, architecture diagrams, coding idioms, testing guidelines, and toolchain rules, refer to the project documentation:

- [Agent Guide (`AGENTS.md`)](./AGENTS.md): Project overview, linear-main workflow, sandbox execution matrix, and agent hazards.
- [Ubiquitous Language & Domain Glossary (`CONTEXT.md`)](./CONTEXT.md): Core domain concepts, subset matching, numeric equality, `$yaml` expander, publishers, and broker interfaces.
- [System Architecture (`docs/ARCHITECTURE.md`)](./docs/ARCHITECTURE.md): Package DAG, Mermaid sequence diagrams, matching mechanics, publisher polymorphism, and container architecture.
- [Coding Conventions (`docs/CONVENTIONS.md`)](./docs/CONVENTIONS.md): Go 1.27 idioms, declarative operators, reflection standards, sentinel errors, and functional options.
- [Testing Standards (`docs/TESTING.md`)](./docs/TESTING.md): Output-oriented tabular tests, 3-tier testing model, and mandatory test mechanics.
- [Developer Toolchain (`docs/TOOLCHAIN.md`)](./docs/TOOLCHAIN.md): Makefile targets, isolated tool modules, vulnerability scanning, and licensing compliance.
