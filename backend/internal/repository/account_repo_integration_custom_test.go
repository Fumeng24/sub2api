//go:build integration

package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/accountgroup"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type schedulerBucketRemovalRecorder struct {
	schedulerCacheRecorder
	removedIDs []int64
}

func (s *schedulerBucketRemovalRecorder) RemoveAccountFromBuckets(_ context.Context, accountID int64) error {
	s.removedIDs = append(s.removedIDs, accountID)
	return nil
}

func (s *AccountRepoSuite) TestListWithFilters_ActiveExcludesOverloadedAndExpired() {
	client, repo := s.newIsolatedAccountRepoCustom()
	mustCreateAccount(s.T(), client, &service.Account{Name: "active-normal", Status: service.StatusActive})

	overloaded := mustCreateAccount(s.T(), client, &service.Account{Name: "active-overloaded", Status: service.StatusActive})
	s.Require().NoError(client.Account.UpdateOneID(overloaded.ID).
		SetOverloadUntil(time.Now().Add(20 * time.Minute)).
		Exec(s.ctx))

	expired := mustCreateAccount(s.T(), client, &service.Account{Name: "active-expired", Status: service.StatusActive})
	s.Require().NoError(client.Account.UpdateOneID(expired.ID).
		SetAutoPauseOnExpired(true).
		SetExpiresAt(time.Now().Add(-time.Minute)).
		Exec(s.ctx))

	accounts, page, err := repo.ListWithFilters(s.ctx, pagination.PaginationParams{Page: 1, PageSize: 10}, "", "", service.StatusActive, "", 0, "")
	s.Require().NoError(err)
	s.Require().Len(accounts, 1)
	s.Require().Equal("active-normal", accounts[0].Name)
	s.Require().Equal(int64(1), page.Total)
}

func (s *AccountRepoSuite) TestListWithFilters_TempUnschedulableExcludesManualOff() {
	client, repo := s.newIsolatedAccountRepoCustom()

	temporary := mustCreateAccount(s.T(), client, &service.Account{Name: "active-temp-unsched", Status: service.StatusActive, Schedulable: true})
	s.Require().NoError(client.Account.UpdateOneID(temporary.ID).
		SetTempUnschedulableUntil(time.Now().Add(15 * time.Minute)).
		Exec(s.ctx))

	manualOff := mustCreateAccount(s.T(), client, &service.Account{Name: "active-temp-unsched-manual-off", Status: service.StatusActive})
	s.Require().NoError(client.Account.UpdateOneID(manualOff.ID).
		SetSchedulable(false).
		SetTempUnschedulableUntil(time.Now().Add(15 * time.Minute)).
		Exec(s.ctx))

	accounts, page, err := repo.ListWithFilters(s.ctx, pagination.PaginationParams{Page: 1, PageSize: 10}, "", "", "temp_unschedulable", "", 0, "")
	s.Require().NoError(err)
	s.Require().Len(accounts, 1)
	s.Require().Equal("active-temp-unsched", accounts[0].Name)
	s.Require().Equal(int64(1), page.Total)
}

func (s *AccountRepoSuite) newIsolatedAccountRepoCustom() (*dbent.Client, *accountRepository) {
	tx := testEntTx(s.T())
	return tx.Client(), newAccountRepositoryWithSQL(tx.Client(), tx, nil)
}

