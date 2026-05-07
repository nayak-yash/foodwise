import { useEffect, useRef } from "react";
import { Check, LoaderCircle, X } from "lucide-react";
import { operationsClient } from "@/lib/operations-client";
import type { Catalog, InventoryItem, RunAction } from "@/lib/types";

export function OperationDialog({
  operation,
  inventory,
  catalog,
  busy,
  onClose,
  runAction,
}: {
  operation: string;
  inventory: InventoryItem[];
  catalog: Catalog;
  busy: string;
  onClose: () => void;
  runAction: RunAction;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    ref.current?.showModal();
  }, []);
  const isReceipt = operation.startsWith("receipt:");
  const title =
    operation === "purchase"
      ? "New purchase order"
      : operation === "order"
        ? "Record order preparation"
        : operation === "waste"
          ? "Record ingredient waste"
          : "Record actual receipt";

  return (
    <dialog
      ref={ref}
      className="modal native-dialog"
      aria-labelledby="operation-title"
      onCancel={(event) => {
        if (busy) event.preventDefault();
        else onClose();
      }}
    >
      <div className="panel-heading">
        <h2 id="operation-title">{title}</h2>
        <button
          className="icon-button"
          aria-label="Close dialog"
          disabled={!!busy}
          onClick={onClose}
        >
          <X size={19} />
        </button>
      </div>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          void runAction(
            "operation",
            async () => {
              await operationsClient.saveOperation(operation, data, catalog);
              onClose();
            },
            "Operation recorded successfully.",
          );
        }}
      >
        {!isReceipt && (
          <label>
            {operation === "order" ? "Menu item" : "Ingredient"}
            <select name="item" required>
              {(operation === "order" ? catalog.menu : inventory).map((item) => (
                <option value={item.id} key={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
        )}
        <label>
          {operation === "order" ? "Portions entering preparation" : "Quantity (kg)"}
          <input
            name="quantity"
            type="number"
            step={operation === "order" ? "1" : "0.001"}
            min={operation === "order" ? "1" : "0.001"}
            required
          />
        </label>
        {(operation === "waste" || isReceipt) && (
          <label>
            {operation === "waste" ? "Reason" : "Receiving document reference"}
            <input name="reason" required />
          </label>
        )}
        {isReceipt && (
          <label>
            Supplier-reported status
            <select name="external_status">
              <option value="completed">Completed</option>
              <option value="partial">Partial</option>
            </select>
          </label>
        )}
        {operation === "purchase" && (
          <p className="muted">Supplier: {catalog.suppliers[0]?.name}</p>
        )}
        {operation === "order" && (
          <p className="muted">Ingredients are deducted once, when preparation starts.</p>
        )}
        <button className="button primary" disabled={!!busy} type="submit">
          {busy ? <LoaderCircle className="spin" size={15} /> : <Check size={15} />}Save operation
        </button>
      </form>
    </dialog>
  );
}
