package handler

import (
	"net/http"
	"order-management-api/internal/domain"
	"order-management-api/internal/middleware"
	"order-management-api/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// CreateOrder godoc
// @Summary      Create a new order
// @Tags         orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body  service.CreateOrderInput  true  "Create order input"
// @Success      201   {object}  domain.Order
// @Failure      400   {object}  map[string]string
// @Router       /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var input service.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.Create(c.Request.Context(), userID, input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// GetOrder godoc
// @Summary      Get order by ID
// @Tags         orders
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Order ID"
// @Success      200  {object}  domain.Order
// @Failure      404  {object}  map[string]string
// @Router       /orders/{id} [get]
func (h *OrderHandler) GetOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID := c.Param("id")
	order, err := h.orderService.GetByID(c.Request.Context(), orderID, userID)
	if err != nil {
		if err == service.ErrOrderNotFound || err == service.ErrUnauthorized {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders godoc
// @Summary      List orders for current user
// @Tags         orders
// @Security     BearerAuth
// @Produce      json
// @Param        limit   query     int  false  "Limit"   default(20)
// @Param        offset  query     int  false  "Offset"  default(0)
// @Success      200     {object}  service.OrderListResponse
// @Router       /orders [get]
func (h *OrderHandler) ListOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	resp, err := h.orderService.GetByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateOrderStatus godoc
// @Summary      Update order status
// @Tags         orders
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string  true  "Order ID"
// @Param        body  body      service.UpdateOrderStatusInput  true  "Status"
// @Success      200   {object}  domain.Order
// @Failure      400   {object}  map[string]string
// @Failure      404   {object}  map[string]string
// @Router       /orders/{id}/status [patch]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	orderID := c.Param("id")
	var input service.UpdateOrderStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validStatuses := map[domain.OrderStatus]bool{
		domain.OrderStatusPending: true, domain.OrderStatusConfirmed: true,
		domain.OrderStatusShipped: true, domain.OrderStatusDelivered: true,
		domain.OrderStatusCancelled: true,
	}
	if !validStatuses[input.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	order, err := h.orderService.UpdateStatus(c.Request.Context(), orderID, userID, input)
	if err != nil {
		if err == service.ErrOrderNotFound || err == service.ErrUnauthorized {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, order)
}
