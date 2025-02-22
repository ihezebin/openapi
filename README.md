# OPENAPI

Document a REST API with an OpenAPI 3.0 specification.

* Code, not configuration.
* No magic comments, tags, or decorators.
* Use with or without a Go web framework.
* Populates schema automatically using reflection.

## Copyright

This project is based on [https://github.com/a-h/rest](https://github.com/a-h/rest) by [Adrian Hesketh](https://github.com/a-h).

Copyright (c) [2025] [Korbin]

This software is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Why would I want to use this?

* Add OpenAPI documentation to an API.
  * Create a `swagger.json` or `swagger.yaml` file.
* Serve the Swagger UI to customers.

## Examples

See the [./examples](./examples) directory for complete examples.

### Create an OpenAPI 3.0 (swagger) file

```go
// Configure the models.
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

 api.Json()
```

## Tasks

### test

```
go test ./...
```

### run-example

Dir: ./examples/stdlib

```
go run main.go
```
