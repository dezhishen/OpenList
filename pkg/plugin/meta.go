package plugin

import (
	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	go_plugin "github.com/hashicorp/go-plugin"
)

// 定义plugin的struct信息

type PluginInfo struct {
	// 名称
	Name        string   `json:"name"`
	Drivers     []string `json:"drivers"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
}

type PluginDriver []driver.Driver

// 定义plugin的接口
type MainDriversPlugin interface {
	Info() PluginInfo
}

var HandshakeConfig = go_plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "OPENLIST_PLUGIN",
	MagicCookieValue: "openlist",
}
