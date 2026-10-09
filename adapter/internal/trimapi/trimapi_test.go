package trimapi

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTrimAPI 起一个伪官方 API(unix socket HTTP server),按 path 断言请求并回放响应。
type fakeTrimAPI struct {
	t       *testing.T
	socket  string
	handler func(t *testing.T, r *TrimappRequest) (int, any)
	srv     *http.Server
	ln      net.Listener
}

// TrimappRequest 捕获到的官方 API 请求体。
type TrimappRequest struct {
	ReqID   string          `json:"reqId"`
	Req     string          `json:"req"`
	AppName string          `json:"appName"`
	Data    json.RawMessage `json:"data"`
	Token   string
}

func newFakeTrimAPI(t *testing.T, handler func(*testing.T, *TrimappRequest) (int, any)) *fakeTrimAPI {
	t.Helper()
	// macOS sun_path 104 字节上限:用 /tmp 短路径。
	dir, err := os.MkdirTemp("/tmp", "pb-trimapi-*")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "api.sock")

	f := &fakeTrimAPI{t: t, socket: socket, handler: handler}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/trimapp", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var req TrimappRequest
		require.NoError(t, json.Unmarshal(body, &req))
		req.Token = r.Header.Get("Authorization")
		status, resp := handler(t, &req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(resp)
	})
	f.ln, err = net.Listen("unix", socket)
	require.NoError(t, err)
	f.srv = &http.Server{Handler: mux}
	go f.srv.Serve(f.ln)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = f.srv.Shutdown(ctx)
	})
	return f
}

func (f *fakeTrimAPI) client() *Client {
	return New(f.socket, "pigeonbox")
}

// envToken 注入 TRIM_API_TOKEN 并注册还原。
func envToken(t *testing.T, token string) {
	t.Helper()
	orig, had := os.LookupEnv("TRIM_API_TOKEN")
	_ = os.Setenv("TRIM_API_TOKEN", token)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("TRIM_API_TOKEN", orig)
		} else {
			_ = os.Unsetenv("TRIM_API_TOKEN")
		}
	})
}

func TestSharedFolders_Success(t *testing.T) {
	envToken(t, "tok-123")
	f := newFakeTrimAPI(t, func(t *testing.T, r *TrimappRequest) (int, any) {
		assert.Equal(t, "trim.file.getSharedAccessibleFolders", r.Req)
		assert.Equal(t, "pigeonbox", r.AppName)
		assert.Equal(t, "Bearer tok-123", r.Token)
		return 200, map[string]any{"reqId": r.ReqID, "code": 0, "msg": "", "data": map[string]any{
			"paths": []string{"/vol1/1000/data"},
		}}
	})

	paths, err := f.client().SharedFolders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"/vol1/1000/data"}, paths)
}

func TestSharedFolders_MissingToken(t *testing.T) {
	envToken(t, "")
	f := newFakeTrimAPI(t, func(t *testing.T, r *TrimappRequest) (int, any) {
		t.Fatal("无 token 不应发出请求")
		return 500, nil
	})
	_, err := f.client().SharedFolders(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TRIM_API_TOKEN")
}

func TestCall_APIError(t *testing.T) {
	envToken(t, "tok")
	f := newFakeTrimAPI(t, func(t *testing.T, r *TrimappRequest) (int, any) {
		// 官方语义:403=scope 未声明。
		return 403, map[string]any{"reqId": r.ReqID, "code": 200003, "msg": "Forbidden"}
	})
	_, err := f.client().SharedFolders(context.Background())
	require.Error(t, err)
	var apiErr *APIError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.HTTPStatus)
	assert.Equal(t, 200003, apiErr.Code)
}

func TestConvertPaths(t *testing.T) {
	envToken(t, "tok")
	f := newFakeTrimAPI(t, func(t *testing.T, r *TrimappRequest) (int, any) {
		assert.Equal(t, "trim.file.convertPath", r.Req)
		var data struct {
			Path     []string `json:"path"`
			Language string   `json:"language"`
		}
		require.NoError(t, json.Unmarshal(r.Data, &data))
		assert.Equal(t, "zh-CN", data.Language)
		return 200, map[string]any{"reqId": r.ReqID, "code": 0, "data": map[string]any{
			"status": 0,
			"result": []map[string]string{
				{"path": "/vol1/1000/photo", "semanticPath": "存储空间1/admin 的文件/photo"},
			},
		}}
	})
	out, err := f.client().ConvertPaths(context.Background(), "zh-CN", []string{"/vol1/1000/photo"})
	require.NoError(t, err)
	require.Len(t, out, 1)
	assert.Equal(t, "存储空间1/admin 的文件/photo", out[0].SemanticPath)
}
