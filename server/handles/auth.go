package handles

import (
	"bytes"
	"encoding/base64"
	"image/png"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
)

type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password"`
	OtpCode  string `json:"otp_code"`
}

// Login Deprecated
//
//	@Summary			User Login Deprecated
//	@Description		User login with username and password. Deprecated, use /api/auth/login/hash instead.
//	@Tags				Authentication
//	@Accept				json
//	@Produce			json
//	@Param				login	body		LoginReq										true	"Login request"
//	@Success			200		{object}	common.jsonResult{data=object{token=string}}	"Login successful, returns token"
//	@Failure			400		{object}	common.jsonResult{data=string}					"Bad Request"
//	@Failure			402		{object}	common.jsonResult{data=string}					"2FA Required"
//	@Failure			429		{object}	common.jsonResult{data=string}					"Too Many Requests"
//	@deprecatedRouter	/api/auth/login [post]
func Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Password = model.StaticHash(req.Password)
	loginHash(c, &req)
}

// LoginHash login with password hashed by sha256
//
//	@Summary		User login with pre-hashed password
//	@Description	Authenticate using username and pre-hashed password (SHA256)
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			login	body		LoginReq										true	"Login request"
//	@Success		200		{object}	common.jsonResult{data=object{token=string}}	"Login successful, returns token"
//	@Failure		400		{object}	common.jsonResult{data=string}					"Bad Request"
//	@Failure		402		{object}	common.jsonResult{data=string}					"2FA Required"
//	@Failure		429		{object}	common.jsonResult{data=string}					"Too Many Requests"
//	@Router			/api/auth/login/hash [post]
func LoginHash(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	loginHash(c, &req)
}

func loginHash(c *gin.Context, req *LoginReq) {
	// check count of login
	ip := c.ClientIP()
	count, ok := model.LoginCache.Get(ip)
	if ok && count >= model.DefaultMaxAuthRetries {
		common.ErrorStrResp(c, "Too many unsuccessful sign-in attempts have been made using an incorrect username or password, Try again later.", 429)
		model.LoginCache.Expire(ip, model.DefaultLockDuration)
		return
	}
	// check username
	user, err := op.GetUserByName(req.Username)
	if err != nil {
		common.ErrorResp(c, err, 400)
		model.LoginCache.Set(ip, count+1)
		return
	}
	// validate password hash
	if err := user.ValidatePwdStaticHash(req.Password); err != nil {
		common.ErrorResp(c, err, 400)
		model.LoginCache.Set(ip, count+1)
		return
	}
	// check 2FA
	if user.OtpSecret != "" {
		if !totp.Validate(req.OtpCode, user.OtpSecret) {
			common.ErrorStrResp(c, "Invalid 2FA code", 402)
			model.LoginCache.Set(ip, count+1)
			return
		}
	}
	// generate token
	token, err := common.GenerateToken(user)
	if err != nil {
		common.ErrorResp(c, err, 400, true)
		return
	}
	common.SuccessResp(c, gin.H{"token": token})
	model.LoginCache.Del(ip)
}

type UserResp struct {
	model.User
	Otp bool `json:"otp"`
}

// CurrentUser get current user by token
// if token is empty, return guest user
//
//	@Summary		Get Current User
//	@Description	Get current user information by token. If token is empty, returns guest user.
//	@Tags			User
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=UserResp}	"Current user information"
//	@Failure		401	{object}	common.jsonResult{data=string}		"Unauthorized"
//	@Router			/api/me [get]
//	@Security		Authorization
func CurrentUser(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	userResp := UserResp{
		User: *user,
	}
	userResp.Password = ""
	if userResp.OtpSecret != "" {
		userResp.Otp = true
	}
	common.SuccessResp(c, userResp)
}

// UpdateCurrent update current user profile
//
//	@Summary		Update Current User
//	@Description	Update current user profile information
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			user	body		model.User						true	"User profile to update"
//	@Success		200		{object}	common.jsonResult{data=string}	"Update successful"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		403		{object}	common.jsonResult{data=string}	"Forbidden"
//	@Router			/api/me [post]
//	@Security		Authorization
func UpdateCurrent(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not update profile", 403)
		return
	}
	user.Username = req.Username
	if req.Password != "" {
		user.SetPassword(req.Password)
	}
	user.SsoID = req.SsoID
	if err := op.UpdateUser(user); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

// Generate2FA generate 2FA secret
//
//	@Summary		Generate 2FA secret
//	@Description	Generate a new 2FA (TOTP) secret for current user
//	@Tags			Authentication
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=object{qr=string,secret=string}}	"Generated QR code and secret"
//	@Failure		403	{object}	common.jsonResult{data=string}							"Forbidden"
//	@Failure		500	{object}	common.jsonResult{data=string}							"Internal Server Error"
//	@Router			/api/auth/2fa/generate [post]
//	@Security		Authorization
func Generate2FA(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not generate 2FA code", 403)
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "OpenList",
		AccountName: user.Username,
	})
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	img, err := key.Image(400, 400)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	// to base64
	var buf bytes.Buffer
	png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	common.SuccessResp(c, gin.H{
		"qr":     "data:image/png;base64," + b64,
		"secret": key.Secret(),
	})
}

type Verify2FAReq struct {
	Code   string `json:"code" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// Verify2FA
//
//	@Summary		Verify and enable 2FA
//
//	@Description	Verify the provided 2FA code and enable 2FA for the current user
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			data	body		Verify2FAReq					true	"2FA verification data"
//	@Success		200		{object}	common.jsonResult{data=string}	"2FA enabled successfully"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		403		{object}	common.jsonResult{data=string}	"Forbidden"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/auth/2fa/verify [post]
//	@Security		Authorization
func Verify2FA(c *gin.Context) {
	var req Verify2FAReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	if user.IsGuest() {
		common.ErrorStrResp(c, "Guest user can not generate 2FA code", 403)
		return
	}
	if !totp.Validate(req.Code, req.Secret) {
		common.ErrorStrResp(c, "Invalid 2FA code", 400)
		return
	}
	user.OtpSecret = req.Secret
	if err := op.UpdateUser(user); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

// LogOut invalidate the token
//
//	@Summary		User Logout
//	@Description	Invalidate current session token
//	@Tags			Authentication
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Logout successful"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/auth/logout [get]
//	@Security		Authorization
func LogOut(c *gin.Context) {
	err := common.InvalidateToken(c.GetHeader("Authorization"))
	if err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}
