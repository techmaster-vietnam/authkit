package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseRuleHandler là generic rule handler
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface] struct {
	ruleService             *service.RuleService
	authorizationMiddleware *middleware.BaseAuthorizationMiddleware[TUser, TRole]
}

// NewBaseRuleHandler tạo mới BaseRuleHandler với generic types
func NewBaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface](
	ruleService *service.RuleService,
	authzMiddleware *middleware.BaseAuthorizationMiddleware[TUser, TRole],
) *BaseRuleHandler[TUser, TRole] {
	return &BaseRuleHandler[TUser, TRole]{
		ruleService:             ruleService,
		authorizationMiddleware: authzMiddleware,
	}
}

// AddRule handles add rule request
// POST /api/rules
func (h *BaseRuleHandler[TUser, TRole]) AddRule(c *fiber.Ctx) error {
	var req service.AddRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	rule, err := h.ruleService.AddRule(req)
	if err != nil {
		return err
	}

	// Invalidate cache after adding rule
	h.authorizationMiddleware.InvalidateCache()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    rule,
	})
}

// UpdateRule handles update rule request
// PUT /api/rules/:id
func (h *BaseRuleHandler[TUser, TRole]) UpdateRule(c *fiber.Ctx) error {
	ruleID := c.Params("id")
	if ruleID == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": ruleID,
		})
	}

	var req service.UpdateRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	rule, err := h.ruleService.UpdateRule(ruleID, req)
	if err != nil {
		return err
	}

	// Invalidate cache after updating rule
	h.authorizationMiddleware.InvalidateCache()

	return c.JSON(fiber.Map{
		"success": true,
		"data":    rule,
	})
}

// RemoveRule handles remove rule request
// DELETE /api/rules/:id
func (h *BaseRuleHandler[TUser, TRole]) RemoveRule(c *fiber.Ctx) error {
	ruleID := c.Params("id")
	if ruleID == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": ruleID,
		})
	}

	if err := h.ruleService.RemoveRule(ruleID); err != nil {
		return err
	}

	// Invalidate cache after removing rule
	h.authorizationMiddleware.InvalidateCache()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Xóa rule thành công",
	})
}

// ListRules handles list rules request
// GET /api/rules
func (h *BaseRuleHandler[TUser, TRole]) ListRules(c *fiber.Ctx) error {
	rules, err := h.ruleService.ListRules()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    rules,
	})
}

