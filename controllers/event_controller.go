package controllers

import (
	"net/http"
  "time"
  "fmt"
	"github.com/gin-gonic/gin"
	"github.com/open-cmuq/passport-backend/models"
	"github.com/open-cmuq/passport-backend/database"
)

// GetEvents retrieves a list of all events with optional filters
func GetEvents(c *gin.Context) {
	// Parse query parameters
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limit := c.Query("limit")

	// Build the query
	query := database.DB.Preload("Organizer").Preload("Awards")

	// Apply date range filter
	if startDate != "" {
		start, err := time.Parse(time.RFC3339, startDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start_date format. Use RFC3339 (e.g., 2023-10-01T00:00:00Z)"})
			return
		}
		query = query.Where("start_time >= ?", start)
	}
	if endDate != "" {
		end, err := time.Parse(time.RFC3339, endDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end_date format. Use RFC3339 (e.g., 2023-10-01T00:00:00Z)"})
			return
		}
		query = query.Where("end_time <= ?", end)
	}

	// Apply limit
	if limit != "" {
		var limitInt int
		if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil || limitInt <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit value. Must be a positive integer"})
			return
		}
		query = query.Limit(limitInt)
	}

	// Fetch events
	var events []models.Event
	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}

	userRole := c.GetString("user_role")

	// If the user is not an admin or staff, remove attendance information from the events
	if userRole != "admin" && userRole != "staff" {
		for i := range events {
		  events[i].Attendees = []models.Attendance{}
		}
	}

	c.JSON(http.StatusOK, events)
}

// CreateEvent creates a new event (requires admin permission)
func CreateEvent(c *gin.Context) {
	var input struct {
		Name            string     `json:"name"`
		Description     string     `json:"description"`
		Location        string     `json:"location"`
		StartTime       *time.Time `json:"start_time"` // Pointer to time.Time
		EndTime         *time.Time `json:"end_time"`   // Pointer to time.Time
		PointsAllocation int       `json:"points_allocation"`
		AwardIDs        []uint     `json:"award_ids"`
		ImageURL        string     `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate start and end times
	if (input.StartTime == nil && input.EndTime != nil) || (input.StartTime != nil && input.EndTime == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time must both be nil or both have values"})
		return
	}
	if input.StartTime != nil && input.EndTime != nil {
		if input.StartTime.After(*input.EndTime) || input.StartTime.Equal(*input.EndTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
			return
		}
	}

	// Get the organizer ID from the JWT token
	organizerID := c.GetUint("user_id")

	// Fetch the awards
	var awards []models.Award
	if err := database.DB.Where("id IN ?", input.AwardIDs).Find(&awards).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid award IDs"})
		return
	}

	// Create the event
	event := models.Event{
		Name:            input.Name,
		Description:     input.Description,
		Location:        input.Location,
		StartTime:       input.StartTime, // Assign the pointer
		EndTime:         input.EndTime,   // Assign the pointer
		OrganizerID:     organizerID,
		PointsAllocation: input.PointsAllocation,
		ImageURL:        input.ImageURL,
		Awards:          awards,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// GetEvent retrieves details of a specific event
func GetEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	var event models.Event

	// Fetch the event with all relationships
	if err := database.DB.Preload("Organizer").Preload("Awards").Preload("Attendees").First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Get the user's role from the JWT token or session
	userRole := c.GetString("user_role") // Assuming the role is stored in the JWT token as "user_role"

	// Check if the user is an admin or staff
	if userRole != "admin" && userRole != "staff" {
		// If the user is not an admin or staff, return the event without attendance information
		event.Attendees = []models.Attendance{}
	}

	c.JSON(http.StatusOK, event)
}

// UpdateEvent updates event details (requires admin permission)
func UpdateEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	var input struct {
		Name            string     `json:"name"`
		Description     string     `json:"description"`
		Location        string     `json:"location"`
		StartTime       *time.Time `json:"start_time"` // Pointer to time.Time
		EndTime         *time.Time `json:"end_time"`   // Pointer to time.Time
		PointsAllocation int      `json:"points_allocation"`
		AwardIDs        []uint    `json:"award_ids"`
		ImageURL        string    `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate start and end times
	if (input.StartTime == nil && input.EndTime != nil) || (input.StartTime != nil && input.EndTime == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time must both be nil or both have values"})
		return
	}
	if input.StartTime != nil && input.EndTime != nil {
		if input.StartTime.After(*input.EndTime) || input.StartTime.Equal(*input.EndTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "start_time must be before end_time"})
			return
		}
	}

	// Fetch the event
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Fetch the awards
	var awards []models.Award
	if err := database.DB.Where("id IN ?", input.AwardIDs).Find(&awards).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid award IDs"})
		return
	}

	// Update the event
	event.Name = input.Name
	event.Description = input.Description
	event.Location = input.Location
	if input.StartTime != nil {
		event.StartTime = input.StartTime
	} else {
		event.StartTime = nil // Explicitly set to nil if input.StartTime is nil
	}
	if input.EndTime != nil {
		event.EndTime = input.EndTime
	} else {
		event.EndTime = nil // Explicitly set to nil if input.EndTime is nil
	}
	event.PointsAllocation = input.PointsAllocation
	event.ImageURL = input.ImageURL
	event.Awards = awards

	if err := database.DB.Save(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// DeleteEvent deletes an event (requires admin permission)
func DeleteEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	if err := database.DB.Delete(&models.Event{}, eventID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Event deleted"})
}
