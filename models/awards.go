package models

import (
  "time"

)

type Award struct {
  ID          uint           `gorm:"primaryKey" json:"id"`
  Name        string         `gorm:"size:255;not null" json:"name"`
  Description string         `gorm:"type:text" json:"description"`
  CreatedAt   time.Time      `json:"created_at"`
  UpdatedAt   time.Time      `json:"updated_at"`
  Points      int            `gorm:"default:0" json:"points"`
  IconURL     string         `gorm:"size:512" json:"icon_url"`
}

