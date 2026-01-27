package handles

import (
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// LoginReq represents ldap login request
//
//	@Summary		LDAP login
//	@Description	Authenticate user via LDAP directory service
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			login	body		LoginReq												true	"Login info"
//	@Success		200		{object}	common.jsonResult{data=common.mapResult{token=string}}	"Login successful, return token"
//	@Failure		400		{object}	common.jsonResult{data=string}							"Bad Request"
//	@Failure		403		{object}	common.jsonResult{data=string}							"Forbidden"
//	@Failure		429		{object}	common.jsonResult{data=string}							"Too Many Requests"
//	@Router			/api/auth/ldap/login [post]
func LoginLdap(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	enabled := setting.GetBool(conf.LdapLoginEnabled)
	if !enabled {
		common.ErrorStrResp(c, "ldap is not enabled", 403)
		return
	}
	user, err := op.GetUserByName(req.Username)
	if err == nil && !user.AllowLdap {
		common.ErrorStrResp(c, "login via ldap is not allowed", 403)
		return
	}

	// check count of login
	ip := c.ClientIP()
	count, ok := model.LoginCache.Get(ip)
	if ok && count >= model.DefaultMaxAuthRetries {
		common.ErrorStrResp(c, "Too many unsuccessful sign-in attempts have been made using an incorrect username or password, Try again later.", 429)
		model.LoginCache.Expire(ip, model.DefaultLockDuration)
		return
	}

	err = common.HandleLdapLogin(req.Username, req.Password)
	if err != nil {
		if errors.Is(err, common.ErrFailedLdapAuth) {
			model.LoginCache.Set(ip, count+1)
			common.ErrorResp(c, err, 400)
		} else {
			common.ErrorResp(c, err, 500)
		}
		return
	}

	if user == nil {
		user, err = common.LdapRegister(req.Username)
		if err != nil {
			common.ErrorResp(c, err, 400)
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