func (s *AccountRepoSuite) TestBindGroups_PreservesSchedulingConfigForRetainedGroups() {
	g1 := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-preserve-1"})
	g2 := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-preserve-2"})
	g3 := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-preserve-3"})
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "acc-preserve"})

	s.Require().NoError(s.repo.BindGroups(s.ctx, account.ID, []int64{g1.ID, g2.ID}))
	_, err := s.client.AccountGroup.Update().
		Where(accountgroup.AccountIDEQ(account.ID), accountgroup.GroupIDEQ(g1.ID)).
		SetRole(service.AccountGroupRoleBackup).
		SetWeight(7).
		SetSortOrder(90).
		SetSchedulingConfigured(true).
		Save(s.ctx)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.BindGroups(s.ctx, account.ID, []int64{g2.ID, g1.ID, g3.ID}))
	preserved, err := s.client.AccountGroup.Query().
		Where(accountgroup.AccountIDEQ(account.ID), accountgroup.GroupIDEQ(g1.ID)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(service.AccountGroupRoleBackup, preserved.Role)
	s.Require().Equal(7, preserved.Weight)
	s.Require().Equal(90, preserved.SortOrder)
	s.Require().True(preserved.SchedulingConfigured)
	s.Require().Equal(2, preserved.Priority)

	retainedDefault, err := s.client.AccountGroup.Query().
		Where(accountgroup.AccountIDEQ(account.ID), accountgroup.GroupIDEQ(g2.ID)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(1, retainedDefault.Priority)
	s.Require().Equal(2, retainedDefault.SortOrder)
	s.Require().True(retainedDefault.SchedulingConfigured)

	addedDefault, err := s.client.AccountGroup.Query().
		Where(accountgroup.AccountIDEQ(account.ID), accountgroup.GroupIDEQ(g3.ID)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(service.AccountGroupRolePrimary, addedDefault.Role)
	s.Require().Equal(100, addedDefault.Weight)
	s.Require().Equal(3, addedDefault.Priority)
	s.Require().Equal(3, addedDefault.SortOrder)
	s.Require().True(addedDefault.SchedulingConfigured)
}

func (s *AccountRepoSuite) TestListSchedulable_FiltersQuotaExceededAfterHydration() {
	group := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-quota"})
	okAcc := mustCreateAccount(s.T(), s.client, &service.Account{Name: "quota-ok", Type: service.AccountTypeAPIKey, Schedulable: true})
	mustBindAccountToGroup(s.T(), s.client, okAcc.ID, group.ID, 1)

	exhausted := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "quota-exhausted",
		Type:        service.AccountTypeAPIKey,
		Schedulable: true,
		Extra: map[string]any{
			"quota_limit": 10.0,
			"quota_used":  10.0,
		},
	})
	mustBindAccountToGroup(s.T(), s.client, exhausted.ID, group.ID, 2)

	accounts, err := s.repo.ListSchedulableByGroupID(s.ctx, group.ID)
	s.Require().NoError(err)
	ids := idsOfAccounts(accounts)
	s.Require().Contains(ids, okAcc.ID)
	s.Require().NotContains(ids, exhausted.ID)
}

func (s *AccountRepoSuite) TestGroupSchedulingConfiguredDoesNotGateSchedulableQueries() {
	group := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-binding-status"})
	account := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:        "binding-not-configured",
		Platform:    service.PlatformAnthropic,
		Schedulable: true,
	})
	mustBindAccountToGroupWithSchedulingConfigured(s.T(), s.client, account.ID, group.ID, 1, false)

	byGroup, err := s.repo.ListSchedulableByGroupID(s.ctx, group.ID)
	s.Require().NoError(err)
	s.Require().Contains(idsOfAccounts(byGroup), account.ID)

	byPlatform, err := s.repo.ListSchedulableByGroupIDAndPlatform(s.ctx, group.ID, service.PlatformAnthropic)
	s.Require().NoError(err)
	s.Require().Contains(idsOfAccounts(byPlatform), account.ID)
}

func (s *AccountRepoSuite) TestSetSchedulable_DoesNotMutateGroupBindingState() {
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "acc-sched-toggle", Schedulable: true})
	group := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-sched-toggle"})
	mustBindAccountToGroupWithSchedulingConfigured(s.T(), s.client, account.ID, group.ID, 1, true)

	s.Require().NoError(s.repo.SetSchedulable(s.ctx, account.ID, false))
	binding, err := s.client.AccountGroup.Query().
		Where(accountgroup.AccountIDEQ(account.ID), accountgroup.GroupIDEQ(group.ID)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Require().True(binding.SchedulingConfigured)
}

func (s *AccountRepoSuite) TestSetSchedulableTrueClearsStaleRuntimeErrorState() {
	until := time.Now().Add(10 * time.Minute)
	account := mustCreateAccount(s.T(), s.client, &service.Account{
		Name:                    "acc-sched-reset",
		Status:                  service.StatusActive,
		Schedulable:             false,
		ErrorMessage:            "Manual disabled 2026-06-22: upstream failures",
		TempUnschedulableUntil:  &until,
		TempUnschedulableReason: "upstream_503",
	})
	cacheRecorder := &schedulerCacheRecorder{}
	s.repo.schedulerCache = cacheRecorder

	s.Require().NoError(s.repo.SetSchedulable(s.ctx, account.ID, true))
	got, err := s.repo.GetByID(s.ctx, account.ID)
	s.Require().NoError(err)
	s.Require().True(got.Schedulable)
	s.Require().Equal(service.StatusActive, got.Status)
	s.Require().Empty(got.ErrorMessage)
	s.Require().Nil(got.TempUnschedulableUntil)
	s.Require().Empty(got.TempUnschedulableReason)
	s.Require().Len(cacheRecorder.setAccounts, 1)
	s.Require().True(cacheRecorder.setAccounts[0].Schedulable)
	s.Require().Empty(cacheRecorder.setAccounts[0].ErrorMessage)
}

