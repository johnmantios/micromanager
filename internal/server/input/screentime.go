package input

import "time"

type Screentime struct {
	UserID   string    `path:"userID" maxLength:"40"`
	Username string    `path:"username" maxLength:"20"`
	Date     time.Time `path:"date"`
	Minutes  int       `path:"minutes"`
}
