package controllers

import (
	"net/http"
	"time"
	"xiaozhi/manager/backend/storage"

	"github.com/gin-gonic/gin"
)

// PoolStatsController is the connection pool statistics controller
type PoolStatsController struct {
	storage *storage.PoolStatsStorage
}

// NewPoolStatsController creates a new connection pool statistics controller
func NewPoolStatsController() *PoolStatsController {
	return &PoolStatsController{
		storage: storage.GetPoolStatsStorage(),
	}
}

// ReportPoolStats receives statistics reported by the main service (internal endpoint, no auth required)
func (c *PoolStatsController) ReportPoolStats(ctx *gin.Context) {
	var request struct {
		Stats map[string]interface{} `json:"stats" binding:"required"`
	}

	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters: " + err.Error()})
		return
	}

	// Save statistics data
	c.storage.AddStats(request.Stats)

	ctx.JSON(http.StatusOK, gin.H{
		"message":   "stats reported successfully",
		"timestamp": time.Now().Unix(),
	})
}

// GetPoolStats retrieves connection pool statistics (admin endpoint)
func (c *PoolStatsController) GetPoolStats(ctx *gin.Context) {
	// Get query parameter
	queryType := ctx.DefaultQuery("type", "latest") // latest, all, range

	switch queryType {
	case "latest":
		// Get the latest data
		latest := c.storage.GetLatestStats()
		if latest == nil {
			ctx.JSON(http.StatusOK, gin.H{
				"data":    nil,
				"message": "no stats available",
			})
			return
		}
		ctx.JSON(http.StatusOK, gin.H{
			"data": latest,
		})

	case "all":
		// Get all data (last 24 hours)
		allStats := c.storage.GetAllStats()
		ctx.JSON(http.StatusOK, gin.H{
			"data":  allStats,
			"count": len(allStats),
		})

	case "range":
		// Get data by time range
		startStr := ctx.Query("start")
		endStr := ctx.Query("end")

		if startStr == "" || endStr == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "time range parameters start and end are required"})
			return
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time format, use RFC3339"})
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time format, use RFC3339"})
			return
		}

		stats := c.storage.GetStatsByTimeRange(start, end)
		ctx.JSON(http.StatusOK, gin.H{
			"data":  stats,
			"count": len(stats),
		})

	default:
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid query type, supported: latest, all, range"})
	}
}

// GetPoolStatsSummary retrieves a statistics summary
func (c *PoolStatsController) GetPoolStatsSummary(ctx *gin.Context) {
	latest := c.storage.GetLatestStats()

	summary := gin.H{
		"total_records":    0,
		"storage_duration": "latest record only",
		"oldest_timestamp": nil,
		"newest_timestamp": nil,
	}

	if latest != nil {
		summary["total_records"] = 1
		summary["newest_timestamp"] = latest.Timestamp
		summary["oldest_timestamp"] = latest.Timestamp
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}
