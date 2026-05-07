package httpapi

import (
	"github.com/gin-gonic/gin"
)

func (s *QueryController) overview(c *gin.Context) {
	rows, err := query(c, s.db, `SELECT b.id,b.name,
 (SELECT count(*) FROM incidents WHERE branch_id=b.id AND status NOT IN ('resolved','escalated')) AS active_incidents,
 (SELECT count(*) FROM actions a JOIN incidents i ON i.id=a.incident_id WHERE i.branch_id=b.id AND a.status='proposed') AS pending_approvals,
 (SELECT count(*) FROM incidents WHERE branch_id=b.id AND status='resolved') AS resolved_incidents,
 (SELECT prep_minutes FROM kitchen_metrics WHERE branch_id=b.id ORDER BY recorded_at DESC LIMIT 1) AS prep_minutes,
 (SELECT cancellation_rate FROM kitchen_metrics WHERE branch_id=b.id ORDER BY recorded_at DESC LIMIT 1) AS cancellation_rate,
 (SELECT max(recorded_at) FROM kitchen_metrics WHERE branch_id=b.id) AS metrics_updated_at,
 (SELECT count(*) FROM outbox WHERE published_at IS NULL AND branch_id=b.id) AS pending_events
 FROM branches b WHERE id=$1`, branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, rows[0])
}
func (s *QueryController) inventory(c *gin.Context) {
	s.list(c, `SELECT i.id,i.name,i.unit,i.reorder_point::float8,COALESCE(sum(m.quantity),0)::float8 AS balance FROM ingredients i LEFT JOIN inventory_movements m ON m.ingredient_id=i.id AND m.branch_id=i.branch_id WHERE i.branch_id=$1 GROUP BY i.id ORDER BY i.name`, branchID)
}
func (s *QueryController) catalog(c *gin.Context) {
	suppliers, err := query(c, s.db, "SELECT id,name,email FROM suppliers WHERE branch_id=$1", branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	menu, err := query(c, s.db, "SELECT id,name,price::float8,available FROM menu_items WHERE branch_id=$1 ORDER BY name", branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, gin.H{"suppliers": suppliers, "menu": menu})
}
func (s *QueryController) purchaseOrders(c *gin.Context) {
	s.list(c, `SELECT p.id,p.ingredient_id,p.quantity::float8,p.received_quantity::float8,(p.quantity-p.received_quantity)::float8 AS outstanding,p.status,p.external_status,p.created_at,i.name AS ingredient,s.name AS supplier FROM purchase_orders p JOIN ingredients i ON i.id=p.ingredient_id JOIN suppliers s ON s.id=p.supplier_id WHERE p.branch_id=$1 ORDER BY p.created_at DESC LIMIT 100`, branchID)
}
func (s *QueryController) incidents(c *gin.Context) {
	s.list(c, "SELECT * FROM incidents WHERE branch_id=$1 ORDER BY created_at DESC LIMIT 100", branchID)
}
func (s *QueryController) incident(c *gin.Context) {
	rows, err := query(c, s.db, "SELECT * FROM incidents WHERE id=$1 AND branch_id=$2", c.Param("id"), branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(rows) == 0 {
		c.JSON(404, gin.H{"error": "incident not found"})
		return
	}
	actions, err := query(c, s.db, "SELECT * FROM actions WHERE incident_id=$1 ORDER BY id", c.Param("id"))
	if err != nil {
		respondError(c, err)
		return
	}
	timeline, err := query(c, s.db, "SELECT * FROM audit_log WHERE incident_id=$1 AND branch_id=$2 ORDER BY created_at,id", c.Param("id"), branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	rows[0]["actions"] = actions
	rows[0]["timeline"] = timeline
	c.JSON(200, rows[0])
}
func (s *QueryController) demandHistory(c *gin.Context) {
	s.list(c, "SELECT ingredient_id,date::text,service,consumption::float8 FROM demand_history WHERE branch_id=$1 ORDER BY date", branchID)
}
func (s *QueryController) stockRequests(c *gin.Context) {
	s.list(c, `SELECT r.id,r.action_id,a.incident_id,r.ingredient_id,i.name AS ingredient,r.quantity::float8,r.status,r.created_at FROM stock_requests r JOIN ingredients i ON i.id=r.ingredient_id JOIN actions a ON a.id=r.action_id WHERE r.branch_id=$1 ORDER BY r.created_at DESC`, branchID)
}

func (s *QueryController) evidence(c *gin.Context) {
	ctx := c.Request.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		respondError(c, err)
		return
	}
	defer tx.Rollback(ctx)
	// A consistent snapshot prevents the investigator mixing pre/post-receipt values.
	if _, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
		respondError(c, err)
		return
	}
	incident, err := query(ctx, tx, "SELECT * FROM incidents WHERE id=$1 AND branch_id=$2", c.Param("id"), branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(incident) == 0 {
		c.JSON(404, gin.H{"error": "incident not found"})
		return
	}
	po, err := query(ctx, tx, `SELECT p.id,p.ingredient_id,p.quantity::float8,p.received_quantity::float8,(p.quantity-p.received_quantity)::float8 AS outstanding,p.status,p.external_status,i.name AS ingredient FROM purchase_orders p JOIN ingredients i ON i.id=p.ingredient_id WHERE p.id=$1 AND p.branch_id=$2`, incident[0]["purchase_order_id"], branchID)
	if err != nil {
		respondError(c, err)
		return
	}
	if len(po) == 0 {
		c.JSON(404, gin.H{"error": "purchase order evidence unavailable"})
		return
	}
	ingredient := po[0]["ingredient_id"]
	queries := map[string]struct {
		sql  string
		args []any
	}{
		"receipts":  {`SELECT id,quantity::float8,source_reference,received_at FROM deliveries WHERE purchase_order_id=$1 ORDER BY received_at`, []any{po[0]["id"]}},
		"inventory": {`SELECT COALESCE(sum(quantity),0)::float8 AS balance,COALESCE(sum(-quantity) FILTER(WHERE kind='waste' AND created_at>now()-interval '24 hours'),0)::float8 AS waste_24h,COALESCE(sum(-quantity) FILTER(WHERE kind='consumption' AND created_at>now()-interval '24 hours'),0)::float8 AS consumption_24h FROM inventory_movements WHERE ingredient_id=$1 AND branch_id=$2`, []any{ingredient, branchID}},
		"movements": {`SELECT id,kind,quantity::float8,reference,created_at FROM inventory_movements WHERE ingredient_id=$1 AND branch_id=$2 ORDER BY created_at DESC LIMIT 100`, []any{ingredient, branchID}},
		"kitchen":   {`SELECT id,prep_minutes,cancellation_rate,order_count,recorded_at FROM kitchen_metrics WHERE branch_id=$1 ORDER BY recorded_at DESC LIMIT 100`, []any{branchID}},
		"actions":   {`SELECT * FROM actions WHERE incident_id=$1`, []any{c.Param("id")}},
	}
	result := gin.H{"incident": incident[0], "purchase_order": po[0], "observed_at": nil}
	for name, q := range queries {
		rows, err := query(ctx, tx, q.sql, q.args...)
		if err != nil {
			respondError(c, err)
			return
		}
		result[name] = rows
	}
	var observed string
	if err = tx.QueryRow(ctx, "SELECT now()::text").Scan(&observed); err != nil {
		respondError(c, err)
		return
	}
	result["observed_at"] = observed
	c.JSON(200, result)
}
