package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNotificationEmailGroupRateChangeEventIsOptionalAndPreviewable(t *testing.T) {
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)
	events := make(map[string]NotificationEmailEventInfo)
	for _, info := range svc.ListEventInfos() {
		events[info.Event] = info
	}

	info, ok := events[NotificationEmailEventGroupRateChangeNotice]
	require.True(t, ok)
	require.True(t, info.Optional)
	require.Contains(t, info.Placeholders, "new_rate_multiplier")

	preview, err := svc.PreviewTemplate(context.Background(), NotificationEmailPreviewInput{
		Event:  NotificationEmailEventGroupRateChangeNotice,
		Locale: "zh",
	})
	require.NoError(t, err)
	require.NotEmpty(t, preview.Subject)
	require.Contains(t, preview.HTML, "$3.42")
	require.Contains(t, preview.HTML, "class=\"hero\"")
}

func TestNotificationEmailCustomCardAppliesToOfficialLowBalanceTemplate(t *testing.T) {
	svc := NewNotificationEmailService(newNotificationEmailMemorySettingRepo(), nil)

	preview, err := svc.PreviewTemplate(context.Background(), NotificationEmailPreviewInput{
		Event:  NotificationEmailEventBalanceLow,
		Locale: "en",
	})
	require.NoError(t, err)
	require.Contains(t, preview.HTML, "class=\"hero\"")
	require.Contains(t, preview.HTML, "Low balance alert")
}
