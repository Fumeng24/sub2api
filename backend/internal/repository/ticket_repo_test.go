package repository

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

func TestTicketRepositoryUnreadSummaryUsesPostgresPlaceholders(t *testing.T) {
	var capturedSQL string
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
		capturedSQL = actualSQL
		if strings.Contains(actualSQL, "?") {
			return fmt.Errorf("query contains mysql-style placeholder: %s", actualSQL)
		}
		normalized := strings.Join(strings.Fields(actualSQL), " ")
		if !strings.Contains(normalized, `m.visibility = $5`) {
			return fmt.Errorf("query does not use postgres placeholders in unread predicate: %s", actualSQL)
		}
		return nil
	})))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := dbent.NewClient(dbent.Driver(drv))
	t.Cleanup(func() { _ = client.Close() })

	userID := int64(123)
	mock.ExpectQuery("ticket unread summary").
		WithArgs(
			userID,
			service.TicketReadActorUser,
			userID,
			false,
			service.TicketMessageVisibilityPublic,
			service.TicketReadActorUser,
			userID,
		).
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).AddRow(service.TicketStatusOpen, int64(2)))

	summary, err := NewTicketRepository(client, db).UnreadSummary(
		context.Background(),
		&userID,
		service.TicketReadActorUser,
		userID,
		false,
	)
	require.NoError(t, err)
	require.Equal(t, &service.TicketUnreadSummary{Total: 2, Open: 2}, summary)
	require.NotEmpty(t, capturedSQL)
	require.NoError(t, mock.ExpectationsWereMet())
}
