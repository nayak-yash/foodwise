package httpapi

import (
	"encoding/json"

	"restaurantops/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var validQuantity = domain.ValidQuantity
var receiptStatus = domain.ReceiptStatus

func (s *PurchasingController) createPO(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			SupplierID   string  `json:"supplier_id"`
			IngredientID string  `json:"ingredient_id"`
			Quantity     float64 `json:"quantity"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if !validQuantity(in.Quantity) {
			return nil, bad("quantity must be positive with at most three decimal places")
		}
		var exists bool
		err := tx.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM suppliers s,ingredients i WHERE s.id=$1 AND i.id=$2 AND s.branch_id=$3 AND i.branch_id=$3)`, in.SupplierID, in.IngredientID, branchID).Scan(&exists)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, bad("supplier and ingredient must belong to this branch")
		}
		id := uuid.NewString()
		_, err = tx.Exec(c, "INSERT INTO purchase_orders(id,branch_id,supplier_id,ingredient_id,quantity) VALUES($1,$2,$3,$4,$5)", id, branchID, in.SupplierID, in.IngredientID, in.Quantity)
		if err != nil {
			return nil, err
		}
		return gin.H{"id": id, "status": "ordered"}, emit(c, tx, "purchase_order.created", gin.H{"purchase_order_id": id})
	})
}

func (s *PurchasingController) receivePO(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			Quantity        float64 `json:"quantity"`
			SourceReference string  `json:"source_reference"`
			ExternalStatus  string  `json:"external_status"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if !validQuantity(in.Quantity) || in.SourceReference == "" {
			return nil, bad("positive quantity and source_reference required")
		}
		if in.ExternalStatus != "completed" && in.ExternalStatus != "partial" {
			return nil, bad("external_status must be completed or partial")
		}
		var ordered, received float64
		var ingredient string
		err := tx.QueryRow(c, "SELECT ingredient_id::text,quantity::float8,received_quantity::float8 FROM purchase_orders WHERE id=$1 AND branch_id=$2 FOR UPDATE", c.Param("id"), branchID).Scan(&ingredient, &ordered, &received)
		if err != nil {
			return nil, err
		}
		if in.Quantity > ordered-received+0.00001 {
			return nil, conflict("receipt exceeds outstanding purchase order quantity")
		}
		if _, err = tx.Exec(c, "SELECT id FROM ingredients WHERE id=$1 FOR UPDATE", ingredient); err != nil {
			return nil, err
		}
		status := receiptStatus(ordered, received+in.Quantity)
		if _, err = tx.Exec(c, "UPDATE purchase_orders SET received_quantity=received_quantity+$1,status=$2,external_status=$3 WHERE id=$4", in.Quantity, status, in.ExternalStatus, c.Param("id")); err != nil {
			return nil, err
		}
		id := uuid.NewString()
		if _, err = tx.Exec(c, "INSERT INTO deliveries(id,purchase_order_id,quantity,source_reference) VALUES($1,$2,$3,$4)", id, c.Param("id"), in.Quantity, in.SourceReference); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference) VALUES($1,$2,$3,'receipt',$4)", branchID, ingredient, in.Quantity, id); err != nil {
			return nil, err
		}
		return gin.H{"id": id, "status": status, "outstanding": ordered - received - in.Quantity}, emit(c, tx, "delivery.received", gin.H{"purchase_order_id": c.Param("id"), "delivery_id": id})
	})
}

func (s *InventoryController) createOrder(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			MenuItemID string `json:"menu_item_id"`
			Quantity   int    `json:"quantity"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if in.Quantity < 1 || in.Quantity > 1000 {
			return nil, bad("quantity must be between 1 and 1000")
		}
		var available bool
		if err := tx.QueryRow(c, "SELECT available FROM menu_items WHERE id=$1 AND branch_id=$2", in.MenuItemID, branchID).Scan(&available); err != nil {
			return nil, err
		}
		if !available {
			return nil, conflict("menu item unavailable")
		}
		recipes, err := query(c, tx, "SELECT r.ingredient_id::text,r.quantity::float8 FROM recipes r JOIN ingredients i ON i.id=r.ingredient_id WHERE r.menu_item_id=$1 AND i.branch_id=$2 ORDER BY r.ingredient_id", in.MenuItemID, branchID)
		if err != nil {
			return nil, err
		}
		if len(recipes) == 0 {
			return nil, conflict("menu item has no recipe")
		}
		id := uuid.NewString()
		for _, r := range recipes {
			ingredient := r["ingredient_id"]
			if _, err = tx.Exec(c, "SELECT id FROM ingredients WHERE id=$1 FOR UPDATE", ingredient); err != nil {
				return nil, err
			}
			var balance float64
			if err = tx.QueryRow(c, "SELECT COALESCE(sum(quantity),0)::float8 FROM inventory_movements WHERE ingredient_id=$1 AND branch_id=$2", ingredient, branchID).Scan(&balance); err != nil {
				return nil, err
			}
			needed := r["quantity"].(float64) * float64(in.Quantity)
			if balance+0.00001 < needed {
				return nil, conflict("insufficient stock to start preparation")
			}
			if _, err = tx.Exec(c, "INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference) VALUES($1,$2,$3,'consumption',$4)", branchID, ingredient, -needed, id); err != nil {
				return nil, err
			}
		}
		if _, err = tx.Exec(c, "INSERT INTO orders(id,branch_id,menu_item_id,quantity) VALUES($1,$2,$3,$4)", id, branchID, in.MenuItemID, in.Quantity); err != nil {
			return nil, err
		}
		return gin.H{"id": id, "status": "preparing"}, emit(c, tx, "order.preparing", gin.H{"order_id": id})
	})
}

