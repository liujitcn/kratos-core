package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	kratosHTTP "github.com/go-kratos/kratos/v3/transport/http"
)

// TestRegisterDataStaticRoute 验证 Core 将本地 OSS 根目录映射到 /data/。
func TestRegisterDataStaticRoute(t *testing.T) {
	rootDirectory := t.TempDir()
	filePath := filepath.Join(rootDirectory, "message", "images", "2026", "09", "03", "image.jpeg")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("image-content"), 0o644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	server := kratosHTTP.NewServer()
	registerDataStaticRoute(server, rootDirectory)
	request := httptest.NewRequest(http.MethodGet, "/data/message/images/2026/09/03/image.jpeg", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("数据文件路由状态码 = %d, want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Body.String() != "image-content" {
		t.Fatalf("数据文件路由响应 = %q, want %q", recorder.Body.String(), "image-content")
	}
}

// TestRegisterLocalSPARoutes 验证本地前端目录按目录名自动注册并支持单页回退。
func TestRegisterLocalSPARoutes(t *testing.T) {
	rootDirectory := t.TempDir()
	adminDirectory := filepath.Join(rootDirectory, "data", "admin")
	if err := os.MkdirAll(filepath.Join(adminDirectory, "assets"), 0o755); err != nil {
		t.Fatalf("创建前端测试目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDirectory, "index.html"), []byte("index-content"), 0o644); err != nil {
		t.Fatalf("创建前端入口失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDirectory, "assets", "app.js"), []byte("asset-content"), 0o644); err != nil {
		t.Fatalf("创建前端静态资源失败: %v", err)
	}

	server := kratosHTTP.NewServer()
	registerLocalSPARoutes(server, filepath.Join(rootDirectory, "data"))

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "入口", path: "/admin", want: "index-content"},
		{name: "静态资源", path: "/admin/assets/app.js", want: "asset-content"},
		{name: "单页回退", path: "/admin/dashboard", want: "index-content"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("前端路由状态码 = %d, want %d", recorder.Code, http.StatusOK)
			}
			if recorder.Body.String() != test.want {
				t.Fatalf("前端路由响应 = %q, want %q", recorder.Body.String(), test.want)
			}
		})
	}
}

// TestStaticAndOSSRootsRemainSeparate 验证自定义本地 OSS 根目录不影响固定 H5 静态目录。
func TestStaticAndOSSRootsRemainSeparate(t *testing.T) {
	rootDirectory := t.TempDir()
	staticRootDirectory := filepath.Join(rootDirectory, "data")
	adminDirectory := filepath.Join(staticRootDirectory, "admin")
	if err := os.MkdirAll(adminDirectory, 0o755); err != nil {
		t.Fatalf("创建 H5 测试目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDirectory, "index.html"), []byte("h5-content"), 0o644); err != nil {
		t.Fatalf("创建 H5 入口失败: %v", err)
	}

	ossRootDirectory := filepath.Join(rootDirectory, "uploads")
	filePath := filepath.Join(ossRootDirectory, "image", "avatar.png")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("创建上传目录失败: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("upload-content"), 0o644); err != nil {
		t.Fatalf("创建上传文件失败: %v", err)
	}

	server := kratosHTTP.NewServer()
	registerDataStaticRoute(server, ossRootDirectory)
	registerLocalSPARoutes(server, staticRootDirectory)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "H5 入口", path: "/admin", want: "h5-content"},
		{name: "上传文件", path: "/data/image/avatar.png", want: "upload-content"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("路由状态码 = %d, want %d", recorder.Code, http.StatusOK)
			}
			if recorder.Body.String() != test.want {
				t.Fatalf("路由响应 = %q, want %q", recorder.Body.String(), test.want)
			}
		})
	}
}
