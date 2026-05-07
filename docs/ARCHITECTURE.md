# Workflow and reliability model

## Ownership

- **Go** owns restaurant business data, stock arithmetic, authorizations, actions, audit records, and event emission.
- **Python** owns forecasting, incident detection, evidence reasoning, retrieval, and workflow checkpoints.
- **PostgreSQL** stores both domains. Python only writes graph checkpoints, event-consumption receipts, and its knowledge index directly; operational mutations use Go APIs.
- **Kafka** delivers operational notifications. A periodic database/API reconciliation pass also recovers incomplete workflows after restarts.

## Delivery pipeline

Go commits a business transaction and an outbox event in the same PostgreSQL transaction. The publisher claims pending outbox rows with `FOR UPDATE SKIP LOCKED`, waits for Kafka acknowledgment, and then marks them published.

A crash after publishing but before marking can publish a duplicate. This is deliberately **at-least-once**, not exactly-once. Python records processed event IDs, Go deduplicates incidents by branch/fingerprint, and all writes use idempotency keys. Kafka offsets are manually committed after processing. An event-triggered graph failure is retried by persisted checkpoint reconciliation, even when its notification has been acknowledged.

The local worker serializes graph work with one asyncio lock. Horizontal worker scaling requires a distributed per-incident lease/advisory lock before invoking the same thread.

## Inventory model

The ledger is append-only through the exposed APIs:

```text
balance = opening + received deliveries + actual replacement receipts
          - recipe consumption - recorded waste
```

Ingredient locks serialize stock deductions. Receipt retries are serialized by an idempotency-key advisory lock. Over-receipts and negative-stock preparation/waste are rejected. Replacement receipts preserve the original supplier shortfall rather than rewriting the supplier's fulfillment history.

## LangGraph

```text
begin → collect → reason → publish → plan
                                      ├── insufficient evidence → escalated → END
                                      └── await_approval [interrupt]
                                               ├── rejected → END
                                               └── execute
                                                     ├── stale approved action → escalate → END
                                                     └── verify
                                                          ├── pending → interrupt → verify
                                                          ├── recovered → resolve → remember → END
                                                          └── two-hour timeout → reopen → re-evaluate → escalate → END
```

`AsyncPostgresSaver` persists graph state and interrupts under a branch-scoped incident thread ID. Approval events resume the thread, but the node checks Go's stored action state rather than trusting the event payload. A 15-second reconciler covers restarts, missing notifications, and observation deadlines.

The LLM is optional. With a key, LangChain exposes read-only evidence, forecast, and knowledge tools for at most three tool rounds, four calls per round. Structured narrative is stored alongside deterministic hypothesis assessments. Quantity calculation and authorization remain outside the LLM.

## Retrieval

SOPs and resolved cases are embedded by the same local 384-dimensional model and stored in pgvector. Cosine distance ranks documents within the branch. Sources are attached to the diagnosis. Similarity suggests relevant context; it is not proof of the current root cause. Live numerical facts come from Go's consistent evidence snapshot.

Model loading happens asynchronously so the system can start while embeddings download. Lexical retrieval is explicitly reported until vector indexing is ready, or if model initialization fails.

## Recovery conditions

The Python verifier and Go resolution gate both require:

1. A recorded replacement receipt linked to this incident's approved action.
2. Positive ingredient balance.
3. Three new measurement windows after receipt, totaling at least 30 orders.
4. Minimum observation duration: 600 seconds normally; 10 seconds in the provided demo configuration.
5. All of the latest three windows show preparation at or below 18 minutes and cancellations at or below 5%.
6. The newest sample is less than 15 minutes old.

Recovery is operational improvement after intervention, not a causal experiment. Baseline- and demand-adjusted causal evaluation is outside this MVP.

## Deliberate extension points

- Replace the internal stock-request adapter with contracted supplier/inter-branch connectors.
- Aggregate live preparation events into demand history and kitchen measurement windows.
- Add anomaly records and detectors for unusual waste, stock count reconciliation, forecast-adjusted demand spikes, and kitchen station delays.
- Expand single-line purchase orders, recipe versions, conversions/yields, stock counts, and partial replacement receiving.
- Add multi-tenant membership, per-user identity, and per-branch authorization instead of the configured local branch.
- Add distributed workflow leases, poison-event handling, retention, paging, and production migrations.
