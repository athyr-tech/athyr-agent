package plugin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBridge_HTTPGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local http = require("http")
		local resp = http.get("` + srv.URL + `")
		result_body = resp.body
		result_status = resp.status
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	body := sb.L.GetGlobal("result_body")
	if body.String() != `{"status":"ok"}` {
		t.Errorf("http.get body = %v, want {\"status\":\"ok\"}", body.String())
	}

	status := sb.L.GetGlobal("result_status")
	if status.String() != "200" {
		t.Errorf("http.get status = %v, want 200", status.String())
	}
}

func TestBridge_HTTPPost(t *testing.T) {
	var receivedBody string
	var receivedContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedBody = string(buf)
		receivedContentType = r.Header.Get("Content-Type")
		w.Write([]byte("accepted"))
	}))
	defer srv.Close()

	sb, err := NewSandbox(nil)
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local http = require("http")
		local resp = http.post("` + srv.URL + `", '{"msg":"hello"}', {["Content-Type"] = "application/json"})
		result_body = resp.body
	`)
	if err != nil {
		t.Fatalf("DoString() error = %v", err)
	}

	if receivedBody != `{"msg":"hello"}` {
		t.Errorf("server received body = %v, want {\"msg\":\"hello\"}", receivedBody)
	}
	if receivedContentType != "application/json" {
		t.Errorf("server received Content-Type = %v, want application/json", receivedContentType)
	}
}

func TestBridge_HTTPRestricted(t *testing.T) {
	sb, err := NewSandbox([]string{"http"})
	if err != nil {
		t.Fatalf("NewSandbox() error = %v", err)
	}
	defer sb.Close()

	RegisterBridge(sb)

	err = sb.DoString(`
		local http = require("http")
		http.get("http://localhost:1234")
	`)
	if err == nil {
		t.Error("http.get should be restricted")
	}
}
