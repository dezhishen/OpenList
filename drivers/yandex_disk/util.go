package yandex_disk

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/OpenListTeam/OpenList/v4/pkg/op"
	"github.com/go-resty/resty/v2"
)

// do others that not defined in Driver interface

func (y *YandexDisk) refreshToken() error {
	// 使用在线API刷新Token，无需ClientID和ClientSecret
	if y.UseOnlineAPI && len(y.APIAddress) > 0 {
		u := y.APIAddress
		var resp struct {
			RefreshToken string `json:"refresh_token"`
			AccessToken  string `json:"access_token"`
			ErrorMessage string `json:"text"`
		}
		_, err := base.RestyClient.R().
			SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Apple macOS 15_5) AppleWebKit/537.36 (KHTML, like Gecko) Safari/537.36 Chrome/138.0.0.0 Openlist/425.6.30").
			SetResult(&resp).
			SetQueryParams(map[string]string{
				"refresh_ui": y.RefreshToken,
				"server_use": "true",
				"driver_txt": "yandexui_go",
			}).
			Get(u)
		if err != nil {
			return err
		}
		if resp.RefreshToken == "" || resp.AccessToken == "" {
			if resp.ErrorMessage != "" {
				return fmt.Errorf("failed to refresh token: %s", resp.ErrorMessage)
			}
			return fmt.Errorf("empty token returned from official API , a wrong refresh token may have been used")
		}
		y.AccessToken = resp.AccessToken
		y.RefreshToken = resp.RefreshToken
		op.MustSaveDriverStorage(y)
		return nil
	}
	// 使用本地客户端的情况下检查是否为空
	if y.ClientID == "" || y.ClientSecret == "" {
		return fmt.Errorf("empty ClientID or ClientSecret")
	}
	// 走原有的刷新逻辑
	u := "https://oauth.yandex.com/token"
	var resp base.TokenResp
	var e TokenErrResp
	_, err := base.RestyClient.R().SetResult(&resp).SetError(&e).SetFormData(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": y.RefreshToken,
		"client_id":     y.ClientID,
		"client_secret": y.ClientSecret,
	}).Post(u)
	if err != nil {
		return err
	}
	if e.Error != "" {
		return fmt.Errorf("%s : %s", e.Error, e.ErrorDescription)
	}
	y.AccessToken, y.RefreshToken = resp.AccessToken, resp.RefreshToken
	op.MustSaveDriverStorage(y)
	return nil
}

func (y *YandexDisk) request(pathname string, method string, callback base.ReqCallback, resp interface{}) ([]byte, error) {
	u := "https://cloud-api.yandex.net/v1/disk/resources" + pathname
	req := base.RestyClient.R()
	req.SetHeader("Authorization", "OAuth "+y.AccessToken)
	if callback != nil {
		callback(req)
	}
	if resp != nil {
		req.SetResult(resp)
	}
	var e ErrResp
	req.SetError(&e)
	res, err := req.Execute(method, u)
	if err != nil {
		return nil, err
	}
	//log.Debug(res.String())
	if e.Error != "" {
		if e.Error == "UnauthorizedError" {
			err = y.refreshToken()
			if err != nil {
				return nil, err
			}
			return y.request(pathname, method, callback, resp)
		}
		return nil, errors.New(e.Description)
	}
	return res.Body(), nil
}

func (y *YandexDisk) getFiles(path string) ([]File, error) {
	limit := 100
	page := 1
	res := make([]File, 0)
	for {
		offset := (page - 1) * limit
		query := map[string]string{
			"path":   path,
			"limit":  strconv.Itoa(limit),
			"offset": strconv.Itoa(offset),
		}
		if y.OrderBy != "" {
			if y.OrderDirection == "desc" {
				query["sort"] = "-" + y.OrderBy
			} else {
				query["sort"] = y.OrderBy
			}
		}
		var resp FilesResp
		_, err := y.request("", http.MethodGet, func(req *resty.Request) {
			req.SetQueryParams(query)
		}, &resp)
		if err != nil {
			return nil, err
		}
		res = append(res, resp.Embedded.Items...)
		if resp.Embedded.Total <= offset+limit {
			break
		}
	}
	return res, nil
}
