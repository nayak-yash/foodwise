package httpapi

import (
	"encoding/json"
	"math"
	"os"
	"strconv"

	"restaurantops/backend/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *IncidentController) createIncident(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			PurchaseOrderID string `json:"purchase_order_id"`
			Title           string `json:"title"`
			Severity        string `json:"severity"`
			Fingerprint     string `json:"fingerprint"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if in.Title == "" || in.Fingerprint == "" || (in.Severity != "high" && in.Severity != "medium" && in.Severity != "critical") {
			return nil, bad("title, fingerprint and valid severity required")
		}
		var po string
		if err := tx.QueryRow(c, "SELECT id::text FROM purchase_orders WHERE id=$1 AND branch_id=$2", in.PurchaseOrderID, branchID).Scan(&po); err != nil {
			return nil, err
		}
		var id string
		err := tx.QueryRow(c, "INSERT INTO incidents(branch_id,purchase_order_id,fingerprint,title,severity) VALUES($1,$2,$3,$4,$5) ON CONFLICT(branch_id,fingerprint) DO UPDATE SET fingerprint=EXCLUDED.fingerprint RETURNING id::text", branchID, po, in.Fingerprint, in.Title, in.Severity).Scan(&id)
		if err != nil {
			return nil, err
		}
		if err = audit(c, tx, id, "detector", "incident.detected", in); err != nil {
			return nil, err
		}
		return gin.H{"id": id}, emit(c, tx, "incident.detected", gin.H{"incident_id": id})
	})
}

var allowedTransition = domain.AllowedTransition

func (s *IncidentController) report(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			Status    string          `json:"status"`
			Diagnosis json.RawMessage `json:"diagnosis"`
			Recovery  json.RawMessage `json:"recovery"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		var status string
		if err := tx.QueryRow(c, "SELECT status FROM incidents WHERE id=$1 AND branch_id=$2 FOR UPDATE", c.Param("id"), branchID).Scan(&status); err != nil {
			return nil, err
		}
		if !allowedTransition(status, in.Status) {
			return nil, conflict("invalid incident state transition: " + status + " → " + in.Status)
		}
		if in.Status == "resolved" {
			ok, err := recoveryAllowed(c, tx, c.Param("id"))
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, conflict("recovery is not yet supported by stock and kitchen measurements")
			}
		}
		if len(in.Diagnosis) == 0 {
			in.Diagnosis = nil
		}
		if len(in.Recovery) == 0 {
			in.Recovery = nil
		}
		_, err := tx.Exec(c, "UPDATE incidents SET status=$1,diagnosis=COALESCE($2::jsonb,diagnosis),recovery=COALESCE($3::jsonb,recovery),updated_at=now() WHERE id=$4", in.Status, in.Diagnosis, in.Recovery, c.Param("id"))
		if err != nil {
			return nil, err
		}
		if err = audit(c, tx, c.Param("id"), "investigator", "incident."+in.Status, in); err != nil {
			return nil, err
		}
		return gin.H{"status": in.Status}, emit(c, tx, "incident.updated", gin.H{"incident_id": c.Param("id"), "status": in.Status})
	})
}

