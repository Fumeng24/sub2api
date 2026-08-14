package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type opsRuntimeAlertRepoMock struct {
	opsRepoMock
	createAlertEventFn func(ctx context.Context, event *OpsAlertEvent) (*OpsAlertEvent, error)
}

func (m *opsRuntimeAlertRepoMock) CreateAlertEvent(ctx context.Context, event *OpsAlertEvent) (*OpsAlertEvent, error) {
	if m.createAlertEventFn != nil {
		return m.createAlertEventFn(ctx, event)
	}
	return event, nil
}

func TestOpsRuntimeAlertService_StoresSanitizedDedupedEvent(t *testing.T) {
	var created []*OpsAlertEvent
	repo := &opsRuntimeAlertRepoMock{
		createAlertEventFn: func(ctx context.Context, event *OpsAlertEvent) (*OpsAlertEvent, error) {
			event.ID = int64(len(created) + 1)
			created = append(created, event)
			return event, nil
		},
	}
	opsSvc := NewOpsService(repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	alertSvc := NewOpsRuntimeAlertService(repo, opsSvc, nil)

	input := OpsRuntimeAlertInput{
		Type:        OpsRuntimeAlertTypeOpenAISelectionEmpty,
		Severity:    "P1",
		Title:       "OpenAI scheduler empty candidates",
		Description: "empty after filters",
		DedupKey:    "selection-empty:12:gpt-5.5:/v1/responses:false",
		DedupWindow: time.Minute,
		Dimensions: map[string]any{
			"account_id":  38795,
			"account_ids": []int64{38795, 38801},
			"api_key":     "sk-secret",
			"token_value": "bearer-secret",
			"reason":      "runtime_circuit_open",
		},
	}

	alertSvc.Emit(context.Background(), input)
	alertSvc.Emit(context.Background(), input)

	require.Len(t, created, 1)
	event := created[0]
	require.Equal(t, OpsAlertStatusFiring, event.Status)
	require.Equal(t, "P1", event.Severity)
	require.Equal(t, OpsRuntimeAlertTypeOpenAISelectionEmpty, event.Dimensions["alert_type"])
	require.Equal(t, 38795, event.Dimensions["account_id"])
	require.Equal(t, []int64{38795, 38801}, event.Dimensions["account_ids"])
	require.Equal(t, "runtime_circuit_open", event.Dimensions["reason"])
	require.NotContains(t, event.Dimensions, "api_key")
	require.NotContains(t, event.Dimensions, "token_value")
}
