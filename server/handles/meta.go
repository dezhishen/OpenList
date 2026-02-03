package handles

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// ListMetas godoc
//
//	@Summary		List Metas
//	@Description	Get a paginated list of metas
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		model.PageReq	false	"Query parameters for pagination"
//	@Success		200		{object}	common.PageResp{content=[]model.Meta,total=int}
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/meta/list [get]
func ListMetas(c *gin.Context) {
	var req model.PageReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Validate()
	log.Debugf("%+v", req)
	metas, total, err := op.GetMetas(req.Page, req.PerPage)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, common.PageResp{
		Content: metas,
		Total:   total,
	})
}

// CreateMeta
//
//	@Summary		Create Meta
//	@Description	Create a new meta
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			meta	body		model.Meta						true	"Meta data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/meta/create [post]
func CreateMeta(c *gin.Context) {
	var req model.Meta
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	r, err := validHide(req.Hide)
	if err != nil {
		common.ErrorStrResp(c, fmt.Sprintf("%s is illegal: %s", r, err.Error()), 400)
		return
	}
	if err := op.CreateMeta(&req); err != nil {
		common.ErrorResp(c, err, 500, true)
	} else {
		common.SuccessResp(c)
	}
}

// UpdateMeta
//
//	@Summary		Update Meta
//	@Description	Update an existing meta
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			meta	body		model.Meta						true	"Meta data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/meta/update [post]
func UpdateMeta(c *gin.Context) {
	var req model.Meta
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	r, err := validHide(req.Hide)
	if err != nil {
		common.ErrorStrResp(c, fmt.Sprintf("%s is illegal: %s", r, err.Error()), 400)
		return
	}
	if err := op.UpdateMeta(&req); err != nil {
		common.ErrorResp(c, err, 500, true)
	} else {
		common.SuccessResp(c)
	}
}

func validHide(hide string) (string, error) {
	rs := strings.Split(hide, "\n")
	for _, r := range rs {
		_, err := regexp2.Compile(r, regexp2.None)
		if err != nil {
			return r, err
		}
	}
	return "", nil
}

// DeleteMeta
//
//	@Summary		Delete Meta
//	@Description	Delete a meta by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	query		int								true	"Meta ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/meta/delete [post]
func DeleteMeta(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.DeleteMetaById(uint(id)); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// GetMeta
//
//	@Summary		Get Meta
//	@Description	Get meta by ID
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			id	query		int	true	"Meta ID"
//	@Success		200	{object}	common.jsonResult{data=model.Meta}
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/meta/get [get]
func GetMeta(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	meta, err := op.GetMetaById(uint(id))
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, meta)
}
