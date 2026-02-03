package handles

import (
	"strconv"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

type SSHKeyAddReq struct {
	Title string `json:"title" binding:"required"`
	Key   string `json:"key" binding:"required"`
}

// AddMyPublicKey add a new SSH public key for current user
//
//	@Summary		Add My Public Key
//	@Description	Add a new SSH public key associated with the current user
//	@Tags			User
//	@Accept			json
//	@Produce		json
//	@Param			sshkey	body		SSHKeyAddReq					true	"SSH Public Key Data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		401		{object}	common.jsonResult{data=string}	"Unauthorized"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/me/sshkey/add [post]
//	@Security		Authorization
func AddMyPublicKey(c *gin.Context) {
	userObj, ok := c.Request.Context().Value(conf.UserKey).(*model.User)
	if !ok || userObj.IsGuest() {
		common.ErrorStrResp(c, "user invalid", 401)
		return
	}
	var req SSHKeyAddReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorStrResp(c, "request invalid", 400)
		return
	}
	if req.Title == "" {
		common.ErrorStrResp(c, "request invalid", 400)
		return
	}
	key := &model.SSHPublicKey{
		Title:  req.Title,
		KeyStr: strings.TrimSpace(req.Key),
		UserId: userObj.ID,
	}
	err, parsed := op.CreateSSHPublicKey(key)
	if !parsed {
		common.ErrorStrResp(c, "provided key invalid", 400)
		return
	} else if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// ListMyPublicKey list current user's public keys
//
//	@Summary		List My Public Keys
//	@Description	List all SSH public keys associated with the current user
//	@Tags			User
//	@Produce		json
//	@Success		200	{object}	common.PageResp{content=[]model.SSHPublicKey,total=int}	"List of user's SSH public keys"
//	@Failure		401	{object}	common.jsonResult{data=string}							"Unauthorized"
//	@Router			/api/me/sshkey/list [get]
//	@Security		Authorization
func ListMyPublicKey(c *gin.Context) {
	userObj, ok := c.Request.Context().Value(conf.UserKey).(*model.User)
	if !ok || userObj.IsGuest() {
		common.ErrorStrResp(c, "user invalid", 401)
		return
	}
	list(c, userObj)
}

// DeleteMyPublicKey delete current user's public key
//
//	@Summary		Delete My Public Key
//	@Description	Delete an SSH public key associated with the current user
//	@Tags			User
//	@Produce		json
//	@Param			id	query		int								true	"ID of the SSH public key to delete"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		401	{object}	common.jsonResult{data=string}	"Unauthorized"
//	@Failure		404	{object}	common.jsonResult{data=string}	"Not Found"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/me/sshkey/delete [delete]
//	@Security		Authorization
func DeleteMyPublicKey(c *gin.Context) {
	userObj, ok := c.Request.Context().Value(conf.UserKey).(*model.User)
	if !ok || userObj.IsGuest() {
		common.ErrorStrResp(c, "user invalid", 401)
		return
	}
	keyId, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		common.ErrorStrResp(c, "id format invalid", 400)
		return
	}
	key, err := op.GetSSHPublicKeyByIdAndUserId(uint(keyId), userObj.ID)
	if err != nil {
		common.ErrorStrResp(c, "failed to get public key", 404)
		return
	}
	err = op.DeleteSSHPublicKeyById(key.ID)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// ListPublicKeys list a user's public keys by user id
//
//	@Summary		List Public Keys
//	@Description	List all SSH public keys associated with a specified user ID
//	@Tags			Admin
//	@Produce		json
//	@Param			uid	query		int														true	"User ID"
//	@Success		200	{object}	common.PageResp{content=[]model.SSHPublicKey,total=int}	"List of user's SSH public keys"
//	@Failure		400	{object}	common.jsonResult{data=string}							"Bad Request"
//	@Failure		404	{object}	common.jsonResult{data=string}							"User Not Found"
//	@Failure		500	{object}	common.jsonResult{data=string}							"Internal Server Error"
//	@Router			/api/admin/sshkey/list [get]
func ListPublicKeys(c *gin.Context) {
	userId, err := strconv.Atoi(c.Query("uid"))
	if err != nil {
		common.ErrorStrResp(c, "user id format invalid", 400)
		return
	}
	userObj, err := op.GetUserById(uint(userId))
	if err != nil {
		common.ErrorStrResp(c, "user invalid", 404)
		return
	}
	list(c, userObj)
}

// DeletePublicKey delete a user's public key by key id
//
//	@Summary		Delete Public Key
//	@Description	Delete an SSH public key by its ID
//	@Tags			Admin
//	@Produce		json
//	@Param			id	query		int								true	"ID of the SSH public key to delete"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/sshkey/delete [post]
func DeletePublicKey(c *gin.Context) {
	keyId, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		common.ErrorStrResp(c, "id format invalid", 400)
		return
	}
	err = op.DeleteSSHPublicKeyById(uint(keyId))
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

func list(c *gin.Context, userObj *model.User) {
	var req model.PageReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Validate()
	keys, total, err := op.GetSSHPublicKeyByUserId(userObj.ID, req.Page, req.PerPage)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, common.PageResp{
		Content: keys,
		Total:   total,
	})
}
