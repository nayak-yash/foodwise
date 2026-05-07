package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"restaurantops/backend/internal/config"
	"restaurantops/backend/internal/domain"
	"restaurantops/backend/internal/persistence"
)

const branchID = domain.BranchID
const chickenID = domain.ChickenID

// Controllers compose a common transaction boundary and receive storage by interface.
type Controller struct{ db persistence.Database }
type QueryController struct{ *Controller }
type PurchasingController struct{ *Controller }
type InventoryController struct{ *Controller }
type IncidentController struct{ *Controller }
type RecoveryController struct{ *Controller }
type DemoController struct{ *Controller }

func NewRouter(database persistence.Database, settings config.Settings) *gin.Engine {
	shared := &Controller{db: database}
	queries := &QueryController{shared}
	purchasing := &PurchasingController{shared}
	inventory := &InventoryController{shared}
	incidents := &IncidentController{shared}
	recovery := &RecoveryController{shared}
	demo := &DemoController{shared}

	router := gin.New()
	router.Use(gin.Recovery(), otelgin.Middleware("restaurantops-backend"))
	registry := prometheus.NewRegistry()
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "restaurantops_http_requests_total",
		Help: "HTTP requests by route and result",
	}, []string{"method", "route", "status"})
	registry.MustRegister(requests)
	router.Use(func(c *gin.Context) {
		c.Next()
		requests.WithLabelValues(c.Request.Method, c.FullPath(), http.StatusText(c.Writer.Status())).Inc()
	})
	router.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	router.GET("/readyz", func(c *gin.Context) {
		if err := database.Ping(c.Request.Context()); err != nil {
			c.JSON(503, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(200, gin.H{"status": "ready"})
	})
	router.GET("/metrics", gin.WrapH(promhttp.HandlerFor(registry, promhttp.HandlerOpts{})))

	api := router.Group("/api/v1", authenticate(settings))
	api.GET("/overview", queries.overview)
	api.GET("/inventory", queries.inventory)
	api.GET("/catalog", queries.catalog)
	api.GET("/purchase-orders", queries.purchaseOrders)
	api.GET("/incidents", queries.incidents)
	api.GET("/incidents/:id", queries.incident)
	api.GET("/incidents/:id/evidence", queries.evidence)
	api.GET("/demand-history", queries.demandHistory)
	api.GET("/stock-requests", queries.stockRequests)
	api.POST("/purchase-orders", requireRole("manager"), purchasing.createPO)
	api.POST("/purchase-orders/:id/receipts", requireRole("manager"), purchasing.receivePO)
	api.POST("/orders", requireRole("manager"), inventory.createOrder)
	api.POST("/waste", requireRole("manager"), inventory.recordWaste)
	api.POST("/kitchen/metrics", requireRole("manager"), inventory.recordMetric)
	api.POST("/incidents", requireRole("agent"), incidents.createIncident)
	api.POST("/incidents/:id/report", requireRole("agent"), incidents.report)
	api.POST("/incidents/:id/plan", requireRole("agent"), incidents.plan)
	api.POST("/actions/:id/approve", requireRole("manager"), recovery.approve)
	api.POST("/actions/:id/reject", requireRole("manager"), recovery.reject)
	api.POST("/actions/:id/execute", recovery.execute)
	api.POST("/stock-requests/:id/receive", requireRole("manager"), recovery.receiveStock)
	if settings.DemoMode {
		api.POST("/demo/short-delivery", requireRole("manager"), demo.demo)
	}
	return router
}

func authenticate(settings config.Settings) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		role := ""
		if subtle.ConstantTimeCompare([]byte(token), []byte(settings.ManagerToken)) == 1 {
			role = "manager"
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(settings.AgentToken)) == 1 {
			role = "agent"
		}
		if role == "" {
			c.AbortWithStatusJSON(401, gin.H{"error": "valid bearer token required"})
			return
		}
		c.Set("role", role)
		c.Next()
	}
}

func requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("role") != role {
			c.AbortWithStatusJSON(403, gin.H{"error": "role not authorized"})
			return
		}
		c.Next()
	}
}
