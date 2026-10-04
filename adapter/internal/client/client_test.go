package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/filescodebox/fnos/adapter/internal/fnosconfig"
)

// 降级模式下不应创建 client(New 返回 nil)。
func TestNew_DisabledReturnsNil(t *testing.T) {
	c := New(fnosconfig.Config{Enabled: false})
	assert.Nil(t, c, "降级模式下不应创建 client")
}

// client.do 应请求到配置的 APIBase,并回传响应体与状态码。
func TestDo_HitsConfiguredBase(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		assert.Equal(t, "/test/path", r.URL.Path, "应请求到指定 path")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(fnosconfig.Config{Enabled: true, AppID: "a", AppSecret: "s", APIBase: srv.URL})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	body, status, err := c.Do(ctx, http.MethodGet, "/test/path", nil)

	assert.NoError(t, err)
	assert.True(t, called, "应请求到测试服务器")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, `{"ok":true}`, string(body))
}

// client.do 应在飞牛返回错误状态码时,仍回传状态码与响应体(由调用方判断)。
func TestDo_PropagatesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"err":"bad signature"}`))
	}))
	defer srv.Close()

	c := New(fnosconfig.Config{Enabled: true, AppID: "a", AppSecret: "s", APIBase: srv.URL})
	body, status, err := c.Do(context.Background(), http.MethodGet, "/x", nil)

	assert.NoError(t, err) // HTTP 层无错误,业务错误码由调用方处理
	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Contains(t, string(body), "bad signature")
}
