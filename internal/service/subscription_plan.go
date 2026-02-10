package service

import (
	"errors"

	"ampmanager/internal/database"
	"ampmanager/internal/model"
	"ampmanager/internal/repository"
)

var (
	ErrPlanNameRequired    = errors.New("套餐名称不能为空")
	ErrInvalidLimitType    = errors.New("无效的限制类型")
	ErrInvalidWindowMode   = errors.New("无效的窗口模式")
	ErrPlanHasActiveSubs   = errors.New("该套餐下存在活跃订阅，无法删除")
	ErrPlanNotFound        = errors.New("套餐不存在")
	ErrDuplicateLimitType  = errors.New("同一限制类型不能重复")
)

type SubscriptionPlanService struct {
	planRepo repository.SubscriptionPlanRepositoryInterface
	subRepo  repository.UserSubscriptionRepositoryInterface
}

func NewSubscriptionPlanService() *SubscriptionPlanService {
	return &SubscriptionPlanService{
		planRepo: repository.NewSubscriptionPlanRepository(),
		subRepo:  repository.NewUserSubscriptionRepository(),
	}
}

func NewSubscriptionPlanServiceWithRepo(
	planRepo repository.SubscriptionPlanRepositoryInterface,
	subRepo repository.UserSubscriptionRepositoryInterface,
) *SubscriptionPlanService {
	return &SubscriptionPlanService{planRepo: planRepo, subRepo: subRepo}
}

var validLimitTypes = map[model.LimitType]bool{
	model.LimitTypeDaily:     true,
	model.LimitTypeWeekly:    true,
	model.LimitTypeMonthly:   true,
	model.LimitTypeRolling5h: true,
	model.LimitTypeTotal:     true,
}

var validWindowModes = map[model.WindowMode]bool{
	model.WindowModeFixed:   true,
	model.WindowModeSliding: true,
}

func (s *SubscriptionPlanService) validateLimits(limits []model.PlanLimitRequest) error {
	seen := make(map[model.LimitType]bool)
	for _, l := range limits {
		if !validLimitTypes[l.LimitType] {
			return ErrInvalidLimitType
		}
		if !validWindowModes[l.WindowMode] {
			return ErrInvalidWindowMode
		}
		if seen[l.LimitType] {
			return ErrDuplicateLimitType
		}
		seen[l.LimitType] = true
	}
	return nil
}

func (s *SubscriptionPlanService) Create(req *model.SubscriptionPlanRequest) (*model.SubscriptionPlanResponse, error) {
	if req.Name == "" {
		return nil, ErrPlanNameRequired
	}
	if err := s.validateLimits(req.Limits); err != nil {
		return nil, err
	}

	plan := &model.SubscriptionPlan{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
	}

	limits := make([]model.SubscriptionPlanLimit, len(req.Limits))
	for i, l := range req.Limits {
		limits[i] = model.SubscriptionPlanLimit{
			LimitType:   l.LimitType,
			WindowMode:  l.WindowMode,
			LimitMicros: l.LimitMicros,
		}
	}

	if err := s.planRepo.Create(plan, limits); err != nil {
		return nil, err
	}

	return &model.SubscriptionPlanResponse{
		ID:          plan.ID,
		Name:        plan.Name,
		Description: plan.Description,
		Enabled:     plan.Enabled,
		Limits:      limits,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}, nil
}

func (s *SubscriptionPlanService) GetByID(id string) (*model.SubscriptionPlanResponse, error) {
	plan, limits, err := s.planRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrPlanNotFound
	}

	return &model.SubscriptionPlanResponse{
		ID:          plan.ID,
		Name:        plan.Name,
		Description: plan.Description,
		Enabled:     plan.Enabled,
		Limits:      limits,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}, nil
}

func (s *SubscriptionPlanService) List() ([]*model.SubscriptionPlanResponse, error) {
	plans, limitsMap, err := s.planRepo.List()
	if err != nil {
		return nil, err
	}

	result := make([]*model.SubscriptionPlanResponse, len(plans))
	for i, p := range plans {
		result[i] = &model.SubscriptionPlanResponse{
			ID:          p.ID,
			Name:        p.Name,
			Description: p.Description,
			Enabled:     p.Enabled,
			Limits:      limitsMap[p.ID],
			CreatedAt:   p.CreatedAt,
			UpdatedAt:   p.UpdatedAt,
		}
	}
	return result, nil
}

func (s *SubscriptionPlanService) Update(id string, req *model.SubscriptionPlanRequest) (*model.SubscriptionPlanResponse, error) {
	if req.Name == "" {
		return nil, ErrPlanNameRequired
	}
	if err := s.validateLimits(req.Limits); err != nil {
		return nil, err
	}

	existing, _, err := s.planRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPlanNotFound
	}

	plan := &model.SubscriptionPlan{
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
	}

	limits := make([]model.SubscriptionPlanLimit, len(req.Limits))
	for i, l := range req.Limits {
		limits[i] = model.SubscriptionPlanLimit{
			LimitType:   l.LimitType,
			WindowMode:  l.WindowMode,
			LimitMicros: l.LimitMicros,
		}
	}

	if err := s.planRepo.Update(id, plan, limits); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

func (s *SubscriptionPlanService) Delete(id string) error {
	existing, _, err := s.planRepo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrPlanNotFound
	}

	db := database.GetDB()
	var count int
	err = db.QueryRow(
		`SELECT COUNT(*) FROM user_subscriptions WHERE plan_id = ? AND status = 'active'`,
		id,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrPlanHasActiveSubs
	}

	return s.planRepo.Delete(id)
}

func (s *SubscriptionPlanService) SetEnabled(id string, enabled bool) error {
	return s.planRepo.SetEnabled(id, enabled)
}
