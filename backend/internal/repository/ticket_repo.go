package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/predicate"
	"github.com/Wei-Shaw/sub2api/ent/ticket"
	"github.com/Wei-Shaw/sub2api/ent/ticketmessage"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type ticketRepository struct {
	client *dbent.Client
	db     *sql.DB
}

func NewTicketRepository(client *dbent.Client, db *sql.DB) service.TicketRepository {
	return &ticketRepository{client: client, db: db}
}

func (r *ticketRepository) CreateWithMessage(ctx context.Context, t *service.Ticket, msg *service.TicketMessage) error {
	if t == nil || msg == nil {
		return service.ErrTicketNilInput
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		created, err := createTicketEntity(txCtx, txClient, t)
		if err != nil {
			return translatePersistenceError(err, nil, nil)
		}
		applyTicketEntityToService(t, created)

		msg.TicketID = created.ID
		createdMsg, err := createTicketMessageEntity(txCtx, txClient, msg)
		if err != nil {
			return translatePersistenceError(err, nil, nil)
		}
		applyTicketMessageEntityToService(msg, createdMsg)
		return nil
	})
}

func (r *ticketRepository) CreateMessage(ctx context.Context, msg *service.TicketMessage) error {
	if msg == nil {
		return service.ErrTicketNilInput
	}
	client := clientFromContext(ctx, r.client)
	created, err := createTicketMessageEntity(ctx, client, msg)
	if err != nil {
		return translatePersistenceError(err, service.ErrTicketNotFound, nil)
	}
	applyTicketMessageEntityToService(msg, created)
	return nil
}

func (r *ticketRepository) AddMessageAndUpdateTicket(ctx context.Context, msg *service.TicketMessage, t *service.Ticket) error {
	if msg == nil || t == nil {
		return service.ErrTicketNilInput
	}
	return r.withTx(ctx, func(txCtx context.Context, txClient *dbent.Client) error {
		createdMsg, err := createTicketMessageEntity(txCtx, txClient, msg)
		if err != nil {
			return translatePersistenceError(err, service.ErrTicketNotFound, nil)
		}
		applyTicketMessageEntityToService(msg, createdMsg)

		updated, err := updateTicketEntity(txCtx, txClient, t)
		if err != nil {
			return translatePersistenceError(err, service.ErrTicketNotFound, nil)
		}
		applyTicketEntityToService(t, updated)
		return nil
	})
}

func (r *ticketRepository) GetByID(ctx context.Context, id int64) (*service.Ticket, error) {
	m, err := r.client.Ticket.Query().
		Where(ticket.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrTicketNotFound, nil)
	}
	return ticketEntityToService(m), nil
}

func (r *ticketRepository) GetByIDForUser(ctx context.Context, id, userID int64) (*service.Ticket, error) {
	m, err := r.client.Ticket.Query().
		Where(ticket.IDEQ(id), ticket.UserIDEQ(userID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrTicketNotFound, nil)
	}
	return ticketEntityToService(m), nil
}

func (r *ticketRepository) Update(ctx context.Context, t *service.Ticket) error {
	if t == nil {
		return service.ErrTicketNilInput
	}
	client := clientFromContext(ctx, r.client)
	updated, err := updateTicketEntity(ctx, client, t)
	if err != nil {
		return translatePersistenceError(err, service.ErrTicketNotFound, nil)
	}
	applyTicketEntityToService(t, updated)
	return nil
}

func (r *ticketRepository) List(ctx context.Context, params pagination.PaginationParams, filters service.TicketListFilters) ([]service.Ticket, *pagination.PaginationResult, error) {
	q := r.client.Ticket.Query()
	q = applyTicketListFilters(q, filters)
	return listTickets(ctx, q, params)
}

func (r *ticketRepository) ListByUser(ctx context.Context, userID int64, params pagination.PaginationParams, filters service.TicketListFilters) ([]service.Ticket, *pagination.PaginationResult, error) {
	q := r.client.Ticket.Query().Where(ticket.UserIDEQ(userID))
	q = applyTicketListFilters(q, filters)
	return listTickets(ctx, q, params)
}

