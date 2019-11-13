package authentication_test

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/qor/mailer"

	. "github.com/FactomWyomingEntity/prosper-pool/authentication"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/require"
)

func TestAuthenticator_Exists(t *testing.T) {
	require := require.New(t)
	a := AuthForTests(t, true)
	defer a.DB.Close()

	RegisterUser(a, "test@gmail.com", "password")

	require.True(a.Exists("test@gmail.com"))
	require.False(a.Exists("unknown@gmail.com"))
}

func TestAuthenticator_Login(t *testing.T) {
	require := require.New(t)
	a := AuthForTests(t, true)
	defer a.DB.Close()

	RegisterUser(a, "test@gmail.com", "password")

	// 200 code is the register page since the user is not confirmed
	resp := LoginUser(a, "test@gmail.com", "password")
	require.Equal(200, resp.Code)

	// Confirm the user
	ConfirmUser(a, "test@gmail.com")

	// Test login
	resp = LoginUser(a, "test@gmail.com", "password")
	// 303 tells us to redirect to home, as in it worked!
	require.Equal(303, resp.Code)
}

func AuthForTests(t *testing.T, memory bool) *Authenticator {
	require := require.New(t)
	path := "tmp.db"
	if memory {
		path = ":memory:"
	}
	db, err := gorm.Open("sqlite3", path) //":memory:")
	require.NoError(err)

	a, _ := NewAuthenticator(viper.GetViper(), db)
	// Silence all mail stuff
	a.Config.Mailer.Sender = &EmptySender{}
	return a
}

func ConfirmUser(a *Authenticator, username string) {
	a.DB.Model(&HotfixedAuthIdentity{}).Where("uid = ?", username).Update("confirmed_at", time.Now())
}

func LoginUser(a *Authenticator, username, password string) *httptest.ResponseRecorder {
	// Register a new user
	form := url.Values{}
	form.Set("login", username)
	form.Set("password", password)
	req := httptest.NewRequest("POST", "/auth/password/login?"+form.Encode(), nil)

	mux := a.NewServeMux()
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	return resp
}

func RegisterUser(a *Authenticator, username, password string) {
	// Register a new user
	form := url.Values{}
	form.Set("login", username)
	form.Set("password", password)
	form.Set("confirm_password", password)
	req := httptest.NewRequest("POST", "/auth/password/register?"+form.Encode(), nil)

	mux := a.NewServeMux()
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
}

func dump(resp *httptest.ResponseRecorder) {
	d, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(resp.Code)
	fmt.Println(string(d))
}

type EmptySender struct {
}

func (EmptySender) Send(email mailer.Email) error {
	return nil
}
