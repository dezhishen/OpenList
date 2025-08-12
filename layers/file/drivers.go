package driver

import (
	"context"
)

type FileDriver interface {
	ListPath(ctx context.Context, path string)
	InfoFile(ctx context.Context, path string)
}
