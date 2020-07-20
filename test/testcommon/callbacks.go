package testcommon

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCallbackServer(t *testing.T, port string, params map[string]string) *httptest.Server {
	handler := handler(t, params)
	testServer := httptest.NewUnstartedServer(handler)
	l, _ := net.Listen("tcp", "127.0.0.1:"+port)
	testServer.Listener = l
	testServer.Start()
	return testServer
}

func handler(t *testing.T, params map[string]string) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		for key, expectedValue := range params {
			value := r.URL.Query().Get(key)
			if value != expectedValue && (key != "timestamp" && key != "authHash") {
				io.WriteString(w, "false")
				w.WriteHeader(http.StatusNotFound)
				t.Fatalf("HTTP Callback error: parameter (%q) expected value (%q) value (%q) ", key, expectedValue, value)
			}
		}
		t.Log("HTTP Callback OK")
		io.WriteString(w, "ok")
	}

	return http.HandlerFunc(fn)
}
