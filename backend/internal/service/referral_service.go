package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const referralCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 32 chars, no modulo bias
const referralCodeLength = 8
const referralCodeMaxRetries = 5

type ReferralService struct {
	referralRepo        ReferralRepository
	userRepo            UserRepository
	settingService      *SettingService
	billingCacheService *BillingCacheService
}

func NewReferralService(
	referralRepo ReferralRepository,
	userRepo UserRepository,
	settingService *SettingService,
	billingCacheService *BillingCacheService,
) *ReferralService {
	return &ReferralService{
		referralRepo:        referralRepo,
		userRepo:            userRepo,
		settingService:      settingService,
		billingCacheService: billingCacheService,
	}
}

// GetOrCreateProfile 获取或懒加载创建用户的专属邀请码
func (s *ReferralService) GetOrCreateProfile(ctx context.Context, userID int64) (*UserReferralProfile, error) {
	profile, err := s.referralRepo.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get referral profile: %w", err)
	}
	if profile != nil {
		return profile, nil
	}

	for i := 0; i < referralCodeMaxRetries; i++ {
		code, err := generateReferralCode()
		if err != nil {
			return nil, fmt.Errorf("generate referral code: %w", err)
		}
		created, err := s.referralRepo.CreateProfile(ctx, userID, code)
		if err == nil {
			return created, nil
		}
		if !isUniqueConflict(err) {
			return nil, fmt.Errorf("create referral profile: %w", err)
		}
		// Fix #2: user_id unique conflict means another goroutine already created it — re-fetch
		existing, fetchErr := s.referralRepo.GetProfileByUserID(ctx, userID)
		if fetchErr == nil && existing != nil {
			return existing, nil
		}
		// Otherwise it was a referral_code collision, retry with a new code
	}
	return nil, fmt.Errorf("failed to generate unique referral code after %d retries", referralCodeMaxRetries)
}

// ValidateReferralCode 验证邀请码，返回邀请人 profile
func (s *ReferralService) ValidateReferralCode(ctx context.Context, code string, inviteeID int64) (*UserReferralProfile, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, nil
	}

	profile, err := s.referralRepo.GetProfileByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	if inviteeID > 0 && profile.UserID == inviteeID {
		return nil, ErrSelfReferralNotAllowed
	}

	return profile, nil
}

// ProcessReferralRegistration 注册完成后处理邀请奖励
// Fix #1: AuthService 通过此方法处理邀请，不直接访问 referralRepo
// 幂等保障：invitee_id UNIQUE 约束 + reward_granted 标志
func (s *ReferralService) ProcessReferralRegistration(ctx context.Context, referralCode string, inviteeID int64) error {
	profile, err := s.ValidateReferralCode(ctx, referralCode, inviteeID)
	if err != nil || profile == nil {
		return err
	}

	inviterReward, inviteeReward := s.settingService.GetReferralRewards(ctx)

	// Fix #6: validate non-negative rewards
	if inviterReward < 0 {
		inviterReward = 0
	}
	if inviteeReward < 0 {
		inviteeReward = 0
	}

	relation := &ReferralRelation{
		InviterID:     profile.UserID,
		InviteeID:     inviteeID,
		InviterReward: inviterReward,
		InviteeReward: inviteeReward,
	}

	// invitee_id UNIQUE 约束保证不会重复创建
	if err := s.referralRepo.CreateRelation(ctx, relation); err != nil {
		return fmt.Errorf("create referral relation: %w", err)
	}

	// 发放奖励
	if inviterReward > 0 {
		if err := s.userRepo.UpdateBalance(ctx, profile.UserID, inviterReward); err != nil {
			logger.LegacyPrintf("service.referral", "grant inviter reward failed: inviter=%d err=%v", profile.UserID, err)
		}
	}
	if inviteeReward > 0 {
		if err := s.userRepo.UpdateBalance(ctx, inviteeID, inviteeReward); err != nil {
			logger.LegacyPrintf("service.referral", "grant invitee reward failed: invitee=%d err=%v", inviteeID, err)
		}
	}

	// 标记奖励已发放
	if err := s.referralRepo.MarkRewardGranted(ctx, relation.ID); err != nil {
		logger.LegacyPrintf("service.referral", "mark reward granted failed: relation=%d err=%v", relation.ID, err)
	}

	s.invalidateRewardCaches(profile.UserID, inviteeID)
	return nil
}