func (r *ticketRepository) ListMessages(ctx context.Context, ticketID int64, includeInternal bool) ([]service.TicketMessage, error) {
	q := r.client.TicketMessage.Query().
		Where(ticketmessage.TicketIDEQ(ticketID)).
		Order(dbent.Asc(ticketmessage.FieldCreatedAt), dbent.Asc(ticketmessage.FieldID))
	if !includeInternal {
		q = q.Where(ticketmessage.VisibilityEQ(service.TicketMessageVisibilityPublic))
	}
	items, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	return ticketMessageEntitiesToService(items), nil
}

func (r *ticketRepository) MarkRead(ctx context.Context, ticketID int64, actorType string, actorID int64, lastReadMessageID *int64) error {
	if ticketID <= 0 || actorID <= 0 {
		return nil
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO ticket_reads (ticket_id, actor_type, actor_id, last_read_message_id, read_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW())
ON CONFLICT (ticket_id, actor_type, actor_id)
DO UPDATE SET
	last_read_message_id = GREATEST(COALESCE(ticket_reads.last_read_message_id, 0), COALESCE(EXCLUDED.last_read_message_id, 0)),
	read_at = NOW(),
	updated_at = NOW()
`, ticketID, actorType, actorID, lastReadMessageID)
	return err
}

func (r *ticketRepository) UnreadCounts(ctx context.Context, ticketIDs []int64, actorType string, actorID int64, includeInternal bool) (map[int64]int, error) {
	out := make(map[int64]int, len(ticketIDs))
	if len(ticketIDs) == 0 || actorID <= 0 {
		return out, nil
	}

	rows, err := r.db.QueryContext(ctx, `
SELECT m.ticket_id, COUNT(*)::bigint
FROM ticket_messages m
LEFT JOIN ticket_reads r
	ON r.ticket_id = m.ticket_id
	AND r.actor_type = $2
	AND r.actor_id = $3
WHERE m.ticket_id = ANY($1)
	AND ($4::boolean OR m.visibility = 'public')
	AND (m.sender_id IS NULL OR m.sender_type <> $2 OR m.sender_id <> $3)
	AND m.id > COALESCE(r.last_read_message_id, 0)
GROUP BY m.ticket_id
`, pq.Array(ticketIDs), actorType, actorID, includeInternal)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			ticketID int64
			count    int64
		)
		if err := rows.Scan(&ticketID, &count); err != nil {
			return nil, err
		}
		out[ticketID] = int(count)
	}
	return out, rows.Err()
}

func (r *ticketRepository) UnreadSummary(ctx context.Context, userID *int64, actorType string, actorID int64, includeInternal bool, filters ...service.TicketListFilters) (*service.TicketUnreadSummary, error) {
	out := &service.TicketUnreadSummary{}
	if actorID <= 0 {
		return out, nil
	}
	q := r.client.Ticket.Query()
	if userID != nil {
		q = q.Where(ticket.UserIDEQ(*userID))
	}
	if len(filters) > 0 {
		q = applyTicketListFilters(q, filters[0])
	}
	q = q.Where(ticketUnreadPredicate(actorType, actorID, includeInternal))

	var rows []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := q.GroupBy(ticket.FieldStatus).Aggregate(dbent.Count()).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		count := int(row.Count)
		switch row.Status {
		case service.TicketStatusOpen:
			out.Open = count
		case service.TicketStatusPending:
			out.Pending = count
		case service.TicketStatusResolved:
			out.Resolved = count
		case service.TicketStatusClosed:
			out.Closed = count
		}
		out.Total += count
	}
	return out, nil
}

func (r *ticketRepository) Stats(ctx context.Context, filters ...service.TicketListFilters) (*service.TicketStats, error) {
	stats := &service.TicketStats{}
	if len(filters) > 0 {
		q := applyTicketListFilters(r.client.Ticket.Query(), filters[0])
		var rows []struct {
			Status string `json:"status"`
			Count  int64  `json:"count"`
		}
		if err := q.GroupBy(ticket.FieldStatus).Aggregate(dbent.Count()).Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			count := int(row.Count)
			stats.Total += count
			switch row.Status {
			case service.TicketStatusOpen:
				stats.Open = count
			case service.TicketStatusPending:
				stats.Pending = count
			case service.TicketStatusResolved:
				stats.Resolved = count
			case service.TicketStatusClosed:
				stats.Closed = count
			}
		}

		unassigned, err := applyTicketListFilters(r.client.Ticket.Query(), filters[0]).
			Where(ticket.AssigneeIDIsNil()).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		stats.Unassigned = unassigned

		slaOverdue, err := applyTicketListFilters(r.client.Ticket.Query(), filters[0]).
			Where(
				ticket.StatusIn(service.TicketStatusOpen, service.TicketStatusPending),
				ticket.SLADueAtNotNil(),
				ticket.SLADueAtLT(time.Now()),
			).
			Count(ctx)
		if err != nil {
			return nil, err
		}
		stats.SLAOverdue = slaOverdue
		return stats, nil
	}
	row := r.db.QueryRowContext(ctx, `
	SELECT
		COUNT(*)::bigint,
		COUNT(*) FILTER (WHERE status = 'open')::bigint,
		COUNT(*) FILTER (WHERE status = 'pending')::bigint,
		COUNT(*) FILTER (WHERE status = 'resolved')::bigint,
		COUNT(*) FILTER (WHERE status = 'closed')::bigint,
		COUNT(*) FILTER (WHERE assignee_id IS NULL)::bigint,
	COUNT(*) FILTER (
		WHERE status IN ('open', 'pending')
			AND sla_due_at IS NOT NULL
			AND sla_due_at < NOW()
	)::bigint
FROM tickets
`)
	var total, open, pending, resolved, closed, unassigned, slaOverdue int64
	if err := row.Scan(&total, &open, &pending, &resolved, &closed, &unassigned, &slaOverdue); err != nil {
		return nil, err
	}
	stats.Total = int(total)
	stats.Open = int(open)
	stats.Pending = int(pending)
	stats.Resolved = int(resolved)
	stats.Closed = int(closed)
	stats.Unassigned = int(unassigned)
	stats.SLAOverdue = int(slaOverdue)
	return stats, nil
}

func (r *ticketRepository) StatsForAssignee(ctx context.Context, assigneeID int64) (*service.TicketStats, error) {
	stats := &service.TicketStats{}
	if assigneeID <= 0 {
		return stats, nil
	}
	row := r.db.QueryRowContext(ctx, `
SELECT
	COUNT(*) FILTER (WHERE assignee_id = $1)::bigint,
	COUNT(*) FILTER (
		WHERE assignee_id = $1
			AND last_admin_message_at IS NOT NULL
	)::bigint,
	COUNT(*) FILTER (
		WHERE escalated_by = $1
			AND escalated_at IS NOT NULL
	)::bigint
FROM tickets
`, assigneeID)
	var assignedToMe, handledByMe, escalated int64
	if err := row.Scan(&assignedToMe, &handledByMe, &escalated); err != nil {
		return nil, err
	}
	stats.AssignedToMe = int(assignedToMe)
	stats.HandledByMe = int(handledByMe)
	stats.Escalated = int(escalated)
	return stats, nil
}

func (r *ticketRepository) AutoCloseResolved(ctx context.Context, before time.Time) (int, error) {
	res, err := r.db.ExecContext(ctx, `
UPDATE tickets
SET status = 'closed',
	closed_at = COALESCE(closed_at, NOW()),
	resolved_at = COALESCE(resolved_at, NOW()),
	updated_at = NOW()
WHERE status = 'resolved'
	AND resolved_at IS NOT NULL
	AND resolved_at < $1
`, before)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

func (r *ticketRepository) ListSLAActionable(ctx context.Context, before time.Time, limit int) ([]service.Ticket, error) {
	if limit <= 0 {
		limit = 100
	}
	items, err := r.client.Ticket.Query().
		Where(
			ticket.StatusIn(service.TicketStatusOpen, service.TicketStatusPending),
			ticket.SLADueAtNotNil(),
			ticket.SLADueAtLTE(before),
		).
		Order(dbent.Asc(ticket.FieldSLADueAt), dbent.Asc(ticket.FieldID)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return ticketEntitiesToService(items), nil
}

func (r *ticketRepository) withTx(ctx context.Context, fn func(txCtx context.Context, txClient *dbent.Client) error) error {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return fn(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin ticket transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx, tx.Client()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit ticket transaction: %w", err)
	}
	return nil
}

func createTicketEntity(ctx context.Context, client *dbent.Client, t *service.Ticket) (*dbent.Ticket, error) {
	builder := client.Ticket.Create().
		SetTicketNo(t.TicketNo).
		SetUserID(t.UserID).
		SetUserEmail(t.UserEmail).
		SetUserName(t.UserName).
		SetSubject(t.Subject).
		SetCategory(t.Category).
		SetPriority(t.Priority).
		SetStatus(t.Status).
		SetSource(t.Source).
		SetTemplateKey(t.TemplateKey).
		SetContextType(t.ContextType).
		SetContextID(t.ContextID).
		SetContextData(t.ContextData).
		SetLastMessageAt(t.LastMessageAt)

	if t.AssigneeID != nil {
		builder.SetAssigneeID(*t.AssigneeID)
	}
	if t.EscalatedAt != nil {
		builder.SetEscalatedAt(*t.EscalatedAt)
	}
	if t.EscalatedBy != nil {
		builder.SetEscalatedBy(*t.EscalatedBy)
	}
	if t.EscalationReason != "" {
		builder.SetEscalationReason(t.EscalationReason)
	}
	if t.SLADueAt != nil {
		builder.SetSLADueAt(*t.SLADueAt)
	}
	if t.SLARemindedAt != nil {
		builder.SetSLARemindedAt(*t.SLARemindedAt)
	}
	if t.LastUserMessageAt != nil {
		builder.SetLastUserMessageAt(*t.LastUserMessageAt)
	}
	if t.LastAdminMessageAt != nil {
		builder.SetLastAdminMessageAt(*t.LastAdminMessageAt)
	}
	if t.ResolvedAt != nil {
		builder.SetResolvedAt(*t.ResolvedAt)
	}
	if t.ClosedAt != nil {
		builder.SetClosedAt(*t.ClosedAt)
	}
	return builder.Save(ctx)
}

func updateTicketEntity(ctx context.Context, client *dbent.Client, t *service.Ticket) (*dbent.Ticket, error) {
	builder := client.Ticket.UpdateOneID(t.ID).
		SetTicketNo(t.TicketNo).
		SetUserID(t.UserID).
		SetUserEmail(t.UserEmail).
		SetUserName(t.UserName).
		SetSubject(t.Subject).
		SetCategory(t.Category).
		SetPriority(t.Priority).
		SetStatus(t.Status).
		SetSource(t.Source).
		SetTemplateKey(t.TemplateKey).
		SetContextType(t.ContextType).
		SetContextID(t.ContextID).
		SetContextData(t.ContextData).
		SetLastMessageAt(t.LastMessageAt)

	if t.AssigneeID != nil {
		builder.SetAssigneeID(*t.AssigneeID)
	} else {
		builder.ClearAssigneeID()
	}
	if t.EscalatedAt != nil {
		builder.SetEscalatedAt(*t.EscalatedAt)
	} else {
		builder.ClearEscalatedAt()
	}
	if t.EscalatedBy != nil {
		builder.SetEscalatedBy(*t.EscalatedBy)
	} else {
		builder.ClearEscalatedBy()
	}
	builder.SetEscalationReason(t.EscalationReason)
	if t.SLADueAt != nil {
		builder.SetSLADueAt(*t.SLADueAt)
	} else {
		builder.ClearSLADueAt()
	}
	if t.SLARemindedAt != nil {
		builder.SetSLARemindedAt(*t.SLARemindedAt)
	} else {
		builder.ClearSLARemindedAt()
	}
	if t.LastUserMessageAt != nil {
		builder.SetLastUserMessageAt(*t.LastUserMessageAt)
	} else {
		builder.ClearLastUserMessageAt()
	}
	if t.LastAdminMessageAt != nil {
		builder.SetLastAdminMessageAt(*t.LastAdminMessageAt)
	} else {
		builder.ClearLastAdminMessageAt()
	}
	if t.ResolvedAt != nil {
		builder.SetResolvedAt(*t.ResolvedAt)
	} else {
		builder.ClearResolvedAt()
	}
	if t.ClosedAt != nil {
		builder.SetClosedAt(*t.ClosedAt)
	} else {
		builder.ClearClosedAt()
	}
	return builder.Save(ctx)
}

func createTicketMessageEntity(ctx context.Context, client *dbent.Client, msg *service.TicketMessage) (*dbent.TicketMessage, error) {
	builder := client.TicketMessage.Create().
		SetTicketID(msg.TicketID).
		SetSenderType(msg.SenderType).
		SetSenderName(msg.SenderName).
		SetVisibility(msg.Visibility).
		SetBody(msg.Body).
		SetAttachments(msg.Attachments)
	if msg.SenderID != nil {
		builder.SetSenderID(*msg.SenderID)
	}
	if msg.EditedAt != nil {
		builder.SetEditedAt(*msg.EditedAt)
	}
	return builder.Save(ctx)
}

func applyTicketListFilters(q *dbent.TicketQuery, filters service.TicketListFilters) *dbent.TicketQuery {
	if filters.Status != "" {
		q = q.Where(ticket.StatusEQ(filters.Status))
	}
	if filters.Priority != "" {
		q = q.Where(ticket.PriorityEQ(filters.Priority))
	}
	if filters.Category != "" {
		q = q.Where(ticket.CategoryEQ(filters.Category))
	}
	if filters.TemplateKey != "" {
		q = q.Where(ticket.TemplateKeyEQ(filters.TemplateKey))
	}
	if filters.EscalatedOnly {
		q = q.Where(ticket.EscalatedAtNotNil())
	}
	if filters.Queue == "support" {
		q = q.Where(ticket.EscalatedAtIsNil())
	}
	if filters.SupportActorID != nil && *filters.SupportActorID > 0 {
		q = q.Where(ticket.Or(
			ticket.AssigneeIDIsNil(),
			ticket.AssigneeIDEQ(*filters.SupportActorID),
		))
	}
	if filters.AssigneeID != nil {
		if *filters.AssigneeID > 0 {
			q = q.Where(ticket.AssigneeIDEQ(*filters.AssigneeID))
		} else {
			q = q.Where(ticket.AssigneeIDIsNil())
		}
	}
	if filters.Search != "" {
		search := strings.TrimSpace(filters.Search)
		q = q.Where(ticket.Or(
			ticket.TicketNoContainsFold(search),
			ticket.SubjectContainsFold(search),
			ticket.UserEmailContainsFold(search),
			ticket.UserNameContainsFold(search),
		))
	}
	if filters.UnreadOnly && filters.ReadActorID > 0 {
		q = q.Where(ticketUnreadPredicate(filters.ReadActorType, filters.ReadActorID, filters.IncludeInternal))
	}
	return q
}

func ticketUnreadPredicate(actorType string, actorID int64, includeInternal bool) predicate.Ticket {
	return predicate.Ticket(func(s *entsql.Selector) {
		s.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString("EXISTS (SELECT 1 FROM ticket_messages m LEFT JOIN ticket_reads r ON r.ticket_id = m.ticket_id AND r.actor_type = ")
			b.Arg(actorType)
			b.WriteString(" AND r.actor_id = ")
			b.Arg(actorID)
			b.WriteString(" WHERE m.ticket_id = ")
			b.WriteString(s.C(ticket.FieldID))
			b.WriteString(" AND (")
			b.Arg(includeInternal)
			b.WriteString(" OR m.visibility = ")
			b.Arg(service.TicketMessageVisibilityPublic)
			b.WriteString(") AND (m.sender_id IS NULL OR m.sender_type <> ")
			b.Arg(actorType)
			b.WriteString(" OR m.sender_id <> ")
			b.Arg(actorID)
			b.WriteString(") AND m.id > COALESCE(r.last_read_message_id, 0))")
		}))
	})
}

func listTickets(ctx context.Context, q *dbent.TicketQuery, params pagination.PaginationParams) ([]service.Ticket, *pagination.PaginationResult, error) {
	total, err := q.Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	itemsQuery := q.
		Offset(params.Offset()).
		Limit(params.Limit())
	for _, order := range ticketListOrders(params) {
		itemsQuery = itemsQuery.Order(order)
	}

	items, err := itemsQuery.All(ctx)
	if err != nil {
		return nil, nil, err
	}
	return ticketEntitiesToService(items), paginationResultFromTotal(int64(total), params), nil
}

func ticketListOrder(params pagination.PaginationParams) (string, string) {
	sortBy := strings.ToLower(strings.TrimSpace(params.SortBy))
	sortOrder := params.NormalizedSortOrder(pagination.SortOrderDesc)

	switch sortBy {
	case "ticket_no":
		return ticket.FieldTicketNo, sortOrder
	case "subject":
		return ticket.FieldSubject, sortOrder
	case "status":
		return ticket.FieldStatus, sortOrder
	case "priority":
		return ticket.FieldPriority, sortOrder
	case "category":
		return ticket.FieldCategory, sortOrder
	case "template_key":
		return ticket.FieldTemplateKey, sortOrder
	case "assignee_id":
		return ticket.FieldAssigneeID, sortOrder
	case "updated_at":
		return ticket.FieldUpdatedAt, sortOrder
	case "created_at":
		return ticket.FieldCreatedAt, sortOrder
	case "id":
		return ticket.FieldID, sortOrder
	case "", "last_message_at":
		return ticket.FieldLastMessageAt, sortOrder
	default:
		return ticket.FieldLastMessageAt, pagination.SortOrderDesc
	}
}

func ticketListOrders(params pagination.PaginationParams) []func(*entsql.Selector) {
	field, sortOrder := ticketListOrder(params)
	if sortOrder == pagination.SortOrderAsc {
		if field == ticket.FieldID {
			return []func(*entsql.Selector){dbent.Asc(field)}
		}
		return []func(*entsql.Selector){dbent.Asc(field), dbent.Asc(ticket.FieldID)}
	}
	if field == ticket.FieldID {
		return []func(*entsql.Selector){dbent.Desc(field)}
	}
	return []func(*entsql.Selector){dbent.Desc(field), dbent.Desc(ticket.FieldID)}
}

func applyTicketEntityToService(dst *service.Ticket, src *dbent.Ticket) {
	if dst == nil || src == nil {
		return
	}
	*dst = *ticketEntityToService(src)
}

func ticketEntityToService(m *dbent.Ticket) *service.Ticket {
	if m == nil {
		return nil
	}
	return &service.Ticket{
		ID:                 m.ID,
		TicketNo:           m.TicketNo,
		UserID:             m.UserID,
		UserEmail:          m.UserEmail,
		UserName:           m.UserName,
		Subject:            m.Subject,
		Category:           m.Category,
		Priority:           m.Priority,
		Status:             m.Status,
		Source:             m.Source,
		TemplateKey:        m.TemplateKey,
		ContextType:        m.ContextType,
		ContextID:          m.ContextID,
		ContextData:        cloneTicketContextData(m.ContextData),
		AssigneeID:         m.AssigneeID,
		EscalatedAt:        m.EscalatedAt,
		EscalatedBy:        m.EscalatedBy,
		EscalationReason:   m.EscalationReason,
		SLADueAt:           m.SLADueAt,
		SLARemindedAt:      m.SLARemindedAt,
		LastMessageAt:      m.LastMessageAt,
		LastUserMessageAt:  m.LastUserMessageAt,
		LastAdminMessageAt: m.LastAdminMessageAt,
		ResolvedAt:         m.ResolvedAt,
		ClosedAt:           m.ClosedAt,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func ticketEntitiesToService(models []*dbent.Ticket) []service.Ticket {
	out := make([]service.Ticket, 0, len(models))
	for i := range models {
		if t := ticketEntityToService(models[i]); t != nil {
			out = append(out, *t)
		}
	}
	return out
}

func cloneTicketContextData(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func applyTicketMessageEntityToService(dst *service.TicketMessage, src *dbent.TicketMessage) {
	if dst == nil || src == nil {
		return
	}
	*dst = *ticketMessageEntityToService(src)
}

func ticketMessageEntityToService(m *dbent.TicketMessage) *service.TicketMessage {
	if m == nil {
		return nil
	}
	return &service.TicketMessage{
		ID:          m.ID,
		TicketID:    m.TicketID,
		SenderType:  m.SenderType,
		SenderID:    m.SenderID,
		SenderName:  m.SenderName,
		Visibility:  m.Visibility,
		Body:        m.Body,
		Attachments: m.Attachments,
		EditedAt:    m.EditedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func ticketMessageEntitiesToService(models []*dbent.TicketMessage) []service.TicketMessage {
	out := make([]service.TicketMessage, 0, len(models))
	for i := range models {
		if msg := ticketMessageEntityToService(models[i]); msg != nil {
			out = append(out, *msg)
		}
	}
	return out
}
