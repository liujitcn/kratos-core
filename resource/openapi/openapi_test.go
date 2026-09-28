package openapi

import (
	"context"
	"testing"

	"github.com/liujitcn/kratos-core/biz"
)

// TestGetOperationCachesParsedDocumentAndPreservesLocaleFallback 验证重复查询复用解析结果且保持语言回退。
func TestGetOperationCachesParsedDocumentAndPreservesLocaleFallback(t *testing.T) {
	registry := &Registry{}
	err := registry.Register(
		Document{
			Key:  "test",
			Name: "Test",
			Data: []byte(`openapi: 3.0.3
paths:
  /test:
    get:
      operationId: GetTest
      summary: default summary
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: string
`),
		},
		Document{
			Key:    "test",
			Name:   "Test",
			Locale: "zh-CN",
			Data: []byte(`openapi: 3.0.3
paths:
  /test:
    get:
      operationId: GetTest
      summary: localized summary
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: string
`),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	openAPI := NewOpenAPI(registry)
	for _, test := range []struct {
		locale      string
		cacheLocale string
		summary     string
	}{
		{locale: "zh-CN", cacheLocale: "zh-CN", summary: "localized summary"},
		{locale: "en-US", cacheLocale: "", summary: "default summary"},
	} {
		ctx := biz.WithLocale(context.Background(), test.locale)
		cache := registry.parsedDocuments[newDocumentKey("test", test.cacheLocale)]
		if cache.document != nil {
			t.Fatalf("locale %q was parsed before the first operation lookup", test.locale)
		}
		for index := range 2 {
			result, err := openAPI.GetOperation(ctx, "/test", "GET")
			if err != nil {
				t.Fatal(err)
			}
			if result.Summary != test.summary {
				t.Fatalf("summary = %q, want %q", result.Summary, test.summary)
			}
			if index == 0 {
				if cache.document == nil {
					t.Fatalf("locale %q did not cache the parsed document", test.locale)
				}
				cache.data = []byte("invalid: [")
			}
		}
	}
}
