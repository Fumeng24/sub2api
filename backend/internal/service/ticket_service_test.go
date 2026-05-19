//go:build unit

package service

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type ticketRepoStub struct {
	ticket                   *Ticket
	messages                 []TicketMessage
	createMessageCalls       int
	addMessageUpdateCalls    int
	updateCalls              int
	markReadCalls            int
	statsFilters             []TicketListFilters
	unreadSummaryFilters     []TicketListFilters
	nextTicketID             int64
	nextMessageID            int64
	lastMessageCreatePayload *TicketMessage
	lastUpdatePayload        *Ticket
}

func newTicketRepoStub(t *Ticket) *ticketRepoStub {
	return &ticketRepoStub{
		ticket:        t,
		nextTicketID:  100,
		nextMessageID: 200,
	}
}

func (s *ticketRepoStub) CreateWithMessage(_ context.Context, t *Ticket, msg *TicketMessage) error {
	if t.ID == 0 {
		t.ID = s.nextTicketID
		s.nextTicketID++
	}
	if msg.ID == 0 {
		msg.ID = s.nextMessageID
		s.nextMessageID++
	}
	msg.TicketID = t.ID
	s.ticket = cloneTicket(t)
	s.messages = append(s.messages, *cloneMessage(msg))
	s.lastMessageCreatePayload = cloneMessage(msg)
	return nil
}

func (s *ticketRepoStub) CreateMessage(_ context.Context, msg *TicketMessage) error {
	if msg.ID == 0 {
		msg.ID = s.nextMessageID
		s.nextMessageID++
	}
	s.createMessageCalls++
	s.messages = append(s.messages, *cloneMessage(msg))
	s.lastMessageCreatePayload = cloneMessage(msg)
	return nil
}

func (s *ticketRepoStub) AddMessageAndUpdateTicket(_ context.Context, msg *TicketMessage, t *Ticket) error {
	if msg.ID == 0 {
		msg.ID = s.nextMessageID
		s.nextMessageID++
	}
	s.addMessageUpdateCalls++
	s.messages = append(s.messages, *cloneMessage(msg))
	s.ticket = cloneTicket(t)
	s.lastMessageCreatePayload = cloneMessage(msg)
	s.lastUpdatePayload = cloneTicket(t)
	return nil
}

func (s *ticketRepoStub) GetByID(context.Context, int64) (*Ticket, error) {
	if s.ticket == nil {
		return nil, ErrTicketNotFound
	}
	return cloneTicket(s.ticket), nil
}

func (s *ticketRepoStub) GetByIDForUser(_ context.Context, _ int64, userID int64) (*Ticket, error) {
	if s.ticket == nil || s.ticket.UserID != userID {
		return nil, ErrTicketNotFound
	}
	return cloneTicket(s.ticket), nil
}

func (s *ticketRepoStub) Update(_ context.Context, t *Ticket) error {
	s.updateCalls++
	s.ticket = cloneTicket(t)
	s.lastUpdatePayload = cloneTicket(t)
	return nil
}

func (s *ticketRepoStub) List(context.Context, pagination.PaginationParams, TicketListFilters) ([]Ticket, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *ticketRepoStub) ListByUser(context.Context, int64, pagination.PaginationParams, TicketListFilters) ([]Ticket, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *ticketRepoStub) ListMessages(context.Context, int64, bool) ([]TicketMessage, error) {
	return append([]TicketMessage(nil), s.messages...), nil
}

func (s *ticketRepoStub) MarkRead(context.Context, int64, string, int64, *int64) error {
	s.markReadCalls++
	return nil
}

func (s *ticketRepoStub) UnreadCounts(context.Context, []int64, string, int64, bool) (map[int64]int, error) {
	return map[int64]int{}, nil
}

func (s *ticketRepoStub) UnreadSummary(_ context.Context, _ *int64, _ string, _ int64, _ bool, filters ...TicketListFilters) (*TicketUnreadSummary, error) {
	s.unreadSummaryFilters = append([]TicketListFilters(nil), filters...)
	return &TicketUnreadSummary{}, nil
}

func (s *ticketRepoStub) Stats(_ context.Context, filters ...TicketListFilters) (*TicketStats, error) {
	s.statsFilters = append([]TicketListFilters(nil), filters...)
	return &TicketStats{}, nil
}

func (s *ticketRepoStub) StatsForAssignee(context.Context, int64) (*TicketStats, error) {
	return &TicketStats{}, nil
}

func (s *ticketRepoStub) AutoCloseResolved(context.Context, time.Time) (int, error) {
	return 0, nil
}

func (s *ticketRepoStub) ListSLAActionable(context.Context, time.Time, int) ([]Ticket, error) {
	return nil, nil
}

