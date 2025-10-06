package tokens

import "time"

type AuthToken struct {
	UserId       string
	DeviceId     string
	AccessToken  string
	RefreshToken string
	Created      time.Time
	Modified     time.Time
}
