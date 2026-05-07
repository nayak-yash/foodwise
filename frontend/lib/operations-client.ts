import type {
  AgentStatus,
  Catalog,
  DashboardData,
  DemandForecast,
  Incident,
  InventoryItem,
  Overview,
  PurchaseOrder,
  StockRequest,
} from "./types";

/** Browser-side client. Credentials remain inside the authenticated Next.js proxy. */
export class OperationsClient {
  constructor(private readonly baseUrl = "/api/ops") {}

  async request<T>(path: string, body?: unknown, key?: string): Promise<T> {
    const response = await fetch(`${this.baseUrl}/${path}`, {
      method: body === undefined ? "GET" : "POST",
      cache: "no-store",
      headers: {
        "Content-Type": "application/json",
        ...(body === undefined ? {} : { "Idempotency-Key": key || crypto.randomUUID() }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
    });
    const data = await response.json();
    if (!response.ok) throw new Error(data.error || data.detail || "Request failed");
    return data as T;
  }

  async dashboard(): Promise<DashboardData> {
    const [overview, inventory, incidents, purchaseOrders, requests, catalog] = await Promise.all([
      this.request<Overview>("overview"),
      this.request<InventoryItem[]>("inventory"),
      this.request<Incident[]>("incidents"),
      this.request<PurchaseOrder[]>("purchase-orders"),
      this.request<StockRequest[]>("stock-requests"),
      this.request<Catalog>("catalog"),
    ]);
    return { overview, inventory, incidents, purchaseOrders, requests, catalog };
  }

  incident(id: string) {
    return this.request<Incident>(`incidents/${id}`);
  }
  forecasts() {
    return this.request<DemandForecast[]>("forecasts");
  }
  status() {
    return this.request<AgentStatus>("status");
  }
  runDemo() {
    return this.request("demo/short-delivery", {}, "demo-short-delivery-v1");
  }

  decideAction(id: string, decision: "approve" | "reject") {
    return this.request(`actions/${id}/${decision}`, {}, `${decision}-${id}`);
  }

  async simulateRecovery(request: StockRequest, onProgress: () => Promise<void>) {
    await this.request(
      `stock-requests/${request.id}/receive`,
      {
        quantity: request.quantity,
        source_reference: `DEMO receiving record ${request.id}`,
      },
      `demo-receipt-${request.id}`,
    );
    for (let index = 0; index < 3; index++) {
      if (index) await new Promise((resolve) => setTimeout(resolve, 5200));
      await this.request(
        "kitchen/metrics",
        {
          prep_minutes: 14,
          cancellation_rate: 0.02,
          order_count: 12,
        },
        `demo-recovery-${request.id}-${index}`,
      );
      await onProgress();
    }
  }

  async saveOperation(operation: string, form: FormData, catalog: Catalog) {
    const quantity = Number(form.get("quantity"));
    if (operation === "order") {
      return this.request("orders", { menu_item_id: form.get("item"), quantity });
    }
    if (operation === "waste") {
      return this.request("waste", {
        ingredient_id: form.get("item"),
        quantity,
        reason: form.get("reason"),
      });
    }
    if (operation === "purchase") {
      return this.request("purchase-orders", {
        ingredient_id: form.get("item"),
        supplier_id: catalog.suppliers[0]?.id,
        quantity,
      });
    }
    if (operation.startsWith("receipt:")) {
      return this.request(`purchase-orders/${operation.split(":")[1]}/receipts`, {
        quantity,
        source_reference: form.get("reason"),
        external_status: form.get("external_status"),
      });
    }
    throw new Error("Unknown operation");
  }
}

export const operationsClient = new OperationsClient();