func TestTicketServiceCreateStoresInitialAttachments(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Username: "alice", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	ticket, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:   1,
		Subject:  "Cannot use model",
		Body:     "Please check the request",
		Category: TicketCategoryTechnical,
		Priority: TicketPriorityHigh,
		Attachments: []TicketAttachment{{
			Name: "screenshot.png",
			URL:  "https://example.com/screenshot.png",
		}},
	})

	require.NoError(t, err)
	require.NotZero(t, ticket.ID)
	require.Len(t, ticket.Messages, 1)
	require.Equal(t, "screenshot.png", ticket.Messages[0].Attachments[0].Name)
	require.Equal(t, "https://example.com/screenshot.png", repo.lastMessageCreatePayload.Attachments[0].URL)
	require.Equal(t, 1, repo.markReadCalls)
}

func TestTicketServiceAcceptsInlineImageAttachment(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-png"))

	ticket, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:   1,
		Subject:  "Cannot use model",
		Body:     "Please check the request",
		Category: TicketCategoryTechnical,
		Priority: TicketPriorityHigh,
		Attachments: []TicketAttachment{{
			Name:        "screenshot.png",
			URL:         dataURL,
			ContentType: "image/png",
			Size:        8,
		}},
	})

	require.NoError(t, err)
	require.NotZero(t, ticket.ID)
	require.Equal(t, dataURL, repo.lastMessageCreatePayload.Attachments[0].URL)
}

func TestTicketServiceRejectsInvalidAttachmentURL(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:      1,
		Subject:     "Bad attachment",
		Body:        "Body",
		Attachments: []TicketAttachment{{Name: "local", URL: "file:///tmp/a.png"}},
	})

	require.ErrorIs(t, err, ErrTicketAttachmentInvalid)
}

func TestTicketServiceRequiredImageFieldRequiresFieldValue(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:      1,
		Subject:     "Group cannot connect",
		Body:        "The group cannot connect and shows an upstream error.",
		TemplateKey: "group_connection_issue",
		ContextData: map[string]any{"group_id": 10},
		Attachments: []TicketAttachment{{
			Name: "unrelated-log.txt",
			URL:  "https://example.com/log.txt",
		}},
	})

	require.ErrorIs(t, err, ErrTicketTemplateFieldInvalid)
	require.Nil(t, repo.ticket)
}

func TestTicketServiceImageFieldAcceptsInlineImageDataURL(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)
	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("fake-png"))

	ticket, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:      1,
		Subject:     "Group cannot connect",
		Body:        "The group cannot connect and shows an upstream error.",
		TemplateKey: "group_connection_issue",
		ContextData: map[string]any{
			"group_id":         10,
			"error_screenshot": dataURL,
		},
	})

	require.NoError(t, err)
	require.NotZero(t, ticket.ID)
	require.Equal(t, dataURL, repo.ticket.ContextData["error_screenshot"])
}

func TestTicketServiceImageFieldRejectsNonImageDataURL(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)
	dataURL := "data:text/plain;base64," + base64.StdEncoding.EncodeToString([]byte("not an image"))

	_, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:      1,
		Subject:     "Group cannot connect",
		Body:        "The group cannot connect and shows an upstream error.",
		TemplateKey: "group_connection_issue",
		ContextData: map[string]any{
			"group_id":         10,
			"error_screenshot": dataURL,
		},
	})

	require.ErrorIs(t, err, ErrTicketTemplateFieldInvalid)
	require.Nil(t, repo.ticket)
}

func TestTicketServiceImageFieldRejectsSVGDataURL(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)
	dataURL := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(`<svg onload="alert(1)"></svg>`))

	_, err := svc.CreateForUser(context.Background(), &CreateTicketInput{
		UserID:      1,
		Subject:     "Group cannot connect",
		Body:        "The group cannot connect and shows an upstream error.",
		TemplateKey: "group_connection_issue",
		ContextData: map[string]any{
			"group_id":         10,
			"error_screenshot": dataURL,
		},
	})

	require.ErrorIs(t, err, ErrTicketTemplateFieldInvalid)
	require.Nil(t, repo.ticket)
}

func TestTicketServiceUserCannotReplyClosedTicket(t *testing.T) {
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusClosed})
	userRepo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Role: RoleUser, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.AddUserMessage(context.Background(), 10, 1, &AddTicketMessageInput{Body: "reopen?"})

	require.ErrorIs(t, err, ErrTicketClosed)
	require.Zero(t, repo.addMessageUpdateCalls)
}

