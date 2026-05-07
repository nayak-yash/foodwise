package events

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"restaurantops/backend/internal/persistence"
)

type MessageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type OutboxPublisher struct {
	database persistence.Database
	writer   MessageWriter
}

func NewOutboxPublisher(database persistence.Database, writer MessageWriter) *OutboxPublisher {
	return &OutboxPublisher{database: database, writer: writer}
}

func (publisher *OutboxPublisher) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := publisher.publishBatch(ctx); err != nil && ctx.Err() == nil {
				slog.Warn("outbox publish retry", "error", err)
			}
		}
	}
}

func (publisher *OutboxPublisher) publishBatch(ctx context.Context) error {
	transaction, err := publisher.database.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Rollback(ctx)

	rows, err := persistence.Query(ctx, transaction, `
        SELECT id::text, type, payload, branch_id::text, created_at
        FROM outbox WHERE published_at IS NULL
        ORDER BY created_at LIMIT 50 FOR UPDATE SKIP LOCKED
    `)
	if err != nil {
		return err
	}
	for _, row := range rows {
		payload, err := json.Marshal(row)
		if err != nil {
			return err
		}
		message := kafka.Message{Key: []byte(row["branch_id"].(string)), Value: payload}
		if err := publisher.writer.WriteMessages(ctx, message); err != nil {
			return err
		}
		if _, err := transaction.Exec(ctx, "UPDATE outbox SET published_at=now() WHERE id=$1", row["id"]); err != nil {
			return err
		}
	}
	return transaction.Commit(ctx)
}
