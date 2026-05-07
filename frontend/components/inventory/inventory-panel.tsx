import { Boxes, Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import type { InventoryItem } from "@/lib/types";

export function InventoryPanel({
  inventory,
  onOperation,
}: {
  inventory: InventoryItem[];
  onOperation: (operation: string) => void;
}) {
  return (
    <section className="panel inventory-panel">
      <div className="panel-heading">
        <div>
          <div className="section-label">INGREDIENT INTELLIGENCE</div>
          <h2>Stock at a glance</h2>
        </div>
        <div className="button-group">
          <button className="button secondary small" onClick={() => onOperation("order")}>
            <Plus size={14} />
            Record order
          </button>
          <button className="button secondary small" onClick={() => onOperation("waste")}>
            Record waste
          </button>
        </div>
      </div>
      <div className="inventory-grid">
        {inventory.map((item) => (
          <article key={item.id} className="stock-card">
            <div className="stock-header">
              <span className="stock-icon">
                <Boxes size={20} />
              </span>
              <Badge value={item.balance < item.reorder_point ? "low_stock" : "healthy"} />
            </div>
            <h3>{item.name}</h3>
            <div className="stock-quantity">
              {item.balance.toFixed(1)}
              <span>{item.unit}</span>
            </div>
            <div className="stock-track">
              <div
                style={{
                  width: `${Math.min(100, (item.balance / (item.reorder_point * 4)) * 100)}%`,
                  background: item.balance < item.reorder_point ? "var(--orange)" : "var(--green)",
                }}
              />
            </div>
            <p>
              Reorder point{" "}
              <strong>
                {item.reorder_point} {item.unit}
              </strong>
            </p>
          </article>
        ))}
      </div>
    </section>
  );
}
