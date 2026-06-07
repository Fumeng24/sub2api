package service

import "testing"

func TestAPIKeyService_RejectsLegacyAuthSnapshots(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}

	for _, tc := range []struct {
		name    string
		version int
	}{
		{name: "v10_without_models_list_config", version: 10},
		{name: "v11_without_force_openai_priority", version: 11},
	} {
		t.Run(tc.name, func(t *testing.T) {
			apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy", &APIKeyAuthCacheEntry{
				Snapshot: &APIKeyAuthSnapshot{
					Version:  tc.version,
					APIKeyID: 1,
					UserID:   2,
					GroupID:  &groupID,
					Status:   StatusActive,
					User: APIKeyAuthUserSnapshot{
						ID:          2,
						Status:      StatusActive,
						Role:        RoleUser,
						Balance:     10,
						Concurrency: 3,
					},
					Group: &APIKeyAuthGroupSnapshot{
						ID:               groupID,
						Name:             "openai",
						Platform:         PlatformOpenAI,
						Status:           StatusActive,
						SubscriptionType: SubscriptionTypeStandard,
						RateMultiplier:   1,
					},
				},
			})

			if err != nil {
				t.Fatalf("expected stale snapshot to be ignored without error, got %v", err)
			}
			if ok {
				t.Fatalf("expected v%d auth snapshot to be rejected", tc.version)
			}
			if apiKey != nil {
				t.Fatalf("expected no API key from stale snapshot, got %#v", apiKey)
			}
		})
	}
}
