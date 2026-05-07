package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"restaurantops/backend/internal/persistence"
)

type apiError struct {
	status  int
	message string
}

func (e apiError) Error() string    { return e.message }
func bad(message string) error      { return apiError{400, message} }
func conflict(message string) error { return apiError{409, message} }

func respondError(c *gin.Context, err error) {
	var a apiError
	var p *pgconn.PgError
	switch {
	case errors.As(err, &a):
		c.JSON(a.status, gin.H{"error": a.message})
	case errors.Is(err, pgx.ErrNoRows):
		c.JSON(404, gin.H{"error": "record not found"})
	case errors.As(err, &p) && (p.Code == "22P02" || p.Code == "23503" || p.Code == "23514"):
		c.JSON(400, gin.H{"error": "invalid reference or business constraint"})
	default:
		slog.Error("request failed", "path", c.FullPath(), "error", err)
		c.JSON(500, gin.H{"error": "internal server error"})
	}
}

var query = persistence.Query

func (s *Controller) list(c *gin.Context, sql string, args ...any) {
	v, err := query(c, s.db, sql, args...)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(200, v)
}

// Every mutation and its outbox events commit atomically. An advisory lock serializes
// concurrent retries of the same key, including retries before its first commit.
func (s *Controller) mutate(c *gin.Context, fn func(pgx.Tx, json.RawMessage) (any, error)) {
	key := c.GetHeader("Idempotency-Key")
	if key == "" || len(key) > 160 {
		respondError(c, bad("Idempotency-Key (1–160 characters) required"))
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		respondError(c, bad("request body too large"))
		return
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}
	hash := sha256.Sum256(append([]byte(c.Request.Method+":"+c.Request.URL.Path+":"+c.GetString("role")+":"), raw...))
	digest := hex.EncodeToString(hash[:])
	tx, err := s.db.Begin(c)
	if err != nil {
		respondError(c, err)
		return
	}
	defer tx.Rollback(c)
	if _, err = tx.Exec(c, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", branchID+key); err != nil {
		respondError(c, err)
		return
	}
	var existing string
	var response json.RawMessage
	err = tx.QueryRow(c, "SELECT request_hash,response FROM idempotency_keys WHERE branch_id=$1 AND key=$2", branchID, key).Scan(&existing, &response)
	if err == nil {
		if existing != digest {
			respondError(c, conflict("idempotency key already used for a different request"))
			return
		}
		c.Data(200, "application/json", response)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		respondError(c, err)
		return
	}
	result, err := fn(tx, raw)
	if err != nil {
		respondError(c, err)
		return
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		respondError(c, err)
		return
	}
	if _, err = tx.Exec(c, "INSERT INTO idempotency_keys(branch_id,key,request_hash,response) VALUES($1,$2,$3,$4)", branchID, key, digest, encoded); err != nil {
		respondError(c, err)
		return
	}
	if err = tx.Commit(c); err != nil {
		respondError(c, err)
		return
	}
	c.Data(200, "application/json", encoded)
}

func decode(raw json.RawMessage, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return bad("invalid JSON body")
	}
	return nil
}
func emit(ctx context.Context, tx pgx.Tx, kind string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO outbox(branch_id,type,payload) VALUES($1,$2,$3)", branchID, kind, b)
	return err
}
func audit(ctx context.Context, tx pgx.Tx, incident any, actor, event string, details any) error {
	b, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO audit_log(branch_id,incident_id,actor,event,details) VALUES($1,$2,$3,$4,$5)", branchID, incident, actor, event, b)
	return err
}
