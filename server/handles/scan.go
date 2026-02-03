package handles

import (
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

type ManualScanReq struct {
	Path  string  `json:"path"`
	Limit float64 `json:"limit"`
}

// StartManualScan starts a manual scan for the specified path with an optional limit
//
//	@Summary		Start Manual Scan
//	@Description	Start a manual scan for the specified path with an optional limit
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			scan	body		ManualScanReq					true	"Manual Scan Data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Router			/api/admin/scan/start [post]
func StartManualScan(c *gin.Context) {
	var req ManualScanReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.BeginManualScan(req.Path, req.Limit); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	common.SuccessResp(c)
}

// StopManualScan stops the ongoing manual scan
//
//	@Summary		Stop Manual Scan
//	@Description	Stop the ongoing manual scan
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Router			/api/admin/scan/stop [post]
func StopManualScan(c *gin.Context) {
	if !op.ManualScanRunning() {
		common.ErrorStrResp(c, "manual scan is not running", 400)
		return
	}
	op.StopManualScan()
	common.SuccessResp(c)
}

type ManualScanResp struct {
	ObjCount uint64 `json:"obj_count"`
	IsDone   bool   `json:"is_done"`
}

// GetManualScanProgress gets the progress of the ongoing manual scan
//
//	@Summary		Get Manual Scan Progress
//	@Description	Get the progress of the ongoing manual scan
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=ManualScanResp}	"Manual scan progress data"
//	@Router			/api/admin/scan/progress [get]
func GetManualScanProgress(c *gin.Context) {
	ret := ManualScanResp{
		ObjCount: op.ScannedCount.Load(),
		IsDone:   !op.ManualScanRunning(),
	}
	common.SuccessResp(c, ret)
}
