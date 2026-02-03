package handles

import (
	"sort"
	"strconv"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/bootstrap/data"
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/internal/sign"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils/random"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/OpenListTeam/OpenList/v4/server/static"
	"github.com/gin-gonic/gin"
)

// ResetToken reset the system token
//
//	@Summary		Reset System Token
//	@Description	Reset the system token and return the new token
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Newly generated token"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/setting/reset_token [post]
func ResetToken(c *gin.Context) {
	token := random.Token()
	item := model.SettingItem{Key: "token", Value: token, Type: conf.TypeString, Group: model.SINGLE, Flag: model.PRIVATE}
	if err := op.SaveSettingItem(&item); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	sign.Instance()
	common.SuccessResp(c, token)
}

// GetSetting get setting by key or keys
//
//	@Summary		Get Setting
//	@Description	Get setting by key or keys
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			key		query		string										false	"Setting key"
//	@Param			keys	query		string										false	"Comma-separated list of setting keys"
//	@Success		200		{object}	common.jsonResult{data=[]model.SettingItem}	"Setting list of setting items"
//	@Success		200		{object}	common.jsonResult{data=model.SettingItem}	"Single setting item"
//	@Failure		400		{object}	common.jsonResult{data=string}				"Bad Request"
//	@Router			/api/admin/setting/get [get]
func GetSetting(c *gin.Context) {
	key := c.Query("key")
	keys := c.Query("keys")
	if key != "" {
		item, err := op.GetSettingItemByKey(key)
		if err != nil {
			common.ErrorResp(c, err, 400)
			return
		}
		common.SuccessResp(c, item)
	} else {
		items, err := op.GetSettingItemInKeys(strings.Split(keys, ","))
		if err != nil {
			common.ErrorResp(c, err, 400)
			return
		}
		common.SuccessResp(c, items)
	}
}

// SaveSettings save settings
//
//	@Summary		Save Settings
//	@Description	Save multiple settings
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			settings	body		[]model.SettingItem				true	"List of setting items to save"
//	@Success		200			{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400			{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500			{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/setting/save [post]
func SaveSettings(c *gin.Context) {
	var req []model.SettingItem
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.SaveSettingItems(req); err != nil {
		common.ErrorResp(c, err, 500)
	} else {
		common.SuccessResp(c)
		static.UpdateIndex()
	}
}

// ListSettings list settings, optionally filtered by group or groups
//
//	@Summary		List Settings
//	@Description	List settings, optionally filtered by group or groups
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			group	query		string										false	"Group number to filter settings"
//	@Param			groups	query		string										false	"Comma-separated list of group numbers to filter settings"
//	@Success		200		{object}	common.jsonResult{data=[]model.SettingItem}	"List of setting items"
//	@Failure		400		{object}	common.jsonResult{data=string}				"Bad Request"
//	@Router			/api/admin/setting/list [get]
func ListSettings(c *gin.Context) {
	groupStr := c.Query("group")
	groupsStr := c.Query("groups")
	var settings []model.SettingItem
	var err error
	if groupsStr == "" && groupStr == "" {
		settings, err = op.GetSettingItems()
	} else {
		var groupStrings []string
		if groupsStr != "" {
			groupStrings = strings.Split(groupsStr, ",")
		} else {
			groupStrings = append(groupStrings, groupStr)
		}
		var groups []int
		for _, str := range groupStrings {
			group, err := strconv.Atoi(str)
			if err != nil {
				common.ErrorResp(c, err, 400)
				return
			}
			groups = append(groups, group)
		}
		settings, err = op.GetSettingItemsInGroups(groups)
	}
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	common.SuccessResp(c, settings)
}

// DefaultSettings get default settings, optionally filtered by group or groups
//
//	@Summary		Get Default Settings
//	@Description	Get default settings, optionally filtered by group or groups
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			group	query		string										false	"Group number to filter settings"
//	@Param			groups	query		string										false	"Comma-separated list of group numbers to filter settings"
//	@Success		200		{object}	common.jsonResult{data=[]model.SettingItem}	"List of default setting items"
//	@Failure		400		{object}	common.jsonResult{data=string}				"Bad Request"
//	@Router			/api/admin/setting/defaults [get]
func DefaultSettings(c *gin.Context) {
	groupStr := c.Query("group")
	groupsStr := c.Query("groups")
	settings := data.InitialSettings()
	if groupsStr == "" && groupStr == "" {
		for i := range settings {
			(&settings[i]).Index = uint(i)
		}
		common.SuccessResp(c, settings)
	} else {
		var groupStrings []string
		if groupsStr != "" {
			groupStrings = strings.Split(groupsStr, ",")
		} else {
			groupStrings = append(groupStrings, groupStr)
		}
		var groups []int
		for _, str := range groupStrings {
			group, err := strconv.Atoi(str)
			if err != nil {
				common.ErrorResp(c, err, 400)
				return
			}
			groups = append(groups, group)
		}
		sort.Ints(groups)
		var resultItems []model.SettingItem
		for _, group := range groups {
			for i := range settings {
				item := settings[i]
				if group == item.Group {
					item.Index = uint(i)
					resultItems = append(resultItems, item)
				}
			}
		}
		common.SuccessResp(c, resultItems)
	}
}

// DeleteSetting delete a setting by key
//
//	@Summary		Delete Setting
//	@Description	Delete a setting by its key
//	@Tags			Admin
//	@Produce		json
//	@Param			key	query		string							true	"Key of the setting to delete"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/setting/delete [post]
func DeleteSetting(c *gin.Context) {
	key := c.Query("key")
	if err := op.DeleteSettingItemByKey(key); err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c)
}

// PublicSettings godoc
//
//	@Summary		Get Public Settings
//	@Description	Get all public settings as a key-value map
//	@Tags			Public
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=map[string]string}	"Map of public settings"
//	@Failure		500	{object}	common.jsonResult{data=string}				"Internal Server Error"
//	@Router			/api/public/settings [get]
func PublicSettings(c *gin.Context) {
	common.SuccessResp(c, op.GetPublicSettingsMap())
}
