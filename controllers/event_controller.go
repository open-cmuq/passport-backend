package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/open-cmuq/passport-backend/database"
	"github.com/open-cmuq/passport-backend/models"
	"gorm.io/gorm"
)

// GetEvents retrieves a list of all events with optional filters
func GetEvents(c *gin.Context) {
	// Parse and validate query parameters
	queryParams := struct {
		BeforeTime  string `form:"before_time"`
		AfterTime   string `form:"after_time"`
		BetweenTime string `form:"between_time"` // comma-separated start,end
		Limit       string `form:"limit"`
		Order       string `form:"order"` // "asc" or "desc"
	}{
		Order: "desc", // default to newest first
	}

	if err := c.ShouldBindQuery(&queryParams); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	// Validate order parameter
	if queryParams.Order != "asc" && queryParams.Order != "desc" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order must be either 'asc' or 'desc'"})
		return
	}

	// Build the base query
	query := database.DB.Preload("Organizer").Preload("Awards")

	// Parse and validate time filters
	var beforeTime, afterTime time.Time
	var err error

	// Handle before_time
	if queryParams.BeforeTime != "" {
		beforeTime, err = time.Parse(time.RFC3339, queryParams.BeforeTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid before_time format. Use RFC3339 (e.g., 2023-10-01T00:00:00Z)"})
			return
		}
		query = query.Where("start_time < ?", beforeTime)
	}

	// Handle after_time
	if queryParams.AfterTime != "" {
		afterTime, err = time.Parse(time.RFC3339, queryParams.AfterTime)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid after_time format. Use RFC3339 (e.g., 2023-10-01T00:00:00Z)"})
			return
		}
		query = query.Where("start_time > ?", afterTime)
	}

	// Handle between_time (comma-separated start,end)
	if queryParams.BetweenTime != "" {
		times := strings.Split(queryParams.BetweenTime, ",")
		if len(times) != 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "between_time must contain exactly 2 comma-separated RFC3339 timestamps"})
			return
		}

		startTime, err := time.Parse(time.RFC3339, times[0])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start time in between_time"})
			return
		}

		endTime, err := time.Parse(time.RFC3339, times[1])
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end time in between_time"})
			return
		}

		if startTime.After(endTime) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start time must be before end time in between_time"})
			return
		}

		query = query.Where("start_time BETWEEN ? AND ?", startTime, endTime)
	}

	// Apply ordering
	query = query.Order(fmt.Sprintf("start_time %s", queryParams.Order))

	// Apply limit if specified
	if queryParams.Limit != "" {
		limit, err := strconv.Atoi(queryParams.Limit)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return
		}
		query = query.Limit(limit)
	}

	// Fetch events
	var events []models.Event
	if err := query.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve events"})
		return
	}

	c.JSON(http.StatusOK, events)
}

// CreateEvent creates a new event (requires admin permission)
func CreateEvent(c *gin.Context) {
	var input struct {
		Name             string     `json:"name"`
		Description      string     `json:"description"`
		Location         string     `json:"location"`
		StartTime        *time.Time `json:"start_time"` // Pointer to time.Time
		EndTime          *time.Time `json:"end_time"`   // Pointer to time.Time
		PointsAllocation int        `json:"points_allocation"`
		AwardIDs         *[]uint    `json:"award_ids"`
		ImageURL         string     `json:"image_url"`
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
	if input.AwardIDs != nil { // Only process if awards were specified
		if err := database.DB.Where("id IN ?", *input.AwardIDs).Find(&awards).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid award IDs"})
			return
		}
	}

	// Create the event
	event := models.Event{
		Name:             input.Name,
		Description:      input.Description,
		Location:         input.Location,
		StartTime:        input.StartTime, // Assign the pointer
		EndTime:          input.EndTime,   // Assign the pointer
		OrganizerID:      organizerID,
		PointsAllocation: input.PointsAllocation,
		ImageURL:         input.ImageURL,
		Awards:           awards,
	}
	if err := database.DB.Create(&event).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event"})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// GetEvent retrieves details of a specific event
// TODO Consider the case where we have 1000s of events, this would lead to blocking the API and slow performance.
// however, this is sufficient for our current needs
func GetEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	var event models.Event

	// Fetch the event with all relationships
	if err := database.DB.Preload("Organizer").Preload("Awards").Preload("Attendees").First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// TODO Consider sharing the event information as the user gets the event
	c.JSON(http.StatusOK, event)
}

