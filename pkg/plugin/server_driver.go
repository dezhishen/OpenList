package plugin

import (
	"context"
	"errors"
	"net/rpc"

	"github.com/hashicorp/go-plugin"

	"github.com/OpenListTeam/OpenList/v4/pkg/driver"
	"github.com/OpenListTeam/OpenList/v4/pkg/model"
)

// RPCDriverServer is the RPC server that delegates calls to a real driver.Driver implementation.
type RPCDriverServer struct {
	Impl driver.Driver
}

// RPCDriverClient is an RPC client that implements driver.Driver by calling the RPC server.
type RPCDriverClient struct {
	client *rpc.Client
}

// RPCDriverPlugin implements plugin.Plugin so we can serve/consume Drivers with hashicorp/go-plugin.
type RPCDriverPlugin struct {
	Impl driver.Driver
}

func (p *RPCDriverPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RPCDriverServer{Impl: p.Impl}, nil
}

func (p *RPCDriverPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCDriverClient{client: c}, nil
}

// ------------------ Meta ------------------

// Config
func (s *RPCDriverServer) Config(args struct{}, resp *driver.Config) error {
	*resp = s.Impl.Config()
	return nil
}

func (c *RPCDriverClient) Config() driver.Config {
	var resp driver.Config
	err := c.client.Call("Plugin.Config", struct{}{}, &resp)
	if err != nil {
		// panic to match the pattern in your example; you can change to returning zero value + error if preferred
		panic(err)
	}
	return resp
}

// GetStorage
func (s *RPCDriverServer) GetStorage(args struct{}, resp *model.Storage) error {
	st := s.Impl.GetStorage()
	if st == nil {
		// return zero value (nil) to caller
		*resp = model.Storage{}
		return nil
	}
	*resp = *st
	return nil
}

func (c *RPCDriverClient) GetStorage() *model.Storage {
	var resp model.Storage
	err := c.client.Call("Plugin.GetStorage", struct{}{}, &resp)
	if err != nil {
		panic(err)
	}
	return &resp
}

// SetStorage
type setStorageArgs struct {
	Storage model.Storage
}

func (s *RPCDriverServer) SetStorage(args setStorageArgs, resp *struct{}) error {
	s.Impl.SetStorage(args.Storage)
	return nil
}

func (c *RPCDriverClient) SetStorage(s model.Storage) {
	_args := setStorageArgs{Storage: s}
	var resp struct{}
	if err := c.client.Call("Plugin.SetStorage", _args, &resp); err != nil {
		panic(err)
	}
}

// GetAddition
func (s *RPCDriverServer) GetAddition(args struct{}, resp *driver.Additional) error {
	*resp = s.Impl.GetAddition()
	return nil
}

func (c *RPCDriverClient) GetAddition() driver.Additional {
	var resp driver.Additional
	if err := c.client.Call("Plugin.GetAddition", struct{}{}, &resp); err != nil {
		panic(err)
	}
	return resp
}

// Init
func (s *RPCDriverServer) Init(args struct{}, resp *struct{}) error {
	return s.Impl.Init(context.Background())
}

func (c *RPCDriverClient) Init(ctx context.Context) error {
	// context cannot be transported over net/rpc easily; use background in server.
	var resp struct{}
	err := c.client.Call("Plugin.Init", struct{}{}, &resp)
	return err
}

// Drop
func (s *RPCDriverServer) Drop(args struct{}, resp *struct{}) error {
	return s.Impl.Drop(context.Background())
}

func (c *RPCDriverClient) Drop(ctx context.Context) error {
	var resp struct{}
	err := c.client.Call("Plugin.Drop", struct{}{}, &resp)
	return err
}

// ------------------ Reader ------------------
type listArgs struct {
	Dir  model.Obj
	Opts model.ListArgs
}

func (s *RPCDriverServer) List(args listArgs, resp *[]model.Obj) error {
	r, err := s.Impl.List(context.Background(), args.Dir, args.Opts)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	var resp []model.Obj
	_args := listArgs{Dir: dir, Opts: args}
	err := c.client.Call("Plugin.List", _args, &resp)
	return resp, err
}

type linkArgs struct {
	File model.Obj
	Args model.LinkArgs
}