func TestTicketServiceInternalAdminNoteDoesNotMutateTicket(t *testing.T) {
	now := time.Now()
	repo := newTicketRepoStub(&Ticket{
		ID:                10,
		UserID:            1,
		Status:            TicketStatusOpen,
		LastMessageAt:     now,
		LastUserMessageAt: &now,
	})
	userRepo := &userRepoStub{user: &User{ID: 2, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	msg, err := svc.AddAdminMessage(context.Background(), 10, 2, &AddTicketMessageInput{Body: "internal only", Internal: true})

	require.NoError(t, err)
	require.Equal(t, TicketMessageVisibilityInternal, msg.Visibility)
	require.Equal(t, 1, repo.createMessageCalls)
	require.Zero(t, repo.addMessageUpdateCalls)
	require.Equal(t, TicketStatusOpen, repo.ticket.Status)
	require.Equal(t, now, repo.ticket.LastMessageAt)
}

func TestTicketServicePublicAdminReplyMarksPending(t *testing.T) {
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusOpen, LastMessageAt: time.Now()})
	userRepo := &userRepoStub{user: &User{ID: 2, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	msg, err := svc.AddAdminMessage(context.Background(), 10, 2, &AddTicketMessageInput{
		Body: "we replied",
		Attachments: []TicketAttachment{{
			Name: "log",
			URL:  "https://example.com/log.txt",
		}},
	})

	require.NoError(t, err)
	require.Equal(t, TicketMessageVisibilityPublic, msg.Visibility)
	require.Equal(t, TicketStatusPending, repo.ticket.Status)
	require.NotNil(t, repo.ticket.LastAdminMessageAt)
	require.Nil(t, repo.ticket.ResolvedAt)
	require.Nil(t, repo.ticket.ClosedAt)
	require.Equal(t, "log", repo.lastMessageCreatePayload.Attachments[0].Name)
	require.Equal(t, 1, repo.addMessageUpdateCalls)
}

func TestTicketServiceUpdateClosedSetsResolutionTimestamps(t *testing.T) {
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusOpen, LastMessageAt: time.Now()})
	svc := NewTicketService(repo, &userRepoStub{user: &User{ID: 2, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive}}, nil, nil)
	status := TicketStatusClosed

	updated, err := svc.UpdateForAdmin(context.Background(), 10, &UpdateTicketInput{ActorID: 2, Status: &status})

	require.NoError(t, err)
	require.Equal(t, TicketStatusClosed, updated.Status)
	require.NotNil(t, updated.ResolvedAt)
	require.NotNil(t, updated.ClosedAt)
	require.Equal(t, 1, repo.updateCalls)
}

func TestTicketServiceRejectsOrdinaryUserAssignee(t *testing.T) {
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusOpen, LastMessageAt: time.Now()})
	userRepo := &userRepoStub{
		user: &User{ID: 2, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
		usersByID: map[int64]*User{
			2: {ID: 2, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
			3: {ID: 3, Email: "user@example.com", Role: RoleUser, Status: StatusActive},
		},
	}
	svc := NewTicketService(repo, userRepo, nil, nil)
	assigneeID := int64(3)
	assigneePtr := &assigneeID

	_, err := svc.UpdateForAdmin(context.Background(), 10, &UpdateTicketInput{
		ActorID:    2,
		AssigneeID: &assigneePtr,
	})

	require.ErrorIs(t, err, ErrTicketAssigneeInvalid)
	require.Zero(t, repo.updateCalls)
}

func TestTicketServiceSupportMustClaimBeforeReply(t *testing.T) {
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusOpen, LastMessageAt: time.Now()})
	userRepo := &userRepoStub{user: &User{ID: 4, Email: "support@example.com", Role: RoleSupport, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.AddAdminMessage(context.Background(), 10, 4, &AddTicketMessageInput{Body: "checking"})

	require.ErrorIs(t, err, ErrTicketPermissionDenied)
	require.Zero(t, repo.addMessageUpdateCalls)
}

func TestTicketServiceSupportCanReplyClaimedTicket(t *testing.T) {
	assigneeID := int64(4)
	repo := newTicketRepoStub(&Ticket{ID: 10, UserID: 1, Status: TicketStatusOpen, AssigneeID: &assigneeID, LastMessageAt: time.Now()})
	userRepo := &userRepoStub{user: &User{ID: 4, Email: "support@example.com", Role: RoleSupport, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.AddAdminMessage(context.Background(), 10, 4, &AddTicketMessageInput{Body: "fixed"})

	require.NoError(t, err)
	require.Equal(t, 1, repo.addMessageUpdateCalls)
}

func TestTicketServiceSupportStatsUseVisibleQueueFilters(t *testing.T) {
	repo := newTicketRepoStub(nil)
	userRepo := &userRepoStub{user: &User{ID: 4, Email: "support@example.com", Role: RoleSupport, Status: StatusActive}}
	svc := NewTicketService(repo, userRepo, nil, nil)

	_, err := svc.StatsForAdmin(context.Background(), 4)

	require.NoError(t, err)
	require.Len(t, repo.statsFilters, 1)
	require.Equal(t, "support", repo.statsFilters[0].Queue)
	require.NotNil(t, repo.statsFilters[0].SupportActorID)
	require.Equal(t, int64(4), *repo.statsFilters[0].SupportActorID)
	require.Len(t, repo.unreadSummaryFilters, 1)
	require.Equal(t, "support", repo.unreadSummaryFilters[0].Queue)
	require.NotNil(t, repo.unreadSummaryFilters[0].SupportActorID)
	require.Equal(t, int64(4), *repo.unreadSummaryFilters[0].SupportActorID)
}

func cloneTicket(in *Ticket) *Ticket {
	if in == nil {
		return nil
	}
	out := *in
	out.Messages = append([]TicketMessage(nil), in.Messages...)
	return &out
}

func cloneMessage(in *TicketMessage) *TicketMessage {
	if in == nil {
		return nil
	}
	out := *in
	out.Attachments = append([]TicketAttachment(nil), in.Attachments...)
	return &out
}