// UpdateEvent updates event details (requires admin permission)
func UpdateEvent(c *gin.Context) {
	eventID := c.Param("eventId")
	var input struct {
		Name             string     `json:"name"`
		Description      string     `json:"description"`
		Location         string     `json:"location"`
		StartTime        *time.Time `json:"start_time"`
		EndTime          *time.Time `json:"end_time"`
		PointsAllocation int        `json:"points_allocation"`
		AwardIDs         []uint     `json:"award_ids"`
		ImageURL         string     `json:"image_url"`
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

	// Start a transaction
	tx := database.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Defer rollback in case of failure
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after Rollback
		}
	}()

	// Fetch the event within transaction
	var event models.Event
	if err := tx.First(&event, eventID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Fetch the awards within transaction if award IDs are provided
	var awards []models.Award
	if len(input.AwardIDs) > 0 {
		if err := tx.Where("id IN ?", input.AwardIDs).Find(&awards).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid award IDs"})
			return
		}
		// Verify we found all requested awards
		if len(awards) != len(input.AwardIDs) {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Some award IDs not found"})
			return
		}
	}

	// Update the event
	event.Name = input.Name
	event.Description = input.Description
	event.Location = input.Location
	event.StartTime = input.StartTime // nil or value both handled
	event.EndTime = input.EndTime     // nil or value both handled
	event.PointsAllocation = input.PointsAllocation
	event.ImageURL = input.ImageURL

	// Only update awards if new ones were provided
	if len(input.AwardIDs) > 0 {
		event.Awards = awards
	}

	// Save within transaction
	if err := tx.Save(&event).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// DeleteEvent deletes an event (requires admin permission)
func DeleteEvent(c *gin.Context) {
	eventID := c.Param("eventId")

	tx := database.DB.Begin()

	// 1. Get event points allocation
	var event models.Event
	if err := tx.First(&event, eventID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// 2. Get all attendance records
	var attendances []models.Attendance
	if err := tx.Where("event_id = ?", eventID).Find(&attendances).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find attendances"})
		return
	}

	// 3. Collect user IDs and deduct points
	userIDs := make([]uint, 0, len(attendances))
	for _, a := range attendances {
		userIDs = append(userIDs, a.UserID)
	}

	if len(userIDs) > 0 {
		if err := tx.Model(&models.User{}).Where("id IN ?", userIDs).
			Update("current_points", gorm.Expr("current_points - ?", event.PointsAllocation)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct points"})
			return
		}
	}

	// 4. Delete attendances
	if err := tx.Where("event_id = ?", eventID).Delete(&models.Attendance{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attendances"})
		return
	}

	// 5. Delete event
	if err := tx.Delete(&models.Event{}, eventID).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Event and related data deleted"})
}

// GetEventAttendees returns basic user info for event attendees
func GetEventAttendees(c *gin.Context) {
	eventID := c.Param("eventId")

	var attendees []struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Email    string `json:"email"`
		PhotoURL string `json:"photo_url"`
	}

	err := database.DB.Table("attendances").
		Select("users.id, users.name, users.email, users.photo_url").
		Joins("JOIN users ON users.id = attendances.user_id").
		Where("attendances.event_id = ?", eventID).
		Scan(&attendees).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve attendees"})
		return
	}

	c.JSON(http.StatusOK, attendees)
}

// AddAttendances handles adding attendance by user ID or email (supports bulk)
func AddAttendances(c *gin.Context) {
	eventID := c.Param("eventId")
	var input struct {
		Identifiers []string `json:"identifiers"` // Can be user IDs or emails
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get the event first to check points allocation
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Resolve identifiers to valid user IDs
	resolvedUsers, invalidIdentifiers := resolveValidIdentifiers(input.Identifiers)
	if len(resolvedUsers) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":             "No valid users found",
			"invalid_identifiers": invalidIdentifiers,
		})
		return
	}

	// Extract just the user IDs for processing
	userIDs := make([]uint, 0, len(resolvedUsers))
	for _, user := range resolvedUsers {
		userIDs = append(userIDs, user.ID)
	}

	// Process in transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. Find existing attendances to avoid duplicates
	var existingAttendances []models.Attendance
	if err := tx.Where("user_id IN ? AND event_id = ?", userIDs, eventID).Find(&existingAttendances).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check existing attendances"})
		return
	}

	// Create set of existing user IDs for quick lookup
	existingUsers := make(map[uint]bool)
	for _, att := range existingAttendances {
		existingUsers[att.UserID] = true
	}

	// Prepare batch insert
	var newAttendances []models.Attendance
	var usersToUpdate []uint
	now := time.Now()

	for _, userID := range userIDs {
		if existingUsers[userID] {
			continue
		}

		newAttendances = append(newAttendances, models.Attendance{
			UserID:      userID,
			EventID:     event.ID,
			ScannedTime: now,
		})
		usersToUpdate = append(usersToUpdate, userID)
	}

	// 2. Batch insert new attendances
	if len(newAttendances) > 0 {
		if err := tx.CreateInBatches(newAttendances, 100).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attendances"})
			return
		}
	}

	// 3. Batch update user points
	if len(usersToUpdate) > 0 {
		if err := tx.Model(&models.User{}).
			Where("id IN ?", usersToUpdate).
			Update("current_points", gorm.Expr("current_points + ?", event.PointsAllocation)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user points"})
			return
		}
	}

	// 4. Grant new awards to users who qualify after point update
	var newAwardsGranted int64 = 0
	if len(usersToUpdate) > 0 {
		// Execute raw SQL to insert qualifying awards, avoid adding duplicates
		result := tx.Exec(`
			INSERT INTO user_badges (user_id, award_id, created_at, updated_at)
			SELECT u.id, a.id, ?, ?
			FROM users u
			CROSS JOIN awards a
			LEFT JOIN user_badges ub ON ub.user_id = u.id AND ub.award_id = a.id
			WHERE u.id IN (?)
			  AND a.points <= u.current_points
			  AND ub.user_id IS NULL
		`, now, now, usersToUpdate)

		if result.Error != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant awards"})
			return
		}
		newAwardsGranted = result.RowsAffected
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Get details of processed users for response
	var users []models.User
	if len(usersToUpdate) > 0 {
		database.DB.Preload("AwardsEarned").Where("id IN ?", usersToUpdate).Find(&users)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Attendance processed",
		"new_attendees":       len(newAttendances),
		"duplicates":          len(userIDs) - len(newAttendances),
		"points_added":        event.PointsAllocation * len(newAttendances),
		"new_awards_granted":  newAwardsGranted,
		"processed_users":     users,
		"invalid_identifiers": invalidIdentifiers,
	})
}

