package common

type Resp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type PageResp struct {
	Content interface{} `json:"content"`
	Total   int64       `json:"total"`
}

// jsonResult is a standard JSON response structure
// Just used in documentation
type jsonResult struct {
	Code int         `json:"code"`
	Msg  string      `json:"message"`
	Data interface{} `json:"data"`
}

var _ jsonResult

// mapResult is special JsonResult with map data
// Just used in documentation
type mapResult struct {
}

var _ mapResult