func (s *RPCDriverServer) Link(args linkArgs, resp *model.Link) error {
	r, err := s.Impl.Link(context.Background(), args.File, args.Args)
	if err != nil {
		return err
	}
	*resp = *r
	return nil
}

func (c *RPCDriverClient) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	var resp model.Link
	_args := linkArgs{File: file, Args: args}
	err := c.client.Call("Plugin.Link", _args, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ------------------ GetRooter / Getter / GetObjInfo ------------------
func (s *RPCDriverServer) GetRoot(args struct{}, resp *model.Obj) error {
	r, err := s.Impl.(driver.GetRooter).GetRoot(context.Background())
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) GetRoot(ctx context.Context) (model.Obj, error) {
	var resp model.Obj
	err := c.client.Call("Plugin.GetRoot", struct{}{}, &resp)
	return resp, err
}

type getArgs struct {
	Path string
}

func (s *RPCDriverServer) Get(args getArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.Getter).Get(context.Background(), args.Path)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) Get(ctx context.Context, path string) (model.Obj, error) {
	var resp model.Obj
	_args := getArgs{Path: path}
	err := c.client.Call("Plugin.Get", _args, &resp)
	return resp, err
}

func (s *RPCDriverServer) GetObjInfo(args getArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.GetObjInfo).GetObjInfo(context.Background(), args.Path)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) GetObjInfo(ctx context.Context, path string) (model.Obj, error) {
	var resp model.Obj
	_args := getArgs{Path: path}
	err := c.client.Call("Plugin.GetObjInfo", _args, &resp)
	return resp, err
}

// ------------------ Mkdir/Move/Rename/Copy/Remove ------------------
type dirNameArgs struct {
	Parent model.Obj
	Name   string
}

func (s *RPCDriverServer) MakeDir(args dirNameArgs, resp *struct{}) error {
	return s.Impl.(driver.Mkdir).MakeDir(context.Background(), args.Parent, args.Name)
}

func (c *RPCDriverClient) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) error {
	_args := dirNameArgs{Parent: parentDir, Name: dirName}
	var resp struct{}
	err := c.client.Call("Plugin.MakeDir", _args, &resp)
	return err
}

func (s *RPCDriverServer) MakeDirResult(args dirNameArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.MkdirResult).MakeDir(context.Background(), args.Parent, args.Name)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) MakeDirResult(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	var resp model.Obj
	_args := dirNameArgs{Parent: parentDir, Name: dirName}
	err := c.client.Call("Plugin.MakeDirResult", _args, &resp)
	return resp, err
}

type srcDstArgs struct {
	Src model.Obj
	Dst model.Obj
}

func (s *RPCDriverServer) Move(args srcDstArgs, resp *struct{}) error {
	return s.Impl.(driver.Move).Move(context.Background(), args.Src, args.Dst)
}

func (c *RPCDriverClient) Move(ctx context.Context, srcObj, dstDir model.Obj) error {
	_args := srcDstArgs{Src: srcObj, Dst: dstDir}
	var resp struct{}
	err := c.client.Call("Plugin.Move", _args, &resp)
	return err
}

func (s *RPCDriverServer) MoveResult(args srcDstArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.MoveResult).Move(context.Background(), args.Src, args.Dst)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) MoveResult(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	var resp model.Obj
	_args := srcDstArgs{Src: srcObj, Dst: dstDir}
	err := c.client.Call("Plugin.MoveResult", _args, &resp)
	return resp, err
}

type renameArgs struct {
	Src     model.Obj
	NewName string
}

func (s *RPCDriverServer) Rename(args renameArgs, resp *struct{}) error {
	return s.Impl.(driver.Rename).Rename(context.Background(), args.Src, args.NewName)
}

func (c *RPCDriverClient) Rename(ctx context.Context, srcObj model.Obj, newName string) error {
	_args := renameArgs{Src: srcObj, NewName: newName}
	var resp struct{}
	err := c.client.Call("Plugin.Rename", _args, &resp)
	return err
}

func (s *RPCDriverServer) RenameResult(args renameArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.RenameResult).Rename(context.Background(), args.Src, args.NewName)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) RenameResult(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	var resp model.Obj
	_args := renameArgs{Src: srcObj, NewName: newName}
	err := c.client.Call("Plugin.RenameResult", _args, &resp)
	return resp, err
}