func (s *InventoryController) recordWaste(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			IngredientID string  `json:"ingredient_id"`
			Quantity     float64 `json:"quantity"`
			Reason       string  `json:"reason"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if !validQuantity(in.Quantity) || in.Reason == "" {
			return nil, bad("ingredient_id, positive quantity and reason required")
		}
		var id string
		if err := tx.QueryRow(c, "SELECT id::text FROM ingredients WHERE id=$1 AND branch_id=$2 FOR UPDATE", in.IngredientID, branchID).Scan(&id); err != nil {
			return nil, err
		}
		var balance float64
		if err := tx.QueryRow(c, "SELECT COALESCE(sum(quantity),0)::float8 FROM inventory_movements WHERE ingredient_id=$1 AND branch_id=$2", id, branchID).Scan(&balance); err != nil {
			return nil, err
		}
		if balance < in.Quantity {
			return nil, conflict("waste exceeds available stock")
		}
		_, err := tx.Exec(c, "INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference) VALUES($1,$2,$3,'waste',$4)", branchID, id, -in.Quantity, in.Reason)
		if err != nil {
			return nil, err
		}
		return gin.H{"status": "recorded"}, emit(c, tx, "waste.recorded", in)
	})
}

func (s *InventoryController) recordMetric(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			Prep         float64 `json:"prep_minutes"`
			Cancellation float64 `json:"cancellation_rate"`
			Orders       int     `json:"order_count"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if in.Prep <= 0 || in.Prep > 240 || in.Cancellation < 0 || in.Cancellation > 1 || in.Orders < 1 {
			return nil, bad("invalid kitchen measurement")
		}
		_, err := tx.Exec(c, "INSERT INTO kitchen_metrics(branch_id,prep_minutes,cancellation_rate,order_count) VALUES($1,$2,$3,$4)", branchID, in.Prep, in.Cancellation, in.Orders)
		if err != nil {
			return nil, err
		}
		return gin.H{"status": "recorded"}, emit(c, tx, "kitchen.measured", in)
	})
}

func (s *DemoController) demo(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, _ json.RawMessage) (any, error) {
		id := uuid.NewString()
		delivery := uuid.NewString()
		if _, err := tx.Exec(c, "SELECT id FROM ingredients WHERE branch_id=$1 ORDER BY id FOR UPDATE", branchID); err != nil {
			return nil, err
		}
		var balance float64
		if err := tx.QueryRow(c, "SELECT COALESCE(sum(quantity),0)::float8 FROM inventory_movements WHERE ingredient_id=$1", chickenID).Scan(&balance); err != nil {
			return nil, err
		}
		if balance+25 < 26 {
			return nil, conflict("demo requires at least 1 kg opening chicken stock")
		}
		_, err := tx.Exec(c, `INSERT INTO purchase_orders(id,branch_id,supplier_id,ingredient_id,quantity,received_quantity,status,external_status) VALUES($1,$2,'33333333-3333-3333-3333-333333333333',$3,40,25,'partial','completed')`, id, branchID, chickenID)
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "INSERT INTO deliveries(id,purchase_order_id,quantity,source_reference) VALUES($1,$2,25,'DEMO signed receiving record: 25 kg')", delivery, id); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference) VALUES($1,$2,25,'receipt',$3),($1,$2,-26,'consumption','DEMO: 104 chicken portions')", branchID, chickenID, delivery); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "INSERT INTO kitchen_metrics(branch_id,prep_minutes,cancellation_rate,order_count) VALUES($1,32,0.18,65)", branchID); err != nil {
			return nil, err
		}
		if err = audit(c, tx, nil, "manager", "demo.started", gin.H{"purchase_order_id": id, "synthetic": true}); err != nil {
			return nil, err
		}
		return gin.H{"purchase_order_id": id, "message": "Synthetic short delivery recorded. Investigation will begin from the Kafka event."}, emit(c, tx, "delivery.received", gin.H{"purchase_order_id": id, "delivery_id": delivery, "synthetic": true})
	})
}
