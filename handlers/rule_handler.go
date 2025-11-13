package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// RuleHandler handles rule endpoints
type RuleHandler struct {
	ruleService             *service.RuleService
	authorizationMiddleware *middleware.AuthorizationMiddleware
}

// NewRuleHandler creates a new rule handler
func NewRuleHandler(
	ruleService *service.RuleService,
	authzMiddleware *middleware.AuthorizationMiddleware,
) *RuleHandler {
	return &RuleHandler{
		ruleService:             ruleService,
		authorizationMiddleware: authzMiddleware,
	}
}

// AddRule handles add rule request
// POST /api/rules
func (h *RuleHandler) AddRule(c *fiber.Ctx) error {
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
func (h *RuleHandler) UpdateRule(c *fiber.Ctx) error {
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
func (h *RuleHandler) RemoveRule(c *fiber.Ctx) error {
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
func (h *RuleHandler) ListRules(c *fiber.Ctx) error {
	rules, err := h.ruleService.ListRules()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    rules,
	})
}
