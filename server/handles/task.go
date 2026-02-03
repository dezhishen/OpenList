package handles

import (
	"math"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/task"

	"github.com/OpenListTeam/OpenList/v4/internal/fs"
	"github.com/OpenListTeam/OpenList/v4/internal/offline_download/tool"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/OpenListTeam/tache"
	"github.com/gin-gonic/gin"
)

type TaskInfo struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Creator     string      `json:"creator"`
	CreatorRole int         `json:"creator_role"`
	State       tache.State `json:"state"`
	Status      string      `json:"status"`
	Progress    float64     `json:"progress"`
	StartTime   *time.Time  `json:"start_time"`
	EndTime     *time.Time  `json:"end_time"`
	TotalBytes  int64       `json:"total_bytes"`
	Error       string      `json:"error"`
}

func getTaskInfo[T task.TaskExtensionInfo](task T) TaskInfo {
	errMsg := ""
	if task.GetErr() != nil {
		errMsg = task.GetErr().Error()
	}
	progress := task.GetProgress()
	// if progress is NaN, set it to 100
	if math.IsNaN(progress) {
		progress = 100
	}
	creatorName := ""
	creatorRole := -1
	if task.GetCreator() != nil {
		creatorName = task.GetCreator().Username
		creatorRole = task.GetCreator().Role
	}
	return TaskInfo{
		ID:          task.GetID(),
		Name:        task.GetName(),
		Creator:     creatorName,
		CreatorRole: creatorRole,
		State:       task.GetState(),
		Status:      task.GetStatus(),
		Progress:    progress,
		StartTime:   task.GetStartTime(),
		EndTime:     task.GetEndTime(),
		TotalBytes:  task.GetTotalBytes(),
		Error:       errMsg,
	}
}

func getTaskInfos[T task.TaskExtensionInfo](tasks []T) []TaskInfo {
	return utils.MustSliceConvert(tasks, getTaskInfo[T])
}

func argsContains[T comparable](v T, slice ...T) bool {
	return utils.SliceContains(slice, v)
}

func getUserInfo(c *gin.Context) (bool, uint, bool) {
	if user, ok := c.Request.Context().Value(conf.UserKey).(*model.User); ok {
		return user.IsAdmin(), user.ID, true
	} else {
		return false, 0, false
	}
}

func getTargetedHandler[T task.TaskExtensionInfo](manager task.Manager[T], callback func(c *gin.Context, task T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			// if there is no bug, here is unreachable
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		t, ok := manager.GetByID(c.Query("tid"))
		if !ok {
			common.ErrorStrResp(c, "task not found", 404)
			return
		}
		if !isAdmin && uid != t.GetCreator().ID {
			// to avoid an attacker using error messages to guess valid TID, return a 404 rather than a 403
			common.ErrorStrResp(c, "task not found", 404)
			return
		}
		callback(c, t)
	}
}

func getBatchHandler[T task.TaskExtensionInfo](manager task.Manager[T], callback func(task T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		var tids []string
		if err := c.ShouldBind(&tids); err != nil {
			common.ErrorStrResp(c, "invalid request format", 400)
			return
		}
		retErrs := make(map[string]string)
		for _, tid := range tids {
			t, ok := manager.GetByID(tid)
			if !ok || (!isAdmin && uid != t.GetCreator().ID) {
				retErrs[tid] = "task not found"
				continue
			}
			callback(t)
		}
		common.SuccessResp(c, retErrs)
	}
}

// 重构后的任务路由映射
func taskRoute[T task.TaskExtensionInfo](g *gin.RouterGroup, manager task.Manager[T]) {
	g.GET("/undone", TaskUndoneHandler(manager))
	g.GET("/done", TaskDoneHandler(manager))
	g.POST("/info", TaskInfoHandler(manager))
	g.POST("/cancel", TaskCancelHandler(manager))
	g.POST("/delete", TaskDeleteHandler(manager))
	g.POST("/retry", TaskRetryHandler(manager))
	g.POST("/cancel_some", TaskBatchCancelHandler(manager))
	g.POST("/delete_some", TaskBatchDeleteHandler(manager))
	g.POST("/retry_some", TaskBatchRetryHandler(manager))
	g.POST("/clear_done", TaskClearDoneHandler(manager))
	g.POST("/clear_succeeded", TaskClearSucceededHandler(manager))
	g.POST("/retry_failed", TaskRetryFailedHandler(manager))
}

