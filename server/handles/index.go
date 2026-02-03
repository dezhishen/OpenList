package handles

import (
	"context"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/search"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type UpdateIndexReq struct {
	Paths    []string `json:"paths"`
	MaxDepth int      `json:"max_depth"`
	//IgnorePaths []string `json:"ignore_paths"`
}

// BuildIndex builds the index from scratch
//
//	@Summary		Build Index
//	@Description	Build the search index from scratch
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/index/build [post]
func BuildIndex(c *gin.Context) {
	if search.Running() {
		common.ErrorStrResp(c, "index is running", 400)
		return
	}
	go func() {
		ctx := context.Background()
		err := search.Clear(ctx)
		if err != nil {
			log.Errorf("clear index error: %+v", err)
			return
		}
		err = search.BuildIndex(context.Background(), []string{"/"},
			conf.SlicesMap[conf.IgnorePaths], setting.GetInt(conf.MaxIndexDepth, 20), true)
		if err != nil {
			log.Errorf("build index error: %+v", err)
		}
	}()
	common.SuccessResp(c)
}

// UpdateIndex updates the index for specified paths
//
//	@Summary		Update Index
//	@Description	Update the search index for specified paths
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			index	body		UpdateIndexReq					true	"Index Update Data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/index/update [post]
func UpdateIndex(c *gin.Context) {
	var req UpdateIndexReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if search.Running() {
		common.ErrorStrResp(c, "index is running", 400)
		return
	}
	if !search.Config(c).AutoUpdate {
		common.ErrorStrResp(c, "update is not supported for current index", 400)
		return
	}
	go func() {
		ctx := context.Background()
		for _, path := range req.Paths {
			err := search.Del(ctx, path)
			if err != nil {
				log.Errorf("delete index on %s error: %+v", path, err)
				return
			}
		}
		err := search.BuildIndex(context.Background(), req.Paths,
			conf.SlicesMap[conf.IgnorePaths], req.MaxDepth, false)
		if err != nil {
			log.Errorf("update index error: %+v", err)
		}
	}()
	common.SuccessResp(c)
}

// StopIndex stops the ongoing indexing process
//
//	@Summary		Stop Index
//	@Description	Stop the ongoing indexing process
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Router			/api/admin/index/stop [post]
func StopIndex(c *gin.Context) {
	quit := search.Quit.Load()
	if quit == nil {
		common.ErrorStrResp(c, "index is not running", 400)
		return
	}
	select {
	case *quit <- struct{}{}:
	default:
	}
	common.SuccessResp(c)
}

// ClearIndex clears the entire search index
//
//	@Summary		Clear Index
//	@Description	Clear the entire search index
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/index/clear [post]
func ClearIndex(c *gin.Context) {
	if search.Running() {
		common.ErrorStrResp(c, "index is running", 400)
		return
	}
	search.Clear(c)
	search.WriteProgress(&model.IndexProgress{
		ObjCount:     0,
		IsDone:       true,
		LastDoneTime: nil,
		Error:        "",
	})
	common.SuccessResp(c)
}

// GetProgress gets the current indexing progress
//
//	@Summary		Get Index Progress
//	@Description	Get the current indexing progress
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=model.IndexProgress}	"Indexing progress data"
//	@Failure		500	{object}	common.jsonResult{data=string}				"Internal Server Error"
//	@Router			/api/admin/index/progress [get]
func GetProgress(c *gin.Context) {
	progress, err := search.Progress()
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, progress)
}
