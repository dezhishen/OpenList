package plugin

import (
	"encoding/gob"
	"net/rpc"

	go_plugin "github.com/hashicorp/go-plugin"
)

func init() {
	registerStructs()
}

// 利用gob.Register注册struct
func registerStructs() {
	// 注册 local.Addition 结构体
	gob.Register(&struct {
		RootPath         string
		DirectorySize    bool
		Thumbnail        bool
		ThumbCacheFolder string
		ThumbConcurrency string
		VideoThumbPos    string
		ShowHidden       bool
		MkdirPerm        string
		RecycleBinPath   string
	}{})
}

type RPCMainPlugin struct {
	Impl MainDriversPlugin
}

var _ go_plugin.Plugin = (*RPCMainPlugin)(nil)

// todo 完成RPC插件的Server和Client方法
//
//	Server(*MuxBroker) (interface{}, error)
func (p *RPCMainPlugin) Server(b *go_plugin.MuxBroker) (interface{}, error) {
	return &MainDriversServer{
		Impl: p.Impl,
	}, nil
}

// Client returns an interface implementation for the plugin you're
// serving that communicates to the server end of the plugin.
// Client(*MuxBroker, *rpc.Client) (interface{}, error)
func (p *RPCMainPlugin) Client(b *go_plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCMainDriversClient{
		Impl:   p.Impl,
		client: c,
	}, nil
}

// MainDriversServer
type MainDriversServer struct {
	Impl MainDriversPlugin
}

// RPCMainDriversClient
type RPCMainDriversClient struct {
	Impl   MainDriversPlugin
	client *rpc.Client
}

// 完善RPCMulitDriversServer的Info和Drivers方法
func (s *MainDriversServer) Info(args struct{}, resp *PluginInfo) error {
	*resp = s.Impl.Info()
	return nil
}

// 完善RPCMulitDriversClient的Info和Drivers方法
func (c *RPCMainDriversClient) Info() PluginInfo {
	var resp PluginInfo
	err := c.client.Call("Plugin.Info", struct{}{}, &resp)
	if err != nil {
		panic(err)
	}
	return resp
}