func (s *RPCDriverServer) Copy(args srcDstArgs, resp *struct{}) error {
	return s.Impl.(driver.Copy).Copy(context.Background(), args.Src, args.Dst)
}

func (c *RPCDriverClient) Copy(ctx context.Context, srcObj, dstDir model.Obj) error {
	_args := srcDstArgs{Src: srcObj, Dst: dstDir}
	var resp struct{}
	err := c.client.Call("Plugin.Copy", _args, &resp)
	return err
}

func (s *RPCDriverServer) CopyResult(args srcDstArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.CopyResult).Copy(context.Background(), args.Src, args.Dst)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) CopyResult(ctx context.Context, srcObj, dstDir model.Obj) ([]model.Obj, error) {
	var resp []model.Obj
	_args := srcDstArgs{Src: srcObj, Dst: dstDir}
	err := c.client.Call("Plugin.CopyResult", _args, &resp)
	return resp, err
}

type removeArgs struct {
	Obj model.Obj
}

func (s *RPCDriverServer) Remove(args removeArgs, resp *struct{}) error {
	return s.Impl.(driver.Remove).Remove(context.Background(), args.Obj)
}

func (c *RPCDriverClient) Remove(ctx context.Context, obj model.Obj) error {
	_args := removeArgs{Obj: obj}
	var resp struct{}
	err := c.client.Call("Plugin.Remove", _args, &resp)
	return err
}

// ------------------ Put / PutResult / PutURL ------------------

// Note: streaming types (model.FileStreamer, UpdateProgress) must be gob-registered in both processes.
// This RPC wrapper will transmit the file streamer value over gob - ensure concrete types are registered.
type putArgs struct {
	Dst model.Obj
	// File and Up are interface-like; they rely on gob-registered concrete types.
	File model.FileStreamer
	Up   driver.UpdateProgress
}

func (s *RPCDriverServer) Put(args putArgs, resp *struct{}) error {
	return s.Impl.(driver.Put).Put(context.Background(), args.Dst, args.File, args.Up)
}

func (c *RPCDriverClient) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) error {
	_args := putArgs{Dst: dstDir, File: file, Up: up}
	var resp struct{}
	err := c.client.Call("Plugin.Put", _args, &resp)
	return err
}

func (s *RPCDriverServer) PutResult(args putArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.PutResult).Put(context.Background(), args.Dst, args.File, args.Up)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) PutResult(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	var resp model.Obj
	_args := putArgs{Dst: dstDir, File: file, Up: up}
	err := c.client.Call("Plugin.PutResult", _args, &resp)
	return resp, err
}

type putURLArgs struct {
	Dst  model.Obj
	Name string
	URL  string
}

func (s *RPCDriverServer) PutURL(args putURLArgs, resp *struct{}) error {
	return s.Impl.(driver.PutURL).PutURL(context.Background(), args.Dst, args.Name, args.URL)
}

func (c *RPCDriverClient) PutURL(ctx context.Context, dstDir model.Obj, name, url string) error {
	_args := putURLArgs{Dst: dstDir, Name: name, URL: url}
	var resp struct{}
	err := c.client.Call("Plugin.PutURL", _args, &resp)
	return err
}

func (s *RPCDriverServer) PutURLResult(args putURLArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.PutURLResult).PutURL(context.Background(), args.Dst, args.Name, args.URL)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) PutURLResult(ctx context.Context, dstDir model.Obj, name, url string) (model.Obj, error) {
	var resp model.Obj
	_args := putURLArgs{Dst: dstDir, Name: name, URL: url}
	err := c.client.Call("Plugin.PutURLResult", _args, &resp)
	return resp, err
}

// ------------------ Archive ------------------
type archiveMetaArgs struct {
	Obj  model.Obj
	Args model.ArchiveArgs
}

func (s *RPCDriverServer) GetArchiveMeta(args archiveMetaArgs, resp *model.ArchiveMeta) error {
	r, err := s.Impl.(driver.ArchiveReader).GetArchiveMeta(context.Background(), args.Obj, args.Args)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) GetArchiveMeta(ctx context.Context, obj model.Obj, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	var resp model.ArchiveMeta
	_args := archiveMetaArgs{Obj: obj, Args: args}
	err := c.client.Call("Plugin.GetArchiveMeta", _args, &resp)
	return resp, err
}

