package bootstrap

import (
	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/pkg/plugin"
)

func InitPlugin() error {
	path := conf.Conf.Plugin.Path
	if path == "" {
		path = "./plugins"
	}
	return plugin.Init(path)
}