// GetMyReferralInfo 获取当前用户邀请信息
func (s *ReferralService) GetMyReferralInfo(ctx context.Context, userID int64, siteBaseURL string) (*ReferralInfo, error) {
	profile, err := s.GetOrCreateProfile(ctx, userID)
	if err != nil {
		return nil, err
	}

	totalInvitees, err := s.referralRepo.CountByInviterID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count invitees: %w", err)
	}

	totalReward, err := s.referralRepo.SumRewardsByInviterID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("sum rewards: %w", err)
	}

	return &ReferralInfo{
		ReferralCode:      profile.ReferralCode,
		ReferralLink:      buildReferralLink(siteBaseURL, profile.ReferralCode),
		TotalInvitees:     totalInvitees,
		TotalRewardEarned: totalReward,
	}, nil
}

// ListMyInvitees 分页获取邀请的用户列表
func (s *ReferralService) ListMyInvitees(ctx context.Context, userID int64, params pagination.PaginationParams) ([]ReferralInvitee, *pagination.PaginationResult, error) {
	relations, result, err := s.referralRepo.ListByInviterID(ctx, userID, params)
	if err != nil {
		return nil, nil, fmt.Errorf("list invitees: %w", err)
	}

	invitees := make([]ReferralInvitee, len(relations))
	for i, rel := range relations {
		invitees[i] = ReferralInvitee{
			EmailMasked:  maskEmail(rel.InviteeEmail),
			RegisteredAt: rel.CreatedAt,
			RewardEarned: rel.InviterReward,
		}
	}

	return invitees, result, nil
}

// CountInvitees 获取邀请人数
func (s *ReferralService) CountInvitees(ctx context.Context, userID int64) (int64, error) {
	return s.referralRepo.CountByInviterID(ctx, userID)
}

// SumRewards 获取总奖励金额
func (s *ReferralService) SumRewards(ctx context.Context, userID int64) (float64, error) {
	return s.referralRepo.SumRewardsByInviterID(ctx, userID)
}

// GetPlatformStats 管理员获取平台邀请统计
func (s *ReferralService) GetPlatformStats(ctx context.Context) (*ReferralStats, error) {
	return s.referralRepo.GetPlatformStats(ctx)
}

func (s *ReferralService) invalidateRewardCaches(inviterID, inviteeID int64) {
	if s.billingCacheService == nil {
		return
	}
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, inviterID); err != nil {
			logger.LegacyPrintf("service.referral", "invalidate inviter balance cache failed: %v", err)
		}
		if err := s.billingCacheService.InvalidateUserBalance(cacheCtx, inviteeID); err != nil {
			logger.LegacyPrintf("service.referral", "invalidate invitee balance cache failed: %v", err)
		}
	}()
}

func generateReferralCode() (string, error) {
	buf := make([]byte, referralCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	code := make([]byte, referralCodeLength)
	for i, b := range buf {
		code[i] = referralCodeCharset[int(b)%len(referralCodeCharset)]
	}
	return string(code), nil
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return email
	}
	local, domain := parts[0], parts[1]
	if len(local) <= 3 {
		return local + "***@" + domain
	}
	return local[:3] + "***@" + domain
}

func buildReferralLink(siteBaseURL, code string) string {
	base := strings.TrimSuffix(strings.TrimSpace(siteBaseURL), "/")
	if base == "" {
		return "?ref=" + code
	}
	return base + "/register?ref=" + code
}

func isUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "duplicate entry")
}
