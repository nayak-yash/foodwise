package persistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Database is the storage boundary consumed by controllers and event publishers.
type Database interface {
	Querier
	Begin(context.Context) (pgx.Tx, error)
	Ping(context.Context) error
}

type Querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func Query(ctx context.Context, database Querier, sql string, args ...any) ([]map[string]any, error) {
	rows, err := database.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]any, 0)
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := map[string]any{}
		for index, field := range rows.FieldDescriptions() {
			value := values[index]
			if id, ok := value.([16]byte); ok {
				value = uuid.UUID(id).String()
			}
			row[field.Name] = value
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
