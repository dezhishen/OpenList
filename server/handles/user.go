package handles

import (
	"strconv"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ListUsers
//
//	@Summary		List Users
//	@Description	Get a paginated list of users
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		model.PageReq	false	"Query parameters for pagination"
//	@Success		200		{object}	common.PageResp{content=[]model.User,total=int}
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/list [get]
func ListUsers(c *gin.Context) {
	var req model.PageReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Validate()
	log.Debugf("%+v", req)
	users, total, err := op.GetUsers(req.Page, req.PerPage)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, common.PageResp{
		Content: users,
		Total:   total,
	})
}

// CreateUser
//
//	@Summary		Create User
//	@Description	Create a new user
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			user	body		model.User						true	"User data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/create [post]
func CreateUser(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if req.IsAdmin() || req.IsGuest() {
		common.ErrorStrResp(c, "admin or guest user can not be created", 400, true)
		return
	}
	req.SetPassword(req.Password)
	req.Password = ""
	req.Authn = "[]"
	if err := op.CreateUser(&req); err != nil {
		common.ErrorResp(c, err, 500, true)
	} else {
		common.SuccessResp(c)
	}
}

// UpdateUser
//
//	@Summary		Update User
//	@Description	Update an existing user
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			user	body		model.User						true	"User data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Update successful"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/update [post]
func UpdateUser(c *gin.Context) {
	var req model.User
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user, err := op.GetUserById(req.ID)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	if user.Role != req.Role {
		common.ErrorStrResp(c, "role can not be changed", 400)
		return
	}
	if req.Password == "" {
		req.PwdHash = user.PwdHash
		req.Salt = user.Salt
	} else {
		req.SetPassword(req.Password)
		req.Password = ""
	}
	if req.OtpSecret == "" {
		req.OtpSecret = user.OtpSecret
	}
	if req.Disabled && req.IsAdmin() {
		common.ErrorStrResp(c, "admin user can not be disabled", 400)
		return
	}
	if err := op.UpdateUser(&req); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
	}
}

// DeleteUser
//
//	@Summary		Delete User
//	@Description	Delete a user by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	query		int								true	"User ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Deletion successful"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/delete [post]
func DeleteUser(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.DeleteUserById(uint(id)); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

// GetUser
//
//	@Summary		Get User
//	@Description	Get user by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	query		int									true	"User ID"
//	@Success		200	{object}	common.jsonResult{data=model.User}	"User object"
//	@Failure		400	{object}	common.jsonResult{data=string}		"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}		"Internal Server Error"
//	@Router			/api/admin/user/get [get]
func GetUser(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	user, err := op.GetUserById(uint(id))
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, user)
}

// Cancel2FAById
//
//	@Summary		Cancel 2FA for User
//	@Description	Cancel two-factor authentication for a user by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	query		int								true	"User ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/cancel_2fa [post]
func Cancel2FAById(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.Cancel2FAById(uint(id)); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

// DelUserCache
//
//	@Summary		Delete User Cache
//	@Description	Delete the cache for a user by username
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			username	query		string							true	"Username"
//	@Success		200			{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		500			{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/user/del_cache [post]
func DelUserCache(c *gin.Context) {
	username := c.Query("username")
	err := op.DelUserCache(username)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}
