package driver

import (
	"context"
)

type FileLayers interface {
	ListPath(ctx context.Context, path string)
	InfoFile(ctx context.Context, path string)
}