func SetupTaskRoute(g *gin.RouterGroup) {
	taskRoute(g.Group("/upload"), fs.UploadTaskManager)
	taskRoute(g.Group("/copy"), fs.CopyTaskManager)
	taskRoute(g.Group("/move"), fs.MoveTaskManager)
	taskRoute(g.Group("/offline_download"), tool.DownloadTaskManager)
	taskRoute(g.Group("/offline_download_transfer"), tool.TransferTaskManager)
	taskRoute(g.Group("/decompress"), fs.ArchiveDownloadTaskManager)
	taskRoute(g.Group("/decompress_upload"), fs.ArchiveContentUploadTaskManager)
}

// TaskUndoneHandler 获取未完成任务
//
//	@Summary		获取未完成任务
//	@Description	获取指定类型下，当前用户权限范围内的所有未完成（进行中、排队等）任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string								true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Success		200		{object}	common.jsonResult{data=[]TaskInfo}	"任务列表"
//	@Failure		401		{object}	common.jsonResult{data=string}		"未授权"
//	@Router			/api/task/{type}/undone [get]
//	@Router			/api/admin/task/{type}/undone [get]
func TaskUndoneHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		common.SuccessResp(c, getTaskInfos(manager.GetByCondition(func(task T) bool {
			return (isAdmin || uid == task.GetCreator().ID) &&
				argsContains(task.GetState(), tache.StatePending, tache.StateRunning, tache.StateCanceling,
					tache.StateErrored, tache.StateFailing, tache.StateWaitingRetry, tache.StateBeforeRetry)
		})))
	}
}

// TaskDoneHandler 获取已完成任务
//
//	@Summary		获取已完成任务
//	@Description	获取指定类型下，当前用户权限范围内的所有已完成（成功、失败、取消）任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string								true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Success		200		{object}	common.jsonResult{data=[]TaskInfo}	"任务列表"
//	@Failure		401		{object}	common.jsonResult{data=string}		"未授权"
//	@Router			/api/task/{type}/done [get]
//	@Router			/api/admin/task/{type}/done [get]
func TaskDoneHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		common.SuccessResp(c, getTaskInfos(manager.GetByCondition(func(task T) bool {
			return (isAdmin || uid == task.GetCreator().ID) &&
				argsContains(task.GetState(), tache.StateCanceled, tache.StateFailed, tache.StateSucceeded)
		})))
	}
}

// TaskInfoHandler 获取任务详情
//
//	@Summary		获取任务详情
//	@Description	根据任务ID获取特定任务的详细执行状态
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string								true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tid		query		string								true	"任务ID"
//	@Success		200		{object}	common.jsonResult{data=TaskInfo}	"任务详情"
//	@Failure		404		{object}	common.jsonResult{data=string}		"任务不存在"
//	@Router			/api/task/{type}/info [post]
//	@Router			/api/admin/task/{type}/info [post]
func TaskInfoHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getTargetedHandler(manager, func(c *gin.Context, task T) {
		common.SuccessResp(c, getTaskInfo(task))
	})
}

// TaskCancelHandler 取消任务
//
//	@Summary		取消任务
//	@Description	根据ID取消一个正在运行或排队中的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tid		query		string				true	"任务ID"
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/cancel [post]
//	@Router			/api/admin/task/{type}/cancel [post]
func TaskCancelHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getTargetedHandler(manager, func(c *gin.Context, t T) {
		manager.Cancel(t.GetID())
		common.SuccessResp(c)
	})
}

// TaskDeleteHandler 删除任务
//
//	@Summary		删除任务
//	@Description	从管理器中移除一个任务记录（无论其状态如何）
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tid		query		string				true	"任务ID"
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/delete [post]
//	@Router			/api/admin/task/{type}/delete [post]
func TaskDeleteHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getTargetedHandler(manager, func(c *gin.Context, t T) {
		manager.Remove(t.GetID())
		common.SuccessResp(c)
	})
}

