package authentication

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"time"
)

type InviteCode struct {
	Code        string    `gorm:"primary_key"`
	ClaimedTime time.Time `gorm:"not null"`
	Claimed     bool      `gorm:"not null"`
	ClaimedBy   string    `gorm:"not null"`
}

func (a *Authenticator) RegisterUser(username, password, invitecode, payoutAddress string) bool {
	if !a.Claim(invitecode, username) {
		return false
	}

	// Register a new user through the 'web' handler
	form := url.Values{}
	form.Set("login", username)
	form.Set("password", password)
	form.Set("confirm_password", password)
	req := httptest.NewRequest("POST", "/auth/password/register?"+form.Encode(), nil)

	mux := a.NewServeMux()
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	fmt.Println(a.DB.Model(&User{}).Where("uid = ?", username).Update("payout_address", payoutAddress).Error)

	return true
}

func (a *Authenticator) NewCode(code string) error {
	return a.DB.Create(&InviteCode{Code: code}).Error
}

func (a *Authenticator) CodeUnclaimed(code string) bool {
	var i InviteCode
	dbErr := a.DB.Where("code = ?", code).Find(&i)
	if dbErr.Error != nil {
		// TODO: ?
		return false
	}

	return i.Code != "" && i.Claimed == false
}

func (a *Authenticator) Claim(code string, user string) bool {
	if !a.CodeUnclaimed(code) {
		return false
	}

	dbErr := a.DB.Model(&InviteCode{}).Where("code = ?", code).Updates(InviteCode{
		Code:        code,
		ClaimedTime: time.Now(),
		Claimed:     true,
		ClaimedBy:   user,
	})
	if dbErr.Error != nil {
		// TODO: ?
		return false
	}
	return dbErr.RowsAffected == 1
}
