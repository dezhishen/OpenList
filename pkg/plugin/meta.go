package plugin

import (
	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	go_plugin "github.com/hashicorp/go-plugin"
)

// 定义plugin的struct信息

type PluginInfo struct {
	// 名称
	Name string `json:"name"`
	// 描述
	Description string `json:"description"`
	// 协议
	Protocol string `json:"protocol"`
}

type PluginDriver []driver.Driver

// 定义plugin的接口
type Plugin interface {
	Info() PluginInfo
	Drivers() PluginDriver
}

var HandshakeConfig = go_plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "OPENLIST_PLUGIN",
	MagicCookieValue: "openlist",
}
