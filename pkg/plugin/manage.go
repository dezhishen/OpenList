package plugin

import (
	"os"
	"os/exec"

	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/op"
	go_plugin "github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
)

func Close() {
	go_plugin.CleanupClients()
}

func Init(path string) error {
	// Here we set up our client. We're a host! Start by launching the plugin process.
	// regsiterPlugins()
	plugins, err := LoadPlugins(path)
	if err != nil {
		return err
	}
	for _, d := range plugins {
		op.RegisterDriver(func() driver.Driver {
			return d
		})
	}
	return nil
}

func LoadPlugins(path string) ([]driver.Driver, error) {
	isDirectory, err := isDir(path)
	if err != nil {
		return nil, err
	}
	var plugins []driver.Driver
	if !isDirectory {
		info, ds, err := loadPlugin(path)
		logrus.Infof("loaded plugin: %s - %s", info.Name, info.Description)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, ds...)
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
		info, ds, err := loadPlugin(path + string(os.PathSeparator) + entry.Name())
		if err != nil {
			logrus.Error("load plugin error:", err)
			continue
		}
		logrus.Infof("loaded plugin: %s - %s", info.Name, info.Description)
		if len(ds) > 0 {
			plugins = append(plugins, ds...)
		}
	}
	return plugins, nil
}

func loadPlugin(path string) (*PluginInfo, []driver.Driver, error) {
	// We're a host! Start by launching the plugin process.
	client := go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins: map[string]go_plugin.Plugin{
			"main": &RPCMainPlugin{},
		},
		Cmd: exec.Command(path),
		AllowedProtocols: []go_plugin.Protocol{
			go_plugin.ProtocolNetRPC,
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
	tmpMain, ok := raw.(MainDriversPlugin)
	if !ok {
		logrus.Error("type assert error")
		return nil, nil, err
	}
	drivers := tmpMain.Info().Drivers
	// 复制一个main
	var mainCopy = &PluginInfo{
		Name:        tmpMain.Info().Name,
		Description: tmpMain.Info().Description,
		Drivers:     []string{},
	}
	rpcClient.Close()
	client.Kill()
	if len(drivers) == 0 {
		logrus.Warnf("plugin %s has no drivers", tmpMain.Info().Name)
		return mainCopy, nil, nil
	}
	// 重新连接rpcClient以获取drivers
	plugins := map[string]go_plugin.Plugin{}
	for _, drv := range drivers {
		plugins[drv] = &RPCDriverPlugin{}
	}
	client = go_plugin.NewClient(&go_plugin.ClientConfig{
		HandshakeConfig: HandshakeConfig,
		Plugins:         plugins,
		Cmd:             exec.Command(path),
		AllowedProtocols: []go_plugin.Protocol{
			go_plugin.ProtocolNetRPC,
		},
	})
	rpcClient, err = client.Client()
	if err != nil {
		logrus.Error(err)
		return nil, nil, err
	}
	var ds []driver.Driver
	for _, drv := range drivers {
		d, err := rpcClient.Dispense(drv)
		if err != nil {
			logrus.Error("dispense driver error:", err)
			continue
		}
		drv, ok := d.(driver.Driver)
		if !ok {
			logrus.Error("type assert driver error")
			continue
		}
		ds = append(ds, drv)
	}
	return mainCopy, ds, nil
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