type archiveInnerArgs struct {
	Obj  model.Obj
	Args model.ArchiveInnerArgs
}

func (s *RPCDriverServer) ListArchive(args archiveInnerArgs, resp *[]model.Obj) error {
	r, err := s.Impl.(driver.ArchiveReader).ListArchive(context.Background(), args.Obj, args.Args)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) ListArchive(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	var resp []model.Obj
	_args := archiveInnerArgs{Obj: obj, Args: args}
	err := c.client.Call("Plugin.ListArchive", _args, &resp)
	return resp, err
}

func (s *RPCDriverServer) Extract(args archiveInnerArgs, resp *model.Link) error {
	r, err := s.Impl.(driver.ArchiveReader).Extract(context.Background(), args.Obj, args.Args)
	if err != nil {
		return err
	}
	*resp = *r
	return nil
}

func (c *RPCDriverClient) Extract(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (*model.Link, error) {
	var resp model.Link
	_args := archiveInnerArgs{Obj: obj, Args: args}
	err := c.client.Call("Plugin.Extract", _args, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *RPCDriverServer) ArchiveGet(args archiveInnerArgs, resp *model.Obj) error {
	r, err := s.Impl.(driver.ArchiveGetter).ArchiveGet(context.Background(), args.Obj, args.Args)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) ArchiveGet(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (model.Obj, error) {
	var resp model.Obj
	_args := archiveInnerArgs{Obj: obj, Args: args}
	err := c.client.Call("Plugin.ArchiveGet", _args, &resp)
	return resp, err
}

type archiveDecompressArgs struct {
	Src  model.Obj
	Dst  model.Obj
	Args model.ArchiveDecompressArgs
}

func (s *RPCDriverServer) ArchiveDecompress(args archiveDecompressArgs, resp *struct{}) error {
	return s.Impl.(driver.ArchiveDecompress).ArchiveDecompress(context.Background(), args.Src, args.Dst, args.Args)
}

func (c *RPCDriverClient) ArchiveDecompress(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) error {
	_args := archiveDecompressArgs{Src: srcObj, Dst: dstDir, Args: args}
	var resp struct{}
	err := c.client.Call("Plugin.ArchiveDecompress", _args, &resp)
	return err
}

func (s *RPCDriverServer) ArchiveDecompressResult(args archiveDecompressArgs, resp *[]model.Obj) error {
	r, err := s.Impl.(driver.ArchiveDecompressResult).ArchiveDecompress(context.Background(), args.Src, args.Dst, args.Args)
	if err != nil {
		return err
	}
	*resp = r
	return nil
}

func (c *RPCDriverClient) ArchiveDecompressResult(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) ([]model.Obj, error) {
	var resp []model.Obj
	_args := archiveDecompressArgs{Src: srcObj, Dst: dstDir, Args: args}
	err := c.client.Call("Plugin.ArchiveDecompressResult", _args, &resp)
	return resp, err
}

// ------------------ WithDetails ------------------
func (s *RPCDriverServer) GetDetails(args struct{}, resp *model.StorageDetails) error {
	r, err := s.Impl.(driver.WithDetails).GetDetails(context.Background())
	if err != nil {
		return err
	}
	*resp = *r
	return nil
}

func (c *RPCDriverClient) GetDetails(ctx context.Context) (*model.StorageDetails, error) {
	var resp model.StorageDetails
	err := c.client.Call("Plugin.GetDetails", struct{}{}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ------------------ Reference (not supported over RPC) ------------------

// Passing a driver.Driver (interface) from host -> plugin or plugin -> host requires a proxy and mux broker.
// Implementing full proxying is possible but out of scope for this file, so we expose a clear error.
func (s *RPCDriverServer) InitReference(args struct{}, resp *struct{}) error {
	// If the underlying implementation supports InitReference and expects a local driver.Driver,
	// it cannot be satisfied over simple net/rpc without proxying. Return an explicit error.
	return errors.New("InitReference over simple RPC is not supported; use a proxied driver.Driver or implement InitReference locally")
}

func (c *RPCDriverClient) InitReference(storage driver.Driver) error {
	return errors.New("InitReference over simple RPC is not supported; use a proxied driver.Driver or implement InitReference locally")
}