func (s *AccountRepoSuite) TestRuntimeBlocksSyncSnapshotButKeepSchedulerBucketMembership() {
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "acc-runtime-block", Status: service.StatusActive, Schedulable: true})
	cacheRecorder := &schedulerBucketRemovalRecorder{}
	s.repo.schedulerCache = cacheRecorder

	future := time.Now().Add(15 * time.Minute)
	s.Require().NoError(s.repo.SetTempUnschedulable(s.ctx, account.ID, future, "upstream_502"))
	s.Require().NoError(s.repo.SetOverloaded(s.ctx, account.ID, future))
	s.Require().NoError(s.repo.SetRateLimited(s.ctx, account.ID, future))
	s.Require().Len(cacheRecorder.setAccounts, 3)
	s.Require().Empty(cacheRecorder.removedIDs)
}

func (s *AccountRepoSuite) TestPermanentUnschedulableRemovesSchedulerBucketMembership() {
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "acc-permanent-block", Status: service.StatusActive, Schedulable: true})
	cacheRecorder := &schedulerBucketRemovalRecorder{}
	s.repo.schedulerCache = cacheRecorder

	s.Require().NoError(s.repo.SetSchedulable(s.ctx, account.ID, false))
	s.Require().Len(cacheRecorder.setAccounts, 1)
	s.Require().Equal([]int64{account.ID}, cacheRecorder.removedIDs)
}

func (s *AccountRepoSuite) TestClearGlobalRuntimeStateKeepsGroupTempUnschedulable() {
	account := mustCreateAccount(s.T(), s.client, &service.Account{Name: "acc-clear-global-runtime"})
	group1 := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-clear-global-runtime-1"})
	group2 := mustCreateGroup(s.T(), s.client, &service.Group{Name: "g-clear-global-runtime-2"})
	until := time.Now().Add(time.Hour).UTC().Truncate(time.Second)

	s.Require().NoError(s.repo.SetGroupTempUnschedulable(s.ctx, account.ID, group1.ID, until, "group-1"))
	s.Require().NoError(s.repo.SetGroupTempUnschedulable(s.ctx, account.ID, group2.ID, until, "group-2"))
	s.Require().NoError(s.repo.SetTempUnschedulable(s.ctx, account.ID, until, "global"))
	s.Require().NoError(s.repo.SetRateLimited(s.ctx, account.ID, until))

	before, err := s.repo.GetByID(s.ctx, account.ID)
	s.Require().NoError(err)
	groupCooldowns := before.Extra["group_temp_unschedulable"]
	s.Require().NotNil(groupCooldowns)

	s.Require().NoError(s.repo.ClearTempUnschedulable(s.ctx, account.ID))
	afterTempClear, err := s.repo.GetByID(s.ctx, account.ID)
	s.Require().NoError(err)
	s.Require().Nil(afterTempClear.TempUnschedulableUntil)
	s.Require().Empty(afterTempClear.TempUnschedulableReason)
	s.Require().Equal(groupCooldowns, afterTempClear.Extra["group_temp_unschedulable"])

	s.Require().NoError(s.repo.ClearRateLimit(s.ctx, account.ID))
	afterRateClear, err := s.repo.GetByID(s.ctx, account.ID)
	s.Require().NoError(err)
	s.Require().Nil(afterRateClear.RateLimitedAt)
	s.Require().Nil(afterRateClear.RateLimitResetAt)
	s.Require().Equal(groupCooldowns, afterRateClear.Extra["group_temp_unschedulable"])

	s.Require().NoError(s.repo.ClearGroupTempUnschedulable(s.ctx, account.ID))
	afterGroupClear, err := s.repo.GetByID(s.ctx, account.ID)
	s.Require().NoError(err)
	s.Require().NotContains(afterGroupClear.Extra, "group_temp_unschedulable")
}
