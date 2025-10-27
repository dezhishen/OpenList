package plugin

import (
	"os"
	"os/exec"

	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/op"
	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
)

func Init(path string) {
	// Here we set up our client. We're a host! Start by launching the plugin process.
	// regsiterPlugins()
	plugins, err := LoadPlugins(path)
	if err != nil {
		logrus.Panic("load plugins error:", err)
	}
	for _, p := range plugins {
		for _, d := range p.Drivers() {
			logrus.Infof("register plugin driver: %s", d.Config().Name)
			op.RegisterDriver(func() driver.Driver {
				return d
			})
		}
	}
}

func LoadPlugins(path string) ([]Plugin, error) {
	isDirectory, err := isDir(path)
	if err != nil {
		return nil, err
	}
	var plugins []Plugin
	if !isDirectory {
		p, _, err := loadPlugin(path)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, p)
		return plugins, nil
	}
	// loop directory
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		p, _, err := loadPlugin(path + string(os.PathSeparator) + entry.Name())
		if err != nil {
			logrus.Error("load plugin error:", err)
			continue
		}
		plugins = append(plugins, p)
	}
	return plugins, nil
}

func loadPlugin(path string) (Plugin, *plugin.Client, error) {
	// We're a host! Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]plugin.Plugin{
			"main": &PluginNetRpcPlugin{},
		},
		Cmd: exec.Command(path),
		AllowedProtocols: []plugin.Protocol{
			plugin.ProtocolNetRPC,
		},
	})
	// defer client.Kill()
	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		logrus.Error(err)
		return nil, nil, err
	}
	// Request the plugin
	raw, err := rpcClient.Dispense("main")
	if err != nil {
		return nil, nil, err
	}
	d, ok := raw.(Plugin)
	if !ok {
		logrus.Error("type assert error")
		return nil, nil, err
	}
	return d, client, nil
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
