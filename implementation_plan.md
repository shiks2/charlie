# OCPP 1.6 Gateway Production-Readiness Evaluation

Evaluation of the current Charlie Gateway implementation based on the 4 production pillars.

## Critical Risks

> [!CAUTION]
> **Map Race Conditions**: `KafkaEmitter.getWriter` modifies a shared map without a mutex. Under load (500+ connections), the gateway **will panic** due to concurrent map writes.

> [!WARNING]
> **Blocking I/O in WebSocket Handlers**: The `MultiEmitter` and `KafkaEmitter` are synchronous. If Kafka experiences latency or downtime, the OCPP WebSocket handler will block, causing mass disconnections of charge points due to heartbeat timeouts.

> [!WARNING]
> **Ephemeral State (Data Loss)**: The `MemoryStore` loses all active transaction context on restart. 'Incomplete Transactions' cannot be properly reconciled if the gateway doesn't persist the TransactionID-to-Charger mapping.

## Evaluation Results

### 1. Concurrency Safety
- **MemoryStore**: Uses `sync.RWMutex`, which is a good start. However, `GetAllChargers` leaks pointers to internal data structures, allowing external races.
- **KafkaEmitter**: **NOT thread-safe**. Map access in `getWriter` is unprotected.

### 2. Resiliency
- **Blocking Multi-Emitter**: Sequential execution means a slow Kafka broker kills the JSON/Log output too.
- **No Backpressure/Buffering**: There is no buffer between the OCPP handler and the Emitters. Direct synchronous emission to Kafka in a high-concurrency environment is a bottleneck.

### 3. Observability
- **Missing Metrics**: Currently, only `fmt.Printf` and `log.Printf` are used. 
- **Gaps**: Missing Prometheus histograms for `message_processing_latency`, counters for `kafka_emission_failures`, and gauges for `connected_charge_points`.

### 4. Protocol Integrity
- **Transaction ID Collisions**: Generating IDs via `UnixNano() % 1000000` is prone to collisions and is not production-grade.
- **Persistence Gap**: Restarting the gateway wipes the `MemoryStore`. When a truck stops charging after a restart, the gateway may not recognize the `transactionId` provided by the truck, leading to "Missing Transaction" errors on the B2B side.

---

## 4-Week Roadmap

### Week 1: Concurrency & Safety (Critical Fixes)
- [ ] Implement `sync.Mutex` in `KafkaEmitter` for writer management.
- [ ] Refactor `MemoryStore` to return deep copies or interfaces to prevent pointer leakage.
- [ ] Introduce an **Asynchronous Emitter Pattern** using Go channels and worker pools to decouple OCPP handlers from Kafka I/O.

### Week 2: Resiliency & Persistence
- [ ] Integrate **Redis** or a database to persist active `TransactionID` mappings.
- [ ] Implement handled reconciliation: On `BootNotification`, check for ongoing transactions that mismatch local state.
- [ ] Add **Kafka Retries** and a "Circuit Breaker" to prevent downstream failures from cascading to the WebSocket layer.

### Week 3: Observability (Monitoring)
- [ ] Integrate `prometheus/client_golang`.
- [ ] Export `ocpp_charge_point_uptime` (Gauge tracked by Heartbeats).
- [ ] Export `ocpp_message_latency_seconds` (Histogram).
- [ ] Replace `log` with a structured logger (e.g., `uber-go/zap`) for better ELK/Grafana integration.

### Week 4: Protocol Hardening & Load Testing
- [ ] Implement a coordinated `TransactionID` generator (Postgres Serial or UUID).
- [ ] Add `Offline` state handling in the mapper for when heartbeats are missed.
- [ ] Run load tests with 500+ simulated connections (using a tool like `k6` with a WebSocket plugin or custom Go scripts).

## Open Questions
- Do we have a budget/infrastructure for Redis/Postgres, or should we use a local KV store (BoltDB/LevelDB) for Transaction persistence?
- Does the B2B platform require exactly-once delivery, or is at-least-once (with deduplication) acceptable?
