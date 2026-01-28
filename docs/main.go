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

var patchedVersionYaml []byte
var patchedVersionJson []byte
var patchOnce sync.Once

func patchVersion() {
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
		patchedVersionYaml = reYaml.ReplaceAll(yamlBytes, []byte("${1} "+version))

		// 替换 JSON 中的 "version": "XXX"
		reJson := regexp.MustCompile(`"version":\s*"[^"]*"`)
		patchedVersionJson = reJson.ReplaceAll(jsonBytes, []byte("\"version\": \""+version+"\""))
	})
}

func InitDoc(r *gin.RouterGroup) {
	if conf.Conf.Doc.Enable == false {
		log.Debugf("doc is disabled")
		return
	}
	// 提供 /doc.yaml 路由
	log.Debugf("doc is enabled")
	patchVersion()
	r.GET("/doc.yaml", func(c *gin.Context) {
		if patchedVersionYaml == nil {
			c.String(404, "not found")
			return
		}
		c.Data(http.StatusOK, "application/x-yaml", patchedVersionYaml)
	})
	r.GET("/doc.json", func(c *gin.Context) {
		if patchedVersionJson == nil {
			c.String(404, "not found")
			return
		}
		c.Data(http.StatusOK, "application/json", patchedVersionJson)
	})
}