func (s *IncidentController) plan(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			Quantity float64 `json:"quantity"`
			Reason   string  `json:"reason"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if !validQuantity(in.Quantity) || in.Reason == "" {
			return nil, bad("positive quantity and reason required")
		}
		var status, ingredient string
		var outstanding float64
		err := tx.QueryRow(c, "SELECT i.status,p.ingredient_id::text,(p.quantity-p.received_quantity)::float8 FROM incidents i JOIN purchase_orders p ON p.id=i.purchase_order_id WHERE i.id=$1 AND i.branch_id=$2 FOR UPDATE OF i,p", c.Param("id"), branchID).Scan(&status, &ingredient, &outstanding)
		if err != nil {
			return nil, err
		}
		if status != "investigating" {
			return nil, conflict("incident must be investigating")
		}
		if in.Quantity > outstanding+0.00001 {
			return nil, conflict("replacement quantity exceeds evidenced shortfall")
		}
		parameters, _ := json.Marshal(gin.H{"ingredient_id": ingredient, "quantity": in.Quantity, "reason": in.Reason, "execution_mode": "internal_stock_request", "verification": "Actual receipt plus 3 healthy kitchen samples totaling 30 orders"})
		var id string
		err = tx.QueryRow(c, "INSERT INTO actions(incident_id,type,parameters) VALUES($1,'replacement_stock',$2) ON CONFLICT(incident_id,type) DO UPDATE SET parameters=EXCLUDED.parameters WHERE actions.status='proposed' RETURNING id::text", c.Param("id"), parameters).Scan(&id)
		if err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE incidents SET status='awaiting_approval',updated_at=now() WHERE id=$1", c.Param("id")); err != nil {
			return nil, err
		}
		if err = audit(c, tx, c.Param("id"), "planner", "plan.proposed", in); err != nil {
			return nil, err
		}
		return gin.H{"id": id, "status": "proposed"}, nil
	})
}

func (s *RecoveryController) approve(c *gin.Context) { s.decide(c, true) }
func (s *RecoveryController) reject(c *gin.Context)  { s.decide(c, false) }
func (s *RecoveryController) decide(c *gin.Context, approve bool) {
	s.mutate(c, func(tx pgx.Tx, _ json.RawMessage) (any, error) {
		var incident, status string
		err := tx.QueryRow(c, "SELECT a.incident_id::text,a.status FROM actions a JOIN incidents i ON i.id=a.incident_id WHERE a.id=$1 AND i.branch_id=$2 FOR UPDATE OF a,i", c.Param("id"), branchID).Scan(&incident, &status)
		if err != nil {
			return nil, err
		}
		if status != "proposed" {
			return nil, conflict("only proposed actions can be approved or rejected")
		}
		decision, incidentStatus := "rejected", "escalated"
		if approve {
			decision = "approved"
			incidentStatus = "approved"
		}
		if _, err = tx.Exec(c, "UPDATE actions SET status=$1,approved_by='manager',approved_at=now() WHERE id=$2", decision, c.Param("id")); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE incidents SET status=$1,updated_at=now() WHERE id=$2", incidentStatus, incident); err != nil {
			return nil, err
		}
		if err = audit(c, tx, incident, "manager", "action."+decision, gin.H{"action_id": c.Param("id")}); err != nil {
			return nil, err
		}
		return gin.H{"status": decision}, emit(c, tx, "action."+decision, gin.H{"incident_id": incident, "action_id": c.Param("id")})
	})
}

func (s *RecoveryController) execute(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, _ json.RawMessage) (any, error) {
		var incident, status string
		var parameters struct {
			IngredientID string  `json:"ingredient_id"`
			Quantity     float64 `json:"quantity"`
		}
		var raw json.RawMessage
		err := tx.QueryRow(c, "SELECT a.incident_id::text,a.status,a.parameters FROM actions a JOIN incidents i ON i.id=a.incident_id WHERE a.id=$1 AND i.branch_id=$2 FOR UPDATE OF a,i", c.Param("id"), branchID).Scan(&incident, &status, &raw)
		if err != nil {
			return nil, err
		}
		if status == "executed" || status == "received" {
			var id string
			err = tx.QueryRow(c, "SELECT id::text FROM stock_requests WHERE action_id=$1", c.Param("id")).Scan(&id)
			return gin.H{"id": id, "status": status}, err
		}
		if status != "approved" {
			return nil, conflict("manager approval required before execution")
		}
		if err = json.Unmarshal(raw, &parameters); err != nil {
			return nil, err
		}
		// Revalidate against live receipts: approval cannot authorize a now-obsolete quantity.
		var outstanding float64
		if err = tx.QueryRow(c, "SELECT (p.quantity-p.received_quantity)::float8 FROM purchase_orders p JOIN incidents i ON i.purchase_order_id=p.id WHERE i.id=$1 FOR UPDATE OF p", incident).Scan(&outstanding); err != nil {
			return nil, err
		}
		if parameters.Quantity > outstanding+0.00001 {
			return nil, conflict("stock situation changed; approved replacement needs review")
		}
		id := uuid.NewString()
		if _, err = tx.Exec(c, "INSERT INTO stock_requests(id,action_id,branch_id,ingredient_id,quantity) VALUES($1,$2,$3,$4,$5)", id, c.Param("id"), branchID, parameters.IngredientID, parameters.Quantity); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE actions SET status='executed',executed_at=now() WHERE id=$1", c.Param("id")); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE incidents SET status='verifying',updated_at=now() WHERE id=$1", incident); err != nil {
			return nil, err
		}
		if err = audit(c, tx, incident, c.GetString("role"), "action.executed", gin.H{"stock_request_id": id, "stock_added": false}); err != nil {
			return nil, err
		}
		return gin.H{"id": id, "status": "requested"}, emit(c, tx, "action.executed", gin.H{"incident_id": incident, "action_id": c.Param("id")})
	})
}

func (s *RecoveryController) receiveStock(c *gin.Context) {
	s.mutate(c, func(tx pgx.Tx, raw json.RawMessage) (any, error) {
		var in struct {
			Quantity        float64 `json:"quantity"`
			SourceReference string  `json:"source_reference"`
		}
		if err := decode(raw, &in); err != nil {
			return nil, err
		}
		if !validQuantity(in.Quantity) || in.SourceReference == "" {
			return nil, bad("actual quantity and receiving reference required")
		}
		var action, ingredient, status, incident string
		var quantity float64
		err := tx.QueryRow(c, "SELECT r.action_id::text,r.ingredient_id::text,r.status,r.quantity::float8,a.incident_id::text FROM stock_requests r JOIN actions a ON a.id=r.action_id WHERE r.id=$1 AND r.branch_id=$2 FOR UPDATE OF r,a", c.Param("id"), branchID).Scan(&action, &ingredient, &status, &quantity, &incident)
		if err != nil {
			return nil, err
		}
		if status != "requested" {
			return nil, conflict("request already received")
		}
		if math.Abs(quantity-in.Quantity) > 0.00001 {
			return nil, conflict("MVP replacement receipt must match requested quantity")
		}
		if _, err = tx.Exec(c, "SELECT id FROM ingredients WHERE id=$1 FOR UPDATE", ingredient); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "INSERT INTO inventory_movements(branch_id,ingredient_id,quantity,kind,reference) VALUES($1,$2,$3,'transfer',$4)", branchID, ingredient, quantity, in.SourceReference); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE stock_requests SET status='received' WHERE id=$1", c.Param("id")); err != nil {
			return nil, err
		}
		if _, err = tx.Exec(c, "UPDATE actions SET status='received',received_at=now() WHERE id=$1", action); err != nil {
			return nil, err
		}
		if err = audit(c, tx, incident, "manager", "stock.received", in); err != nil {
			return nil, err
		}
		return gin.H{"status": "received"}, emit(c, tx, "stock.received", gin.H{"incident_id": incident})
	})
}

func recoveryAllowed(c *gin.Context, tx pgx.Tx, incident string) (bool, error) {
	seconds := 600.0
	if os.Getenv("DEMO_MODE") == "true" {
		if value, err := strconv.ParseFloat(os.Getenv("VERIFY_WINDOW_SECONDS"), 64); err == nil && value >= 10 {
			seconds = value
		}
	}
	var ok bool
	err := tx.QueryRow(c, `WITH received AS (
 SELECT a.received_at,(a.parameters->>'ingredient_id')::uuid AS ingredient_id
 FROM actions a WHERE a.incident_id=$1 AND a.status='received'
 ), samples AS (
 SELECT k.* FROM kitchen_metrics k,received r WHERE k.branch_id=$2 AND k.recorded_at>=r.received_at
 ORDER BY k.recorded_at DESC LIMIT 3
 ) SELECT EXISTS(SELECT 1 FROM received r WHERE
 (SELECT COALESCE(sum(quantity),0) FROM inventory_movements WHERE ingredient_id=r.ingredient_id AND branch_id=$2)>0)
 AND (SELECT count(*)>=3 AND sum(order_count)>=30 AND max(prep_minutes)<=18 AND max(cancellation_rate)<=0.05
 AND extract(epoch FROM max(recorded_at)-min(recorded_at)) >= $3
 AND max(recorded_at)>now()-interval '15 minutes' FROM samples)`, incident, branchID, seconds).Scan(&ok)
	return ok, err
}
