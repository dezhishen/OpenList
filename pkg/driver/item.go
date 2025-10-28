package driver

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/pkg/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
)

func init() {
	gob.Register(&SimpleAdditional{})
	gob.Register(&RootPath{})
	gob.Register(&RootID{})
	gob.Register(map[string]any{})
	gob.Register(&model.Object{})
	gob.Register(&utils.HashInfo{})
	gob.Register(&utils.HashType{})
	gob.Register(&model.ObjThumb{})
	gob.Register(&model.OtherArgs{})
	gob.Register(&model.LinkArgs{})
	gob.Register(&model.ListArgs{})
	gob.Register(&model.Link{})
	gob.Register(&model.FsOtherArgs{})
	gob.Register(&model.ObjThumb{})
}

// 定义一个约束，限定只能是 RootPath 或 RootID
type RootConstraint interface {
	RootPath | RootID
}

type Additional interface {
	GetItems() []Item
	GetData() map[string]any
	GetRoot() any
	UnmarshalData(v any) error
}

// SimpleAdditional is a simple implementation of Additional
// Root must be RootPath or RootID
type SimpleAdditional struct {
	Items []Item         `json:"items"`
	Data  map[string]any `json:"data"`
	Root  any            `json:"root"`
}

func (s SimpleAdditional) GetString(key string, defaultValue string) string {
	if val, ok := s.Data[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func (s SimpleAdditional) GetBool(key string, defaultValue bool) bool {
	if val, ok := s.Data[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

func (s SimpleAdditional) GetItems() []Item {
	return s.Items
}
func (s SimpleAdditional) GetData() map[string]any {
	return s.Data
}
func (s SimpleAdditional) GetRoot() any {
	return s.Root
}

func NewSimpleAdditionalWithoutRoot(v any) (Additional, error) {
	var additional = SimpleAdditional{
		Root:  nil,
		Data:  make(map[string]any),
		Items: []Item{},
	}
	data, err := json.Marshal(v)
	if err != nil {
		return additional, err
	}
	err = json.Unmarshal(data, &additional.Data)
	if err != nil {
		return additional, err
	}
	// 读取和处理 struct 标签
	additional.Items = getAdditionalItems(reflect.TypeOf(v))
	return additional, nil
}

func NewSimpleAdditional[T RootConstraint](t T, v any) (Additional, error) {
	var additional = SimpleAdditional{
		Root:  t,
		Data:  make(map[string]any),
		Items: []Item{},
	}
	data, err := json.Marshal(v)
	if err != nil {
		return additional, err
	}
	err = json.Unmarshal(data, &additional.Data)
	if err != nil {
		return additional, err
	}
	// 读取和处理 struct 标签
	additional.Items = getAdditionalItems(reflect.TypeOf(v))
	return additional, nil
}

func (s SimpleAdditional) UnmarshalData(v any) error {
	if s.Data == nil {
		return nil
	}
	bs, err := json.Marshal(s.Data)
	if err != nil {
		return err
	}
	// 判断v是否为指针
	typeOfV := reflect.TypeOf(v)
	if typeOfV.Kind() != reflect.Ptr {
		return fmt.Errorf("v must be a pointer")
	}
	return json.Unmarshal(bs, v)
}

var _ Additional = (*SimpleAdditional)(nil)

type Select string

type Item struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Options  string `json:"options"`
	Required bool   `json:"required"`
	Help     string `json:"help"`
}

type Info struct {
	Common     []Item `json:"common"`
	Additional []Item `json:"additional"`
	Config     Config `json:"config"`
}

type IRootPath interface {
	GetRootPath() string
}

type IRootId interface {
	GetRootId() string
}

type RootPath struct {
	RootFolderPath string `json:"root_folder_path"`
}

type RootID struct {
	RootFolderID string `json:"root_folder_id"`
}

func (r RootPath) GetRootPath() string {
	return r.RootFolderPath
}

func (r *RootPath) SetRootPath(path string) {
	r.RootFolderPath = path
}

func (r RootID) GetRootId() string {
	return r.RootFolderID
}

func getAdditionalItems(t reflect.Type) []Item {
	var items []Item
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Type.Kind() == reflect.Struct {
			items = append(items, getAdditionalItems(field.Type)...)
			continue
		}
		tag := field.Tag
		ignore, ok1 := tag.Lookup("ignore")
		name, ok2 := tag.Lookup("json")
		if (ok1 && ignore == "true") || !ok2 {
			continue
		}
		item := Item{
			Name:     name,
			Type:     strings.ToLower(field.Type.Name()),
			Default:  tag.Get("default"),
			Options:  tag.Get("options"),
			Required: tag.Get("required") == "true",
			Help:     tag.Get("help"),
		}
		if tag.Get("type") != "" {
			item.Type = tag.Get("type")
		}
		// if item.Name == "root_folder_id" || item.Name == "root_folder_path" {
		// 	if item.Default == "" {
		// 		item.Default = defaultRoot
		// 	}
		// 	item.Required = item.Default != ""
		// }
		// set default type to string
		if item.Type == "" {
			item.Type = "string"
		}
		items = append(items, item)
	}
	return items
}
