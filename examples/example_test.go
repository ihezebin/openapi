package examples

import (
	"net/http"
	"testing"

	"github.com/ihezebin/openapi"
	"github.com/ihezebin/openapi/examples/models"
	"github.com/stretchr/testify/assert"
)

func TestGenerateOpenAPISpec(t *testing.T) {
	// 创建API配置
	api := NewTestAPI()

	// 生成规范
	spec, err := api.Spec()
	if !assert.NoError(t, err, "生成OpenAPI规范不应返回错误") {
		return
	}

	// 生成 JSON
	data, err := api.Json()
	if !assert.NoError(t, err, "生成JSON不应返回错误") {
		return
	}
	if !assert.NotEmpty(t, data, "生成的JSON应该不为空") {
		return
	}

	// 验证基本结构
	if !assert.Equal(t, "3.0.0", spec.OpenAPI) {
		return
	}
	if !assert.Equal(t, "messages", spec.Info.Title) {
		return
	}

	// 验证路径
	pathItem := spec.Paths.Find("/topic/{id}")
	if !assert.NotNil(t, pathItem, "应存在/topic/{id}路径") {
		return
	}

	// 验证GET操作
	getOp := pathItem.Get
	if !assert.NotNil(t, getOp, "应存在GET方法") {
		return
	}
	if !assert.Equal(t, "getOneTopic", getOp.Summary) {
		return
	}
	if !assert.Contains(t, getOp.Tags, "Topic") {
		return
	}

	// 验证路径参数
	if !assert.Len(t, getOp.Parameters, 1) {
		return
	}
	param := getOp.Parameters[0].Value
	if !assert.Equal(t, "id", param.Name) {
		return
	}
	if !assert.Equal(t, "\\d+", param.Schema.Value.Pattern) {
		return
	}

	// 验证响应
	if !assert.NotNil(t, getOp.Responses.Status(200)) {
		return
	}
	if !assert.NotNil(t, getOp.Responses.Status(500)) {
		return
	}

	// 打印 json
	t.Log(string(data))
}

func NewTestAPI() *openapi.API {
	api := openapi.NewAPI("messages")
	api.StripPkgPaths = []string{"github.com/ihezebin/openapi/example", "github.com/a-h/respond"}

	api.Get("/topic/{id}").
		HasPathParameter("id", openapi.PathParam{
			Description: "id of the topic",
			Regexp:      `\d+`,
		}).
		HasResponseModel(http.StatusOK, openapi.ModelOf[models.Body[models.Topic]]()).
		HasResponseModel(http.StatusInternalServerError, openapi.ModelOf[models.Body[map[string]string]]()).
		HasTags([]string{"Topic"}).
		HasDescription("Get one topic by id").
		HasSummary("getOneTopic")

	return api
}
