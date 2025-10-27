package plugin

import (
	"context"
	"net/rpc"

	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/model"
	go_plugin "github.com/hashicorp/go-plugin"
)

// PluginNetRpcPlugin is the go-plugin glue for the top-level Plugin interface.
// It allows the host to Dispense("main") and get a Plugin that internally
// talks over net/rpc to the plugin process.
type PluginNetRpcPlugin struct {
	// Impl will be non-nil in the plugin process (server side).
	Impl Plugin
}

// Server returns the RPC server for the plugin process.
func (p *PluginNetRpcPlugin) Server(broker *go_plugin.MuxBroker) (interface{}, error) {
	return &PluginRPCServer{Impl: p.Impl}, nil
}

// Client returns a client-side implementation of Plugin that talks to the RPC server.
func (p *PluginNetRpcPlugin) Client(broker *go_plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &PluginRPC{client: c}, nil
}

/* ------------------------
   RPC types and arguments
   ------------------------ */

// DriverMeta is the minimal information about a driver instance returned
// by the plugin so the host can know the name for logging, etc.
type DriverMeta struct {
	Index  int           `json:"index"`
	Config driver.Config `json:"config"`
}

// ListRPCArgs wraps arguments for List RPC call.
type ListRPCArgs struct {
	DriverIndex int            `json:"driver_index"`
	Dir         model.Obj      `json:"dir"`
	ListArgs    model.ListArgs `json:"list_args"`
}

// LinkRPCArgs wraps arguments for Link RPC call.
type LinkRPCArgs struct {
	DriverIndex int            `json:"driver_index"`
	File        model.Obj      `json:"file"`
	LinkArgs    model.LinkArgs `json:"link_args"`
}

// GenericDriverIndex is used for simple requests that only need the driver index.
type GenericDriverIndex struct {
	DriverIndex int `json:"driver_index"`
}

/* ------------------------
   Server-side implementation
   (runs in the plugin process)
   ------------------------ */

// PluginRPCServer is the RPC server that exposes Plugin methods and forwards
// driver operations back to the actual in-process Plugin implementation.
type PluginRPCServer struct {
	Impl Plugin
}

func (s *PluginRPCServer) Info(_ struct{}, resp *PluginInfo) error {
	*resp = s.Impl.Info()
	return nil
}

// Drivers returns a list of DriverMeta; it does not return actual driver proxies.
// The host will create driver proxies that call back to this server with the driver index.
func (s *PluginRPCServer) Drivers(_ struct{}, resp *[]DriverMeta) error {
	drvs := s.Impl.Drivers()
	out := make([]DriverMeta, 0, len(drvs))
	for i, d := range drvs {
		out = append(out, DriverMeta{
			Index:  i,
			Config: d.Config(),
		})
	}
	*resp = out
	return nil
}

// List forwards List calls to the actual driver in the plugin process.
func (s *PluginRPCServer) List(args ListRPCArgs, resp *[]model.Obj) error {
	drivers := s.Impl.Drivers()
	if args.DriverIndex < 0 || args.DriverIndex >= len(drivers) {
		return ErrInvalidDriverIndex(args.DriverIndex)
	}
	d := drivers[args.DriverIndex]
	objs, err := d.List(context.Background(), args.Dir, args.ListArgs)
	if err != nil {
		return err
	}
	*resp = objs
	return nil
}

// Link forwards Link calls to the actual driver in the plugin process.
func (s *PluginRPCServer) Link(args LinkRPCArgs, resp **model.Link) error {
	drivers := s.Impl.Drivers()
	if args.DriverIndex < 0 || args.DriverIndex >= len(drivers) {
		return ErrInvalidDriverIndex(args.DriverIndex)
	}
	d := drivers[args.DriverIndex]
	link, err := d.Link(context.Background(), args.File, args.LinkArgs)
	if err != nil {
		return err
	}
	*resp = link
	return nil
}

// Config returns the Config for a given driver index.
func (s *PluginRPCServer) Config(args GenericDriverIndex, resp *driver.Config) error {
	drivers := s.Impl.Drivers()
	if args.DriverIndex < 0 || args.DriverIndex >= len(drivers) {
		return ErrInvalidDriverIndex(args.DriverIndex)
	}
	*resp = drivers[args.DriverIndex].Config()
	return nil
}

