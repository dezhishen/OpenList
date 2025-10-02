package openlist_share

import (
	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/op"
)

type Addition struct {
	driver.RootPath
	Address           string `json:"url" required:"true"`
	ShareId           string `json:"sid" required:"true"`
	Pwd               string `json:"pwd"`
	ForwardArchiveReq bool   `json:"forward_archive_requests" default:"true"`
}

var config = driver.Config{
	Name:        "OpenListShare",
	LocalSort:   true,
	NoUpload:    true,
	DefaultRoot: "/",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &OpenListShare{}
	})
}
