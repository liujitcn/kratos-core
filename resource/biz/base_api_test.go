package biz

import (
	"testing"
)

func TestOpenAPIDataToBaseAPIInfersCommonOnlyServiceFromTerminal(t *testing.T) {
	openAPI := []byte(`openapi: 3.0.0
paths:
  /api/v1/admin/base/table-source/option:
    get:
      operationId: BaseTableSourceService_OptionBaseTableSource
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/common.v1.StringValues'
  /api/v1/admin/base/table-source/table/option:
    get:
      operationId: BaseTableSourceService_OptionBaseTable
      parameters:
        - name: source_name
          in: query
          schema:
            type: string
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/common.v1.StringValues'
  /api/v1/admin/base/user:
    get:
      operationId: BaseUserService_PageBaseUser
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/system.admin.v1.PageBaseUserResponse'
`)

	items, err := (&BaseAPICase{}).OpenAPIDataToBaseAPI(openAPI)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"/system.admin.v1.BaseTableSourceService/OptionBaseTableSource": false,
		"/system.admin.v1.BaseTableSourceService/OptionBaseTable":       false,
		"/system.admin.v1.BaseUserService/PageBaseUser":                 false,
	}
	for _, item := range items {
		if _, ok := want[item.Operation]; ok {
			want[item.Operation] = true
		}
	}
	for operation, found := range want {
		if !found {
			t.Errorf("operation %q was not generated", operation)
		}
	}
}

func TestOpenAPIDataToBaseAPIDoesNotGuessAmbiguousTerminalPackage(t *testing.T) {
	openAPI := []byte(`openapi: 3.0.0
paths:
  /api/v1/admin/common-only:
    get:
      operationId: CommonOnlyService_Get
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/common.v1.StringValues'
  /api/v1/admin/system:
    get:
      operationId: SystemService_Get
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/system.admin.v1.SystemResponse'
  /api/v1/admin/shop:
    get:
      operationId: ShopService_Get
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/shop.admin.v1.ShopResponse'
`)

	items, err := (&BaseAPICase{}).OpenAPIDataToBaseAPI(openAPI)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Operation == "/system.admin.v1.CommonOnlyService/Get" {
			t.Fatal("ambiguous terminal package should not be guessed")
		}
	}
}
