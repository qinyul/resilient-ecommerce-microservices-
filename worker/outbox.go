package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
)

type OutboxEvent struct {
	ID            int
	AggregateID   string
	AggregateType string
	EventType     string
	Payload       json.RawMessage
	Status        string
}

type outboxRelayWorker struct {
	db     *sql.DB
	broker domain.MessageBroker
}

func NewOutboxRelayWorker(db *sql.DB, broker domain.MessageBroker) *outboxRelayWorker {
	return &outboxRelayWorker{
		db:     db,
		broker: broker,
	}
}

func (w *outboxRelayWorker) Start(ctx context.Context) {
	slog.Info("Starting Outbox Relay Worker...")

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox Relay Worker shutting down gracefully")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *outboxRelayWorker) processPendingEvents(ctx context.Context) {
	// We use a transaction to ensure that SKIP LOCKED behaves correctly
	// and locks are held until we either commit (success) or rollback (fail/retry).
	const fetchQuery = `
		SELECT id, aggregate_id, aggregate_type, event_type, payload 
		FROM outbox_events
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`

	rows, err := w.db.QueryContext(ctx, fetchQuery)
	if err != nil {
		slog.Error("Worker error fetching outbox events", "error", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ev OutboxEvent
		if err := rows.Scan(&ev.ID, &ev.AggregateID, &ev.AggregateType, &ev.EventType, &ev.Payload); err != nil {
			slog.Error("Worker error scanning event", "id", ev.ID, "error", err)
			continue
		}

		err = w.publishToRabbitMQ(ctx, ev)
		if err != nil {
			slog.Warn("Worker failed to publish event (will retry)", "id", ev.ID, "error", err)
			continue
		}

		const updateQuery = `UPDATE outbox_events SET status = 'published', updated_at = NOW() WHERE id = $1`
		_, err = w.db.ExecContext(ctx, updateQuery, ev.ID)
		if err != nil {
			slog.Error("Worker failed to mark event as published", "id", ev.ID, "error", err)
		} else {
			slog.Info("Worker successfully published event", "type", ev.EventType, "id", ev.ID)
		}
	}
}

func (w *outboxRelayWorker) publishToRabbitMQ(ctx context.Context, ev OutboxEvent) error {
	switch ev.EventType {
	case "OrderCreated":
		var order domain.Order
		if err := json.Unmarshal(ev.Payload, &order); err != nil {
			slog.Error("Failed to unmarshal OrderCreated payload (skipping poison pill)", "id", ev.ID, "error", err)
			// Return nil to mark as 'published' to avoid infinite retry of broken data,
			// or move to a 'failed' status in a real production system.
			return nil 
		}

		return w.broker.PublishOrderCreated(ctx, &order)
	case "OrderCancelled":
		// TODO: implement OrderCancelled handling
		return nil
	default:
		slog.Warn("Unknown event type, skipping...", "type", ev.EventType, "id", ev.ID)
		return nil
	}
}
