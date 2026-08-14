package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	gocache "github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

type userGroupRateResolver struct {
	repo         UserGroupRateRepository
	cache        *gocache.Cache
	cacheTTL     time.Duration
	sf           *singleflight.Group
	logComponent string
}

type legacyUserGroupRateRepository interface {
	GetByUserAndGroup(ctx context.Context, userID, groupID int64) (*float64, error)
}

func (r *userGroupRateResolver) loadRateConfig(ctx context.Context, userID, groupID int64) (config *UserGroupRateConfig, err error) {
	defer func() {
		if recover() == nil {
			return
		}
		legacy, ok := r.repo.(legacyUserGroupRateRepository)
		if !ok {
			err = fmt.Errorf("user group rate repository panicked without legacy fallback")
			return
		}
		var rate *float64
		rate, err = legacy.GetByUserAndGroup(ctx, userID, groupID)
		if err != nil || rate == nil {
			return
		}
		config = &UserGroupRateConfig{RateMultiplier: rate}
	}()
	return r.repo.GetRateConfigByUserAndGroup(ctx, userID, groupID)
}

func newUserGroupRateResolver(repo UserGroupRateRepository, cache *gocache.Cache, cacheTTL time.Duration, sf *singleflight.Group, logComponent string) *userGroupRateResolver {
	if cacheTTL <= 0 {
		cacheTTL = defaultUserGroupRateCacheTTL
	}
	if cache == nil {
		cache = gocache.New(cacheTTL, time.Minute)
	}
	if logComponent == "" {
		logComponent = "service.gateway"
	}
	if sf == nil {
		sf = &singleflight.Group{}
	}

	return &userGroupRateResolver{
		repo:         repo,
		cache:        cache,
		cacheTTL:     cacheTTL,
		sf:           sf,
		logComponent: logComponent,
	}
}

func (r *userGroupRateResolver) Resolve(ctx context.Context, userID, groupID int64, groupDefaultMultiplier float64) float64 {
	if r == nil || userID <= 0 || groupID <= 0 {
		return groupDefaultMultiplier
	}

	// Include the group default in the cache key so discount-based rates follow
	// group repricing immediately instead of reusing a cached old effective rate.
	defaultKey := strconv.FormatFloat(groupDefaultMultiplier, 'f', -1, 64)
	key := fmt.Sprintf("%d:%d:%s", userID, groupID, defaultKey)
	if r.cache != nil {
		if cached, ok := r.cache.Get(key); ok {
			if multiplier, castOK := cached.(float64); castOK {
				userGroupRateCacheHitTotal.Add(1)
				return multiplier
			}
		}
	}
	if r.repo == nil {
		return groupDefaultMultiplier
	}
	userGroupRateCacheMissTotal.Add(1)

	value, err, shared := r.sf.Do(key, func() (any, error) {
		if r.cache != nil {
			if cached, ok := r.cache.Get(key); ok {
				if multiplier, castOK := cached.(float64); castOK {
					userGroupRateCacheHitTotal.Add(1)
					return multiplier, nil
				}
			}
		}

		userGroupRateCacheLoadTotal.Add(1)
		rateConfig, repoErr := r.loadRateConfig(ctx, userID, groupID)
		if repoErr != nil {
			return nil, repoErr
		}

		multiplier := groupDefaultMultiplier
		if rateConfig != nil {
			if rateConfig.RateMultiplier != nil {
				multiplier = *rateConfig.RateMultiplier
			} else if rateConfig.DiscountMultiplier != nil {
				multiplier = groupDefaultMultiplier * *rateConfig.DiscountMultiplier
			}
		}
		if r.cache != nil {
			r.cache.Set(key, multiplier, r.cacheTTL)
		}
		return multiplier, nil
	})
	if shared {
		userGroupRateCacheSFSharedTotal.Add(1)
	}
	if err != nil {
		userGroupRateCacheFallbackTotal.Add(1)
		logger.LegacyPrintf(r.logComponent, "get user group rate failed, fallback to group default: user=%d group=%d err=%v", userID, groupID, err)
		return groupDefaultMultiplier
	}

	multiplier, ok := value.(float64)
	if !ok {
		userGroupRateCacheFallbackTotal.Add(1)
		return groupDefaultMultiplier
	}
	return multiplier
}
