//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAffiliateRepository_BindInviterAndClaimBindBonus_ReusesOuterTransaction(t *testing.T) {
	ctx := context.Background()

	client := integrationEntClient
	repo := NewAffiliateRepository(client, integrationDB)

	inviter := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-bind-inviter-%d@example.com", time.Now().UnixNano()),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})
	invitee := mustCreateUser(t, client, &service.User{
		Email:        fmt.Sprintf("affiliate-bind-invitee-%d@example.com", time.Now().UnixNano()+1),
		PasswordHash: "hash",
		Role:         service.RoleUser,
		Status:       service.StatusActive,
		Concurrency:  5,
	})

	_, err := repo.EnsureUserAffiliate(ctx, inviter.ID)
	require.NoError(t, err)
	_, err = repo.EnsureUserAffiliate(ctx, invitee.ID)
	require.NoError(t, err)

	outerTx, err := integrationEntClient.Tx(ctx)
	require.NoError(t, err, "begin outer tx")
	t.Cleanup(func() { _ = outerTx.Rollback() })
	txCtx := dbent.NewTxContext(ctx, outerTx)
	txClient := outerTx.Client()

	bound, err := repo.BindInviter(txCtx, invitee.ID, inviter.ID)
	require.NoError(t, err)
	require.True(t, bound, "invitee must bind to inviter")

	claimed, balance, err := repo.ClaimBindBonus(txCtx, invitee.ID, 2.5)
	require.NoError(t, err)
	require.True(t, claimed, "invitee must claim bind bonus")
	require.InDelta(t, 2.5, balance, 1e-9)

	innerBalance := querySingleFloat(t, txCtx, txClient,
		"SELECT balance::double precision FROM users WHERE id = $1", invitee.ID)
	require.InDelta(t, 2.5, innerBalance, 1e-9)

	claimedCount := querySingleInt(t, txCtx, txClient,
		"SELECT COUNT(*) FROM user_affiliates WHERE user_id = $1 AND bind_bonus_claimed_at IS NOT NULL", invitee.ID)
	require.Equal(t, 1, claimedCount)

	require.NoError(t, outerTx.Rollback())

	persistedBalance := querySingleFloat(t, ctx, client,
		"SELECT balance::double precision FROM users WHERE id = $1", invitee.ID)
	require.InDelta(t, 0.0, persistedBalance, 1e-9,
		"ClaimBindBonus must be rolled back with the outer tx")

	persistedCount := querySingleInt(t, ctx, client,
		"SELECT aff_count FROM user_affiliates WHERE user_id = $1", inviter.ID)
	require.Equal(t, 0, persistedCount,
		"BindInviter must not persist aff_count increments after rollback")
}
