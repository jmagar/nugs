// @title Nugs Collection API
// @version 1.0.0
// @description Comprehensive REST API for managing music collections, downloads, monitoring, and analytics with enterprise-grade features including background job processing, webhook notifications, and detailed analytics.
// @termsOfService https://example.com/terms

// @contact.name Nugs Collection API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmagar/nugs/cron/docs"
	"github.com/jmagar/nugs/cron/internal/api/handlers"
	"github.com/jmagar/nugs/cron/internal/api/middleware"
	"github.com/jmagar/nugs/cron/internal/database"
	"github.com/jmagar/nugs/cron/internal/models"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Config holds the API server configuration
type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	JWTSecret   []byte
}

func main() {
	// Load configuration
	config := loadConfig()

	// Gate emoji logs for non-production environments only
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = os.Getenv("GO_ENV")
	}
	if env == "" {
		env = "production" // Default to production
	}
	
	if env != "production" {
		log.Printf("🔥 Hot reload test: API server starting on port %s - changes detected!", config.Port)
	} else {
		log.Printf("API server starting on port %s", config.Port)
	}

	// Initialize database
	db, err := database.Initialize(config.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Setup router with database connection
	router := setupRouter(config, db)

	// Start server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("Server startup failed:", err)
	}
}

func setupRouter(config *Config, db *sql.DB) *gin.Engine {
	// Set Gin mode based on environment
	if config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize job manager
	jobManager := models.NewJobManager()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, config.JWTSecret)
	catalogHandler := handlers.NewCatalogHandler(db)
	refreshHandler := handlers.NewRefreshHandler(db, jobManager)
	downloadHandler := handlers.NewDownloadHandler(db, jobManager)
	monitoringHandler := handlers.NewMonitoringHandler(db, jobManager)
	analyticsHandler := handlers.NewAnalyticsHandler(db, jobManager)
	webhookHandler := handlers.NewWebhookHandler(db, jobManager)
	adminHandler := handlers.NewAdminHandler(db, jobManager)
	schedulerHandler := handlers.NewSchedulerHandler(db, jobManager)

	// Global middleware
	router.Use(middleware.Logger())
	router.Use(middleware.ErrorHandler())
	router.Use(middleware.RequestID())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS([]string{"*"})) // Configure as needed
	router.Use(middleware.ContentType())

	// Health check endpoint
	router.GET("/health", middleware.HealthCheck())

	// Swagger documentation endpoint
	docs.SwaggerInfo.BasePath = "/api/v1"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Nugs Collection API v1.0.0",
				"docs":    "/docs",
			})
		})

		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
		}

		// Debug endpoint (temporary)
		v1.GET("/debug/users", func(c *gin.Context) {
			rows, err := db.Query("SELECT id, username, email, role, active FROM users")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			var users []map[string]interface{}
			for rows.Next() {
				var id int
				var username, email, role string
				var active bool
				if err := rows.Scan(&id, &username, &email, &role, &active); err != nil {
					log.Printf("Error scanning row: %v", err)
					continue
				}
				users = append(users, map[string]interface{}{
					"id": id, "username": username, "email": email, "role": role, "active": active,
				})
			}
			
			// Check for errors from iterating over rows
			if err := rows.Err(); err != nil {
				log.Printf("Error iterating rows: %v", err)
			}
			
			c.JSON(http.StatusOK, gin.H{"users": users})
		})

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth(string(config.JWTSecret)))
		{
			// Auth verification
			protected.GET("/auth/verify", authHandler.Verify)

			// Catalog endpoints
			catalog := protected.Group("/catalog")
			{
				// Artists
				catalog.GET("/artists", catalogHandler.GetArtists)
				catalog.GET("/artists/:id", catalogHandler.GetArtist)
				catalog.GET("/artists/:id/shows", catalogHandler.GetArtistShows)

				// Shows
				catalog.GET("/shows/search", catalogHandler.SearchShows)
				catalog.GET("/shows/:id", catalogHandler.GetShow)

				// Refresh endpoints
				catalog.POST("/refresh", refreshHandler.StartRefresh)
				catalog.GET("/refresh/status/:job_id", refreshHandler.GetRefreshStatus)
				catalog.GET("/refresh/jobs", refreshHandler.ListRefreshJobs)
				catalog.DELETE("/refresh/:job_id", refreshHandler.CancelRefresh)
				catalog.GET("/refresh/info", refreshHandler.GetRefreshInfo)
			}

			// Download endpoints
			downloads := protected.Group("/downloads")
			{
				downloads.GET("/", downloadHandler.GetDownloads)
				downloads.POST("/queue", downloadHandler.QueueDownload)
				downloads.GET("/queue", downloadHandler.GetDownloadQueue)
				downloads.POST("/queue/reorder", downloadHandler.ReorderQueue)
				downloads.GET("/stats", downloadHandler.GetDownloadStats)
				downloads.GET("/:id", downloadHandler.GetDownload)
				downloads.DELETE("/:id", downloadHandler.CancelDownload)
			}

			// Monitoring endpoints
			monitoring := protected.Group("/monitoring")
			{
				// Monitor management
				monitoring.POST("/monitors", monitoringHandler.CreateMonitor)
				monitoring.POST("/monitors/bulk", monitoringHandler.CreateBulkMonitors)
				monitoring.GET("/monitors", monitoringHandler.GetMonitors)
				monitoring.GET("/monitors/:id", monitoringHandler.GetMonitor)
				monitoring.PUT("/monitors/:id", monitoringHandler.UpdateMonitor)
				monitoring.DELETE("/monitors/:id", monitoringHandler.DeleteMonitor)

				// Monitoring execution
				monitoring.POST("/check/all", monitoringHandler.CheckAllMonitors)
				monitoring.POST("/check/artist/:id", monitoringHandler.CheckArtist)

				// Alerts
				monitoring.GET("/alerts", monitoringHandler.GetAlerts)
				monitoring.PUT("/alerts/:id/acknowledge", monitoringHandler.AcknowledgeAlert)

				// Statistics
				monitoring.GET("/stats", monitoringHandler.GetMonitoringStats)
			}

			// Analytics endpoints
			analytics := protected.Group("/analytics")
			{
				// Report generation
				analytics.POST("/reports", analyticsHandler.GenerateReport)

				// Core analytics
				analytics.GET("/collection", analyticsHandler.GetCollectionStats)
				analytics.GET("/artists", analyticsHandler.GetArtistAnalytics)
				analytics.GET("/downloads", analyticsHandler.GetDownloadAnalytics)
				analytics.GET("/system", analyticsHandler.GetSystemMetrics)
				analytics.GET("/performance", analyticsHandler.GetPerformanceMetrics)

				// Top lists
				analytics.GET("/top/artists", analyticsHandler.GetTopArtists)
				analytics.GET("/top/venues", analyticsHandler.GetTopVenues)

				// Trends and insights
				analytics.GET("/trends/downloads", analyticsHandler.GetDownloadTrends)

				// Dashboard
				analytics.GET("/summary", analyticsHandler.GetDashboardSummary)
				analytics.GET("/health", analyticsHandler.GetHealthScore)
			}

			// Webhook endpoints
			webhooks := protected.Group("/webhooks")
			{
				// Webhook management
				webhooks.POST("/", webhookHandler.CreateWebhook)
				webhooks.GET("/", webhookHandler.GetWebhooks)
				webhooks.GET("/:id", webhookHandler.GetWebhook)
				webhooks.PUT("/:id", webhookHandler.UpdateWebhook)
				webhooks.DELETE("/:id", webhookHandler.DeleteWebhook)

				// Webhook testing
				webhooks.POST("/:id/test", webhookHandler.TestWebhook)

				// Delivery tracking
				webhooks.GET("/:id/deliveries", webhookHandler.GetWebhookDeliveries)
				webhooks.GET("/deliveries", webhookHandler.GetAllDeliveries)

				// Webhook information
				webhooks.GET("/events", webhookHandler.GetAvailableEvents)
				webhooks.GET("/stats", webhookHandler.GetWebhookStats)
			}

			// Admin endpoints (require admin role in production)
			admin := protected.Group("/admin")
			{
				// User management
				admin.POST("/users", adminHandler.CreateUser)
				admin.GET("/users", adminHandler.GetUsers)
				admin.PUT("/users/:id", adminHandler.UpdateUser)
				admin.DELETE("/users/:id", adminHandler.DeleteUser)

				// System configuration
				admin.GET("/config", adminHandler.GetSystemConfig)
				admin.PUT("/config/:key", adminHandler.UpdateConfig)

				// System status and health
				admin.GET("/status", adminHandler.GetSystemStatus)
				admin.GET("/stats", adminHandler.GetAdminStats)

				// Maintenance operations
				admin.POST("/maintenance/cleanup", adminHandler.RunCleanup)

				// Audit logs
				admin.GET("/audit", adminHandler.GetAuditLogs)

				// Job management
				admin.GET("/jobs", adminHandler.GetJobs)
				admin.GET("/jobs/:id", adminHandler.GetJob)
				admin.DELETE("/jobs/:id", adminHandler.CancelJob)

				// Database management
				admin.POST("/database/backup", adminHandler.CreateDatabaseBackup)
				admin.POST("/database/optimize", adminHandler.OptimizeDatabase)
				admin.GET("/database/stats", adminHandler.GetDatabaseStats)
			}

			// Scheduler endpoints (background job scheduling)
			scheduler := protected.Group("/scheduler")
			{
				// Scheduler control
				scheduler.POST("/start", schedulerHandler.StartScheduler)
				scheduler.POST("/stop", schedulerHandler.StopScheduler)
				scheduler.GET("/status", schedulerHandler.GetSchedulerStatus)
				scheduler.GET("/stats", schedulerHandler.GetSchedulerStats)

				// Schedule management
				scheduler.POST("/schedules", schedulerHandler.CreateSchedule)
				scheduler.GET("/schedules", schedulerHandler.GetSchedules)
				scheduler.GET("/schedules/:id", schedulerHandler.GetSchedule)
				scheduler.PUT("/schedules/:id", schedulerHandler.UpdateSchedule)
				scheduler.DELETE("/schedules/:id", schedulerHandler.DeleteSchedule)
				scheduler.POST("/schedules/bulk", schedulerHandler.BulkScheduleOperation)

				// Execution tracking
				scheduler.GET("/schedules/:id/executions", schedulerHandler.GetScheduleExecutions)
				scheduler.GET("/executions", schedulerHandler.GetAllExecutions)

				// Templates and helpers
				scheduler.GET("/templates", schedulerHandler.GetScheduleTemplates)
				scheduler.GET("/cron-patterns", schedulerHandler.GetCronPatterns)
			}
		}
	}

	return router
}

func loadConfig() *Config {
	config := &Config{
		Port:        "8080",
		Environment: "development",
		DatabaseURL: "./data/nugs_api.db",
		JWTSecret:   []byte("change-this-in-production"),
	}

	// Override with environment variables
	if port := os.Getenv("API_PORT"); port != "" {
		config.Port = port
	}

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		config.Environment = env
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.DatabaseURL = dbURL
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.JWTSecret = []byte(jwtSecret)
	}

	return config
}
