CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE branches (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    timezone text NOT NULL DEFAULT 'UTC'
);
CREATE TABLE ingredients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    name text NOT NULL,
    unit text NOT NULL CHECK (unit IN ('kg', 'litre', 'unit')),
    reorder_point numeric(12,3) NOT NULL DEFAULT 10 CHECK (reorder_point >= 0),
    UNIQUE(branch_id, name)
);
CREATE TABLE suppliers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    name text NOT NULL,
    email text NOT NULL
);
CREATE TABLE menu_items (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    name text NOT NULL,
    available boolean NOT NULL DEFAULT true,
    price numeric(12,2) NOT NULL CHECK (price >= 0)
);
CREATE TABLE recipes (
    menu_item_id uuid REFERENCES menu_items,
    ingredient_id uuid REFERENCES ingredients,
    quantity numeric(12,3) NOT NULL CHECK (quantity > 0),
    PRIMARY KEY(menu_item_id, ingredient_id)
);
CREATE TABLE purchase_orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    supplier_id uuid NOT NULL REFERENCES suppliers,
    ingredient_id uuid NOT NULL REFERENCES ingredients,
    quantity numeric(12,3) NOT NULL CHECK (quantity > 0),
    received_quantity numeric(12,3) NOT NULL DEFAULT 0 CHECK (received_quantity >= 0 AND received_quantity <= quantity),
    status text NOT NULL DEFAULT 'ordered' CHECK (status IN ('ordered','partial','fulfilled')),
    external_status text NOT NULL DEFAULT 'pending',
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK ((status = 'fulfilled') = (received_quantity = quantity))
);
CREATE TABLE deliveries (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_order_id uuid NOT NULL REFERENCES purchase_orders,
    quantity numeric(12,3) NOT NULL CHECK (quantity > 0),
    source_reference text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE inventory_movements (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    ingredient_id uuid NOT NULL REFERENCES ingredients,
    quantity numeric(12,3) NOT NULL CHECK (quantity <> 0),
    kind text NOT NULL CHECK (kind IN ('opening','receipt','consumption','waste','transfer')),
    reference text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX inventory_balance_idx ON inventory_movements(branch_id, ingredient_id);
CREATE TABLE orders (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    menu_item_id uuid NOT NULL REFERENCES menu_items,
    quantity integer NOT NULL CHECK (quantity > 0),
    status text NOT NULL DEFAULT 'preparing',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE demand_history (
    branch_id uuid NOT NULL REFERENCES branches,
    ingredient_id uuid NOT NULL REFERENCES ingredients,
    date date NOT NULL,
    service text NOT NULL CHECK (service IN ('lunch','dinner')),
    consumption numeric(12,3) NOT NULL CHECK (consumption >= 0),
    PRIMARY KEY(branch_id, ingredient_id, date, service)
);
CREATE TABLE kitchen_metrics (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    prep_minutes double precision NOT NULL CHECK (prep_minutes > 0),
    cancellation_rate double precision NOT NULL CHECK (cancellation_rate BETWEEN 0 AND 1),
    order_count integer NOT NULL CHECK (order_count > 0),
    recorded_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE incidents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    purchase_order_id uuid REFERENCES purchase_orders,
    fingerprint text NOT NULL,
    title text NOT NULL,
    severity text NOT NULL CHECK (severity IN ('medium','high','critical')),
    status text NOT NULL DEFAULT 'detected' CHECK (status IN ('detected','investigating','awaiting_approval','approved','executing','verifying','resolved','escalated')),
    diagnosis jsonb NOT NULL DEFAULT '{}',
    recovery jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(branch_id, fingerprint)
);
CREATE TABLE actions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    incident_id uuid NOT NULL REFERENCES incidents,
    type text NOT NULL CHECK (type IN ('replacement_stock')),
    parameters jsonb NOT NULL,
    status text NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed','approved','rejected','executed','received')),
    approved_by text,
    approved_at timestamptz,
    executed_at timestamptz,
    received_at timestamptz,
    UNIQUE(incident_id, type)
);
CREATE TABLE stock_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    action_id uuid NOT NULL UNIQUE REFERENCES actions,
    branch_id uuid NOT NULL REFERENCES branches,
    ingredient_id uuid NOT NULL REFERENCES ingredients,
    quantity numeric(12,3) NOT NULL CHECK (quantity > 0),
    status text NOT NULL DEFAULT 'requested' CHECK (status IN ('requested','received')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE audit_log (
    id bigserial PRIMARY KEY,
    branch_id uuid NOT NULL REFERENCES branches,
    incident_id uuid REFERENCES incidents,
    actor text NOT NULL,
    event text NOT NULL,
    details jsonb NOT NULL DEFAULT '{}',
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE outbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);
CREATE INDEX outbox_pending_idx ON outbox(created_at) WHERE published_at IS NULL;
CREATE TABLE idempotency_keys (
    branch_id uuid NOT NULL REFERENCES branches,
    key text NOT NULL,
    request_hash text NOT NULL,
    response jsonb NOT NULL,
    PRIMARY KEY(branch_id, key)
);
CREATE TABLE processed_events (
    consumer text NOT NULL,
    event_id uuid NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY(consumer, event_id)
);
CREATE TABLE knowledge (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id uuid NOT NULL REFERENCES branches,
    source text NOT NULL,
    content text NOT NULL,
    embedding vector(384),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(branch_id, source)
);
CREATE INDEX knowledge_embedding_idx ON knowledge USING hnsw (embedding vector_cosine_ops);
