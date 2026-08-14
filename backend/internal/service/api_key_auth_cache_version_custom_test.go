package service

import (
	"fmt"
	"testing"
)

func TestAPIKeyService_RejectsV11AndV12AuthSnapshots(t *testing.T) {
	groupID := int64(9)
	svc := &APIKeyService{}
	for _, version := range []int{11, 12} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			apiKey, ok, err := svc.applyAuthCacheEntry("k-legacy", &APIKeyAuthCacheEntry{
				Snapshot: &APIKeyAuthSnapshot{Version: version, APIKeyID: 1, UserID: 2, GroupID: &groupID, Status: StatusActive,
					User:  APIKeyAuthUserSnapshot{ID: 2, Status: StatusActive, Role: RoleUser, Balance: 10, Concurrency: 3},
					Group: &APIKeyAuthGroupSnapshot{ID: groupID, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive, SubscriptionType: SubscriptionTypeStandard, RateMultiplier: 1}},
			})
			if err != nil || ok || apiKey != nil {
				t.Fatalf("expected v%d snapshot to be rejected without error, got key=%#v ok=%v err=%v", version, apiKey, ok, err)
			}
		})
	}
}
