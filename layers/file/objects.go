package driver

type FileObject struct {
	Name string // 文件名称
	Size int64  // 文件大小
	Mask int16  // 文件权限
	Type bool   // 文件类型
}

type FileManage interface {
	GetLink(file FileObject, path string)
	Renamed(file FileObject, path string)
	Deleted(file FileObject, path string)
	CopyDir(file FileObject, path string)
	MoveDir(file FileObject, path string)
}
