package yandex_disk

import (
	"context"
	"net/http"
	"path"
	"strconv"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/go-resty/resty/v2"
)

type YandexDisk struct {
	model.Storage
	Addition
	AccessToken string
}

func (y *YandexDisk) Config() driver.Config {
	return config
}

func (y *YandexDisk) GetAddition() driver.Additional {
	additional, err := driver.NewSimpleAdditional(y.RootPath, y.Addition)
	if err != nil {
		panic(err)
	}
	return additional
}

func (y *YandexDisk) SetAddition(additional driver.Additional) {
	if additional != nil {
		y.Addition = Addition{}
		err := additional.UnmarshalData(&y.Addition)
		if err != nil {
			panic(err)
		}
	}
}

func (y *YandexDisk) Init(ctx context.Context) error {
	return y.refreshToken()
}

func (y *YandexDisk) Drop(ctx context.Context) error {
	return nil
}

func (y *YandexDisk) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	files, err := y.getFiles(dir.GetPath())
	if err != nil {
		return nil, err
	}
	return utils.SliceConvert(files, func(src File) (model.Obj, error) {
		return fileToObj(src), nil
	})
}

func (y *YandexDisk) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var resp DownResp
	_, err := y.request("/download", http.MethodGet, func(req *resty.Request) {
		req.SetQueryParam("path", file.GetPath())
	}, &resp)
	if err != nil {
		return nil, err
	}
	link := model.Link{
		URL: resp.Href,
	}
	return &link, nil
}

func (y *YandexDisk) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	_, err := y.request("", http.MethodPut, func(req *resty.Request) {
		req.SetQueryParam("path", path.Join(parentDir.GetPath(), dirName))
	}, nil)
	return err
}

func (y *YandexDisk) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	_, err := y.request("/move", http.MethodPost, func(req *resty.Request) {
		req.SetQueryParams(map[string]string{
			"from":      srcObj.GetPath(),
			"path":      path.Join(dstDir.GetPath(), srcObj.GetName()),
			"overwrite": "true",
		})
	}, nil)
	return err
}

func (y *YandexDisk) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	_, err := y.request("/move", http.MethodPost, func(req *resty.Request) {
		req.SetQueryParams(map[string]string{
			"from":      srcObj.GetPath(),
			"path":      path.Join(path.Dir(srcObj.GetPath()), newName),
			"overwrite": "true",
		})
	}, nil)
	return err
}

func (y *YandexDisk) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	_, err := y.request("/copy", http.MethodPost, func(req *resty.Request) {
		req.SetQueryParams(map[string]string{
			"from":      srcObj.GetPath(),
			"path":      path.Join(dstDir.GetPath(), srcObj.GetName()),
			"overwrite": "true",
		})
	}, nil)
	return err
}

func (y *YandexDisk) Remove(ctx context.Context, obj model.Obj) error {
	_, err := y.request("", http.MethodDelete, func(req *resty.Request) {
		req.SetQueryParam("path", obj.GetPath())
	}, nil)
	return err
}

func (y *YandexDisk) Put(ctx context.Context, dstDir model.Obj, s model.FileStreamer, up driver.UpdateProgress) error {
	var resp UploadResp
	_, err := y.request("/upload", http.MethodGet, func(req *resty.Request) {
		req.SetQueryParams(map[string]string{
			"path":      path.Join(dstDir.GetPath(), s.GetName()),
			"overwrite": "true",
		})
	}, &resp)
	if err != nil {
		return err
	}
	reader := driver.NewLimitedUploadStream(ctx, &driver.ReaderUpdatingProgress{
		Reader:         s,
		UpdateProgress: up,
	})
	req, err := http.NewRequestWithContext(ctx, resp.Method, resp.Href, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Length", strconv.FormatInt(s.GetSize(), 10))
	req.Header.Set("Content-Type", "application/octet-stream")
	res, err := base.HttpClient.Do(req)
	if err != nil {
		return err
	}
	_ = res.Body.Close()
	return err
}

var _ driver.Driver = (*YandexDisk)(nil)
