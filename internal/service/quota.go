package service

import (
	"errors"
	"math"
	"time"

	"ampmanager/internal/model"
	"ampmanager/internal/repository"
)

var (
	ErrUnknownLimitType = errors.New("未知的限制类型")
)

var farFuture = time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC)

type QuotaService struct {
	eventRepo repository.BillingEventRepositoryInterface
	subRepo   repository.UserSubscriptionRepositoryInterface
	planRepo  repository.SubscriptionPlanRepositoryInterface
}

func NewQuotaService() *QuotaService {
	return &QuotaService{
		eventRepo: repository.NewBillingEventRepository(),
		subRepo:   repository.NewUserSubscriptionRepository(),
		planRepo:  repository.NewSubscriptionPlanRepository(),
	}
}

func NewQuotaServiceWithRepo(
	eventRepo repository.BillingEventRepositoryInterface,
	subRepo repository.UserSubscriptionRepositoryInterface,
	planRepo repository.SubscriptionPlanRepositoryInterface,
) *QuotaService {
	return &QuotaService{eventRepo: eventRepo, subRepo: subRepo, planRepo: planRepo}
}

func GetWindowBounds(limitType model.LimitType, windowMode model.WindowMode, now time.Time, subscriptionStartsAt time.Time) (start, end time.Time, err error) {
	switch limitType {
	case model.LimitTypeDaily:
		if windowMode == model.WindowModeFixed {
			start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			end = start.Add(24 * time.Hour)
		} else {
			start = now.Add(-24 * time.Hour)
			end = now
		}
	case model.LimitTypeWeekly:
		if windowMode == model.WindowModeFixed {
			weekday := int(now.Weekday())
			if weekday == 0 {
				weekday = 7
			}
			start = time.Date(now.Year(), now.Month(), now.Day()-(weekday-1), 0, 0, 0, 0, time.UTC)
			end = start.AddDate(0, 0, 7)
		} else {
			start = now.AddDate(0, 0, -7)
			end = now
		}
	case model.LimitTypeMonthly:
		if windowMode == model.WindowModeFixed {
			start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			end = start.AddDate(0, 1, 0)
		} else {
			start = now.AddDate(0, -1, 0)
			end = now
		}
	case model.LimitTypeRolling5h:
		const windowSec int64 = 18000
		if windowMode == model.WindowModeFixed {
			unix := now.Unix()
			floorUnix := (unix / windowSec) * windowSec
			start = time.Unix(floorUnix, 0).UTC()
			end = start.Add(5 * time.Hour)
		} else {
			start = now.Add(-5 * time.Hour)
			end = now
		}
	case model.LimitTypeTotal:
		start = subscriptionStartsAt
		end = farFuture
	default:
		return time.Time{}, time.Time{}, ErrUnknownLimitType
	}
	return start, end, nil
}

func (s *QuotaService) GetSubscriptionRemaining(userID string) (int64, []model.WindowRemaining, error) {
	sub, err := s.subRepo.GetActiveByUserID(userID)
	if err != nil {
		return 0, nil, err
	}
	if sub == nil {
		return 0, nil, nil
	}

	_, limits, err := s.planRepo.GetByID(sub.PlanID)
	if err != nil {
		return 0, nil, err
	}
	if len(limits) == 0 {
		return 0, nil, nil
	}

	now := time.Now().UTC()
	windows := make([]model.WindowRemaining, 0, len(limits))
	minRemaining := int64(math.MaxInt64)

	for _, limit := range limits {
		start, end, err := GetWindowBounds(limit.LimitType, limit.WindowMode, now, sub.StartsAt)
		if err != nil {
			return 0, nil, err
		}

		used, err := s.eventRepo.GetUsageInWindow(sub.ID, start, end)
		if err != nil {
			return 0, nil, err
		}

		left := limit.LimitMicros - used
		if left < 0 {
			left = 0
		}

		windows = append(windows, model.WindowRemaining{
			LimitType:   limit.LimitType,
			WindowMode:  limit.WindowMode,
			LimitMicros: limit.LimitMicros,
			UsedMicros:  used,
			LeftMicros:  left,
			WindowStart: start,
			WindowEnd:   end,
		})

		if left < minRemaining {
			minRemaining = left
		}
	}

	if minRemaining == math.MaxInt64 {
		minRemaining = 0
	}

	return minRemaining, windows, nil
}

func (s *QuotaService) GetWindowsDetail(userID string) ([]model.WindowRemaining, error) {
	_, windows, err := s.GetSubscriptionRemaining(userID)
	return windows, err
}
