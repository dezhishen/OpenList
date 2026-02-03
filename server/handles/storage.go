package handles

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/internal/setting"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type StorageResp struct {
	model.Storage
	MountDetails *model.StorageDetails `json:"mount_details,omitempty"`
}

type detailWithIndex struct {
	idx int
	val *model.StorageDetails
}

func makeStorageResp(ctx *gin.Context, storages []model.Storage) []*StorageResp {
	ret := make([]*StorageResp, len(storages))
	detailsChan := make(chan detailWithIndex, len(storages))
	workerCount := 0
	for i, s := range storages {
		ret[i] = &StorageResp{
			Storage:      s,
			MountDetails: nil,
		}
		if setting.GetBool(conf.HideStorageDetailsInManagePage) {
			continue
		}
		d, err := op.GetStorageByMountPath(s.MountPath)
		if err != nil {
			continue
		}
		_, ok := d.(driver.WithDetails)
		if !ok {
			continue
		}
		workerCount++
		go func(dri driver.Driver, idx int) {
			details, e := op.GetStorageDetails(ctx, dri)
			if e != nil {
				if !errors.Is(e, errs.NotImplement) && !errors.Is(e, errs.StorageNotInit) {
					log.Errorf("failed get %s details: %+v", dri.GetStorage().MountPath, e)
				}
			}
			detailsChan <- detailWithIndex{idx: idx, val: details}
		}(d, i)
	}
	for workerCount > 0 {
		select {
		case r := <-detailsChan:
			ret[r.idx].MountDetails = r.val
			workerCount--
		case <-time.After(time.Second * 3):
			workerCount = 0
		}
	}
	return ret
}

// ListStorages
//
//	@Summary		List all storages
//	@Description	Get a paginated list of storages
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			page	query		model.PageReq	false	"Query parameters for pagination"
//	@Success		200		{object}	common.PageResp{content=[]StorageResp,total=int}
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/list [get]
func ListStorages(c *gin.Context) {
	var req model.PageReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	req.Validate()
	log.Debugf("%+v", req)
	storages, total, err := db.GetStorages(req.Page, req.PerPage)
	if err != nil {
		common.ErrorResp(c, err, 500)
		return
	}
	common.SuccessResp(c, common.PageResp{
		Content: makeStorageResp(c, storages),
		Total:   total,
	})
}

// CreateStorage create a new storage
//
//	@Summary		Create Storage
//	@Description	Create a new storage
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			storage	body		model.Storage							true	"Storage data"
//	@Success		200		{object}	common.jsonResult{data=object{id=int}}	"Success message with new storage ID"
//	@Failure		400		{object}	common.jsonResult{data=string}			"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}			"Internal Server Error"
//	@Router			/api/admin/storage/create [post]
func CreateStorage(c *gin.Context) {
	var req model.Storage
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if id, err := op.CreateStorage(c.Request.Context(), req); err != nil {
		common.ErrorWithDataResp(c, err, 500, gin.H{
			"id": id,
		}, true)
	} else {
		common.SuccessResp(c, gin.H{
			"id": id,
		})
	}
}

// UpdateStorage update an existing storage
//
//	@Summary		Update Storage
//	@Description	Update an existing storage
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Param			storage	body		model.Storage					true	"Storage data"
//	@Success		200		{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400		{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500		{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/update [post]
func UpdateStorage(c *gin.Context) {
	var req model.Storage
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.UpdateStorage(c.Request.Context(), req); err != nil {
		common.ErrorResp(c, err, 500, true)
	} else {
		common.SuccessResp(c)
	}
}

// DeleteStorage delete a storage by id
//
//	@Summary		Delete Storage
//	@Description	Delete a storage by ID
//	@Tags			Admin
//	@Produce		json
//	@Param			id	query		int								true	"Storage ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/delete [post]
func DeleteStorage(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.DeleteStorageById(c.Request.Context(), uint(id)); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// DisableStorage disable a storage by id
//
//	@Summary		Disable Storage
//	@Description	Disable a storage by ID
//	@Tags			Admin
//	@Produce		json
//	@Param			id	query		int								true	"Storage ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/disable [post]
func DisableStorage(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.DisableStorage(c.Request.Context(), uint(id)); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// EnableStorage enable a storage by id
//
//	@Summary		Enable Storage
//	@Description	Enable a storage by ID
//	@Tags			Admin
//	@Produce		json
//	@Param			id	query		int								true	"Storage ID"
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		400	{object}	common.jsonResult{data=string}	"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/enable [post]
func EnableStorage(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	if err := op.EnableStorage(c.Request.Context(), uint(id)); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

// GetStorage get storage by id
//
//	@Summary		Get Storage
//	@Description	Get storage details by ID
//	@Tags			Admin
//	@Produce		json
//	@Param			id	query		int									true	"Storage ID"
//	@Success		200	{object}	common.jsonResult{data=StorageResp}	"Storage details"
//	@Failure		400	{object}	common.jsonResult{data=string}		"Bad Request"
//	@Failure		500	{object}	common.jsonResult{data=string}		"Internal Server Error"
//	@Router			/api/admin/storage/get [get]
func GetStorage(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	storage, err := db.GetStorageById(uint(id))
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, storage)
}

// LoadAllStorages load all enabled storages
//
//	@Summary		Reload all storages
//	@Description	Trigger loading of all enabled storages
//	@Tags			Admin
//	@Produce		json
//	@Success		200	{object}	common.jsonResult{data=string}	"Success message"
//	@Failure		500	{object}	common.jsonResult{data=string}	"Internal Server Error"
//	@Router			/api/admin/storage/load_all [post]
func LoadAllStorages(c *gin.Context) {
	storages, err := db.GetEnabledStorages()
	if err != nil {
		log.Errorf("failed get enabled storages: %+v", err)
		common.ErrorResp(c, err, 500, true)
		return
	}
	conf.ResetStoragesLoadSignal()
	go func(storages []model.Storage) {
		for _, storage := range storages {
			storageDriver, err := op.GetStorageByMountPath(storage.MountPath)
			if err != nil {
				log.Errorf("failed get storage driver: %+v", err)
				continue
			}
			// drop the storage in the driver
			if err := storageDriver.Drop(context.Background()); err != nil {
				log.Errorf("failed drop storage: %+v", err)
				continue
			}
			if err := op.LoadStorage(context.Background(), storage); err != nil {
				log.Errorf("failed get enabled storages: %+v", err)
				continue
			}
			log.Infof("success load storage: [%s], driver: [%s]",
				storage.MountPath, storage.Driver)
		}
		conf.SendStoragesLoadedSignal()
	}(storages)
	common.SuccessResp(c)
}