// TaskRetryHandler 重试任务
//
//	@Summary		重试任务
//	@Description	重试一个执行失败的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tid		query		string				true	"任务ID"
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/retry [post]
//	@Router			/api/admin/task/{type}/retry [post]
func TaskRetryHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getTargetedHandler(manager, func(c *gin.Context, t T) {
		manager.Retry(t.GetID())
		common.SuccessResp(c)
	})
}

// TaskBatchCancelHandler 批量取消任务
//
//	@Summary		批量取消任务
//	@Description	提供一组任务ID，批量取消这些任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string										true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tids	body		[]string									true	"任务ID列表"
//	@Success		200		{object}	common.jsonResult{data=map[string]string}	"成功描述（包含每个ID的处理结果）"
//	@Router			/api/task/{type}/cancel_some [post]
//	@Router			/api/admin/task/{type}/cancel_some [post]
func TaskBatchCancelHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getBatchHandler(manager, func(task T) {
		manager.Cancel(task.GetID())
	})
}

// TaskBatchDeleteHandler 批量删除任务
//
//	@Summary		批量删除任务
//	@Description	批量移除任务记录
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string										true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tids	body		[]string									true	"任务ID列表"
//	@Success		200		{object}	common.jsonResult{data=map[string]string}	"成功描述"
//	@Router			/api/task/{type}/delete_some [post]
//	@Router			/api/admin/task/{type}/delete_some [post]
func TaskBatchDeleteHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getBatchHandler(manager, func(task T) {
		manager.Remove(task.GetID())
	})
}

// TaskBatchRetryHandler 批量重试任务
//
//	@Summary		批量重试任务
//	@Description	批量重试已失败的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string										true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Param			tids	body		[]string									true	"任务ID列表"
//	@Success		200		{object}	common.jsonResult{data=map[string]string}	"成功描述"
//	@Router			/api/task/{type}/retry_some [post]
//	@Router			/api/admin/task/{type}/retry_some [post]
func TaskBatchRetryHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return getBatchHandler(manager, func(task T) {
		manager.Retry(task.GetID())
	})
}

// TaskClearDoneHandler 清理所有已完成任务
//
//	@Summary		清理所有已完成任务
//	@Description	一键清理指定类型下所有已结束（成功/失败/取消）的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/clear_done [post]
//	@Router			/api/admin/task/{type}/clear_done [post]
func TaskClearDoneHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		manager.RemoveByCondition(func(task T) bool {
			return (isAdmin || uid == task.GetCreator().ID) &&
				argsContains(task.GetState(), tache.StateCanceled, tache.StateFailed, tache.StateSucceeded)
		})
		common.SuccessResp(c)
	}
}

// TaskClearSucceededHandler 清理已成功任务
//
//	@Summary		清理已成功任务
//	@Description	仅清理指定类型下所有状态为“成功”的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/clear_succeeded [post]
//	@Router			/api/admin/task/{type}/clear_succeeded [post]
func TaskClearSucceededHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		manager.RemoveByCondition(func(task T) bool {
			return (isAdmin || uid == task.GetCreator().ID) && task.GetState() == tache.StateSucceeded
		})
		common.SuccessResp(c)
	}
}

// TaskRetryFailedHandler 重试所有失败任务
//
//	@Summary		重试所有失败任务
//	@Description	一键重试指定类型下所有状态为“失败”的任务
//	@Tags			Task
//	@Accept			json
//	@Produce		json
//	@Param			type	path		string				true	"任务类型"	Enums(upload, copy, move, offline_download, offline_download_transfer, decompress, decompress_upload)
//	@Success		200		{object}	common.jsonResult	"描述"
//	@Router			/api/task/{type}/retry_failed [post]
//	@Router			/api/admin/task/{type}/retry_failed [post]
func TaskRetryFailedHandler[T task.TaskExtensionInfo](manager task.Manager[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, uid, ok := getUserInfo(c)
		if !ok {
			common.ErrorStrResp(c, "user invalid", 401)
			return
		}
		tasks := manager.GetByCondition(func(task T) bool {
			return (isAdmin || uid == task.GetCreator().ID) && task.GetState() == tache.StateFailed
		})
		for _, t := range tasks {
			manager.Retry(t.GetID())
		}
		common.SuccessResp(c)
	}
}
