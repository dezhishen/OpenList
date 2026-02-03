package handles

import (
	"fmt"

	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

// ListDriverInfo list all driver info
//
//	@Summary		List all driver info
//	@Description	Get information about all drivers
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=map[string]driver.Info}	"Map of driver info"
//	@Failure		500	{object}	common.jsonResult{data=string}					"Internal Server Error"
//	@Router			/api/admin/driver/list [get]
func ListDriverInfo(c *gin.Context) {
	common.SuccessResp(c, op.GetDriverInfoMap())
}

// ListDriverNames list all driver names
//
//	@Summary		List all driver names
//	@Description	Get a list of all driver names
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=[]string}	"List of driver names"
//	@Failure		500	{object}	common.jsonResult{data=string}		"Internal Server Error"
//	@Router			/api/admin/driver/names [get]
func ListDriverNames(c *gin.Context) {
	common.SuccessResp(c, op.GetDriverNames())
}

// GetDriverInfo get driver info by name
//
//	@Summary		Get Driver Info
//	@Description	Get information about a specific driver by name
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			driver	query		string								true	"Driver Name"
//	@Success		200		{object}	common.jsonResult{data=driver.Info}	"Driver info key-value map"
//	@Failure		404		{object}	common.jsonResult{data=string}		"Driver Not Found"
//	@Failure		500		{object}	common.jsonResult{data=string}		"Internal Server Error"
//	@Router			/api/admin/driver/info [get]
func GetDriverInfo(c *gin.Context) {
	driverName := c.Query("driver")
	infoMap := op.GetDriverInfoMap()
	items, ok := infoMap[driverName]
	if !ok {
		common.ErrorStrResp(c, fmt.Sprintf("driver [%s] not found", driverName), 404)
		return
	}
	common.SuccessResp(c, items)
}
