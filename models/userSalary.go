package models

import "time"

type UserSalary struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	UserID      uint      `json:"userId" gorm:"not null"`
	Amount      int       `json:"amount" gorm:"not null"`
	Attendance  int       `json:"attendance" gorm:"not null"`
	Absence     int       `json:"absence" gorm:"not null"`
	SalaryDate  time.Time `json:"salaryDate" gorm:"not null"`
	Bonus       int       `json:"bonus"`
	Penalty     int       `json:"penalty"`
	Status      bool      `json:"status" gorm:"default:false"`
	TotalSalary int       `json:"totalSalary" gorm:"not null"`

	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}