// Init calls the driver's Init(ctx) and returns any error.
func (s *PluginRPCServer) Init(args GenericDriverIndex, resp *struct{}) error {
	drivers := s.Impl.Drivers()
	if args.DriverIndex < 0 || args.DriverIndex >= len(drivers) {
		return ErrInvalidDriverIndex(args.DriverIndex)
	}
	return drivers[args.DriverIndex].Init(context.Background())
}

// Drop calls the driver's Drop(ctx) and returns any error.
func (s *PluginRPCServer) Drop(args GenericDriverIndex, resp *struct{}) error {
	drivers := s.Impl.Drivers()
	if args.DriverIndex < 0 || args.DriverIndex >= len(drivers) {
		return ErrInvalidDriverIndex(args.DriverIndex)
	}
	return drivers[args.DriverIndex].Drop(context.Background())
}

/* ------------------------
   Client-side implementation
   (runs in the host process)
   ------------------------ */

// PluginRPC is the client-side proxy for Plugin. Calls are forwarded over RPC.
type PluginRPC struct {
	client *rpc.Client
}

func (p *PluginRPC) Info() PluginInfo {
	var resp PluginInfo
	// ignore error here — plugin.Info is expected to exist. Caller can handle zero-valued resp.
	_ = p.client.Call("Plugin.Info", struct{}{}, &resp)
	return resp
}

func (p *PluginRPC) Drivers() PluginDriver {
	var metas []DriverMeta
	if err := p.client.Call("Plugin.Drivers", struct{}{}, &metas); err != nil {
		// return empty slice on error
		return nil
	}
	drivers := make([]driver.Driver, 0, len(metas))
	for _, m := range metas {
		// create a proxy driver that will call back to the plugin server with the driver index
		drivers = append(drivers, &DriverProxy{
			client: p.client,
			index:  m.Index,
			meta:   m,
		})
	}
	return drivers
}

/* ------------------------
   DriverProxy implements driver.Driver by forwarding calls over RPC to the plugin process.
   This proxy supports the Reader and Meta parts of the driver interface used by the host.
   Extend it to support additional driver methods (Put, Remove, etc.) as needed.
   ------------------------ */

type DriverProxy struct {
	client *rpc.Client
	index  int
	meta   DriverMeta
}

// Config implements driver.Meta.Config by fetching the config over RPC.
func (p *DriverProxy) Config() driver.Config {
	var cfg driver.Config
	_ = p.client.Call("Plugin.Config", GenericDriverIndex{DriverIndex: p.index}, &cfg)
	return cfg
}

// GetStorage is not proxied in this example. Return nil.
// Implement and proxy as needed.
func (p *DriverProxy) GetStorage() *model.Storage {
	return nil
}

func (p *DriverProxy) SetStorage(s model.Storage) {
	// no-op in proxy; implement RPC if required
}

func (p *DriverProxy) GetAddition() driver.Additional {
	return nil
}

func (p *DriverProxy) Init(ctx context.Context) error {
	return p.client.Call("Plugin.Init", GenericDriverIndex{DriverIndex: p.index}, &struct{}{})
}

func (p *DriverProxy) Drop(ctx context.Context) error {
	return p.client.Call("Plugin.Drop", GenericDriverIndex{DriverIndex: p.index}, &struct{}{})
}

// List forwards List over RPC.
func (p *DriverProxy) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	var resp []model.Obj
	rpcArgs := ListRPCArgs{
		DriverIndex: p.index,
		Dir:         dir,
		ListArgs:    args,
	}
	err := p.client.Call("Plugin.List", rpcArgs, &resp)
	return resp, err
}

// Link forwards Link over RPC.
func (p *DriverProxy) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var resp *model.Link
	rpcArgs := LinkRPCArgs{
		DriverIndex: p.index,
		File:        file,
		LinkArgs:    args,
	}
	err := p.client.Call("Plugin.Link", rpcArgs, &resp)
	return resp, err
}

/* ------------------------
   Errors and helpers
   ------------------------ */

type InvalidDriverIndexError int

func (e InvalidDriverIndexError) Error() string {
	return "invalid driver index"
}

func ErrInvalidDriverIndex(i int) error {
	return InvalidDriverIndexError(i)
}
