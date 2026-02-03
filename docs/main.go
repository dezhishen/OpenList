package docs

import (
	"embed"
	"net/http"
	"regexp"
	"sync"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

//go:embed *
var docsFS embed.FS

func InitDoc(r *gin.RouterGroup) {
	if conf.Conf.Doc.Enable == false {
		log.Debugf("doc is disabled")
		return
	}
	// 提供 /doc.yaml 路由
	log.Debugf("doc is enabled")
	patchDoc()
	r.GET("/doc.yaml", func(c *gin.Context) {
		if patchedYaml == nil {
			c.String(404, "not found")
			return
		}
		c.Data(http.StatusOK, "application/x-yaml", patchPathForYaml())
	})
	r.GET("/doc.json", func(c *gin.Context) {
		if patchedJson == nil {
			c.String(404, "not found")
			return
		}
		c.Data(http.StatusOK, "application/json", patchPathForJson())
	})
}

var patchedYaml []byte
var patchedJson []byte
var patchOnce sync.Once

func patchPathForYaml() []byte {
	if patchedYaml == nil {
		return nil
	}
	basePath := conf.URL.Path
	// 替换 YAML 中的 basePath: /
	reBasePath := regexp.MustCompile(`(?m)^(\s*basePath:).+$`)
	return reBasePath.ReplaceAll(patchedYaml, []byte("${1} "+basePath))

}

func patchPathForJson() []byte {
	if patchedJson == nil {
		return nil
	}
	basePath := conf.URL.Path
	// 替换 JSON 中的 "basePath": "/"
	reJsonBasePath := regexp.MustCompile(`"basePath":\s*"[^"]*"`)
	return reJsonBasePath.ReplaceAll(patchedJson, []byte("\"basePath\": \""+basePath+"\""))
}

func patchDoc() {
	patchOnce.Do(func() {
		version := conf.Conf.LastLaunchedVersion
		yamlBytes, err := docsFS.ReadFile("swagger.yaml")
		if err != nil {
			log.Errorf("read swagger.yaml error: %+v", err)
			return
		}
		jsonBytes, err := docsFS.ReadFile("swagger.json")
		if err != nil {
			log.Errorf("read swagger.json error: %+v", err)
			return
		}

		// 替换 YAML 中的 version: XXX
		reYaml := regexp.MustCompile(`(?m)^(\s*version:).+$`)
		patchedYaml = reYaml.ReplaceAll(yamlBytes, []byte("${1} "+version))

		// 替换 JSON 中的 "version": "XXX"
		reJson := regexp.MustCompile(`"version":\s*"[^"]*"`)
		patchedJson = reJson.ReplaceAll(jsonBytes, []byte("\"version\": \""+version+"\""))
	})
}
