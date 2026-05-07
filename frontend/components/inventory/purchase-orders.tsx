import { Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { PurchaseOrder } from "@/lib/types";

export function PurchaseOrders({
  orders,
  onOperation,
}: {
  orders: PurchaseOrder[];
  onOperation: (operation: string) => void;
}) {
  return (
    <section className="panel">
      <div className="panel-heading">
        <div>
          <div className="section-label">RECEIVING & RECONCILIATION</div>
          <h2>Purchase orders</h2>
        </div>
        <button className="button primary small" onClick={() => onOperation("purchase")}>
          <Plus size={14} />
          New purchase order
        </button>
      </div>
      <div className="table-scroll">
        <table>
          <thead>
            <tr>
              <th>Ingredient / supplier</th>
              <th>Ordered</th>
              <th>Received</th>
              <th>Outstanding</th>
              <th>Canonical status</th>
              <th>Supplier flag</th>
              <th />
            </tr>
          </thead>
          <tbody>
            {orders.map((order) => (
              <tr key={order.id}>
                <td>
                  <strong>{order.ingredient}</strong>
                  <small>{order.supplier}</small>
                </td>
                <td>{order.quantity} kg</td>
                <td>{order.received_quantity} kg</td>
                <td className={order.outstanding ? "orange-text" : ""}>{order.outstanding} kg</td>
                <td>
                  <Badge value={order.status} />
                </td>
                <td>{order.external_status}</td>
                <td>
                  {order.outstanding > 0 && (
                    <button
                      className="button secondary small"
                      onClick={() => onOperation(`receipt:${order.id}`)}
                    >
                      Record receipt
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {orders.length === 0 && (
          <div className="empty-small">Create a purchase order or run the incident demo.</div>
        )}
      </div>
    </section>
  );
}
