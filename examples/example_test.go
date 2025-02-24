package examples

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
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
	if !assert.Len(t, getOp.Parameters, 3) {
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
	api := openapi.NewAPI("messages", openapi.WithInfo(openapi3.Info{
		Version:     "1.0.0",
		Description: "A simple messages API",
		Contact: &openapi3.Contact{
			Name:  "Korbin",
			Email: "ihezebin@qq.com",
		},
	}), openapi.WithServer(openapi3.Server{
		URL:         "http://localhost:8080",
		Description: "Local server",
	}))
	api.StripPkgPaths = []string{"github.com/ihezebin/openapi/example", "github.com/a-h/respond"}

	api.Get("/topic/{id}").
		HasPathParameter("id", openapi.PathParam{
			Description: "id of the topic",
			Regexp:      `\d+`,
		}).
		HasRequestModel(openapi.Model{
			Type: reflect.TypeOf(struct {
				Message string `json:"message"`
			}{}),
		}).
		HasResponseModel(http.StatusOK, openapi.ModelOf[models.Body[**models.Topic]]()).
		HasResponseModel(http.StatusBadRequest, openapi.ModelOf[models.Body[struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}]]()).
		HasResponseModel(http.StatusInternalServerError, openapi.ModelOf[models.Body[map[string]string]]()).
		HasResponseModel(http.StatusAccepted, openapi.ModelOf[models.Body[any]]()).
		HasResponseModel(http.StatusMovedPermanently, openapi.ModelOf[any]()).
		HasTags([]string{"Topic"}).
		HasDescription("Get one topic by id").
		HasSummary("getOneTopic").
		HasHeaderParameter("Authorization", openapi.HeaderParam{
			Description: "Bearer token",
			Required:    true,
			Type:        openapi.PrimitiveTypeString,
		}).
		HasQueryParameter("limit", openapi.QueryParam{
			Description: "limit",
			Required:    true,
		}).
		HasResponseHeader(http.StatusOK, "Token", openapi.HeaderParam{
			Description: "token",
			Required:    true,
			Type:        openapi.PrimitiveTypeString,
		}).HasDeprecated(true)

	return api
}
