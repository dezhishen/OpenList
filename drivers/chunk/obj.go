package chunk

import "github.com/OpenListTeam/OpenList/v4/pkg/model"

type chunkObject struct {
	model.Object
	chunkSizes []int64
}