// DeleteAttendances handles removing attendance by user ID or email (supports bulk)
func DeleteAttendances(c *gin.Context) {
	eventID := c.Param("eventId")
	var input struct {
		Identifiers []string `json:"identifiers"` // Can be user IDs or emails
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Get the event first to check points allocation
	var event models.Event
	if err := database.DB.First(&event, eventID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
		return
	}

	// Resolve identifiers to valid user IDs
	resolvedUsers, invalidIdentifiers := resolveValidIdentifiers(input.Identifiers)
	if len(resolvedUsers) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":             "No valid users found",
			"invalid_identifiers": invalidIdentifiers,
		})
		return
	}

	// Extract just the user IDs for processing
	userIDs := make([]uint, 0, len(resolvedUsers))
	for _, user := range resolvedUsers {
		userIDs = append(userIDs, user.ID)
	}

	// Process in transaction
	tx := database.DB.Begin()

	// Delete attendances
	result := tx.Where("event_id = ? AND user_id IN ?", eventID, userIDs).
		Delete(&models.Attendance{})
	if result.Error != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete attendances"})
		return
	}

	// Deduct points from users
	if err := tx.Model(&models.User{}).Where("id IN ?", userIDs).
		Update("current_points", gorm.Expr("current_points - ?", event.PointsAllocation)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct points"})
		return
	}

	tx.Commit()

	// Get details of processed users for response
	var users []models.User
	database.DB.Where("id IN ?", userIDs).Find(&users)

	c.JSON(http.StatusOK, gin.H{
		"message":             "Attendance removed",
		"removed_count":       result.RowsAffected,
		"points_deducted":     event.PointsAllocation * int(result.RowsAffected),
		"processed_users":     users,
		"invalid_identifiers": invalidIdentifiers,
	})
}

// Helper function to resolve mixed identifiers (IDs or emails) to valid user records
func resolveValidIdentifiers(identifiers []string) ([]models.User, []string) {
	var users []models.User
	var invalidIdentifiers []string

	// Separate numeric IDs and emails
	var potentialIDs []uint
	var emails []string

	for _, identifier := range identifiers {
		if id, err := strconv.Atoi(identifier); err == nil {
			potentialIDs = append(potentialIDs, uint(id))
		} else {
			emails = append(emails, identifier)
		}
	}

	// Find users by ID
	if len(potentialIDs) > 0 {
		var idUsers []models.User
		if err := database.DB.Where("id IN ?", potentialIDs).Find(&idUsers).Error; err == nil {
			users = append(users, idUsers...)

			// Track which IDs weren't found
			foundIDs := make(map[uint]bool)
			for _, user := range idUsers {
				foundIDs[user.ID] = true
			}

			for _, id := range potentialIDs {
				if !foundIDs[id] {
					invalidIdentifiers = append(invalidIdentifiers, fmt.Sprintf("%d", id))
				}
			}
		}
	}

	// Find users by email
	if len(emails) > 0 {
		var emailUsers []models.User
		if err := database.DB.Where("email IN ?", emails).Find(&emailUsers).Error; err == nil {
			users = append(users, emailUsers...)

			// Track which emails weren't found
			foundEmails := make(map[string]bool)
			for _, user := range emailUsers {
				foundEmails[user.Email] = true
			}

			for _, email := range emails {
				if !foundEmails[email] {
					invalidIdentifiers = append(invalidIdentifiers, email)
				}
			}
		}
	}

	return users, invalidIdentifiers
}
