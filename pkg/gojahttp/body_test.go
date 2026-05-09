package gojahttp

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseBodyMultipartFormFields(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("title", "Trail notes"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("tag", "Planning"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("attachment", "note.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "/upload", strings.NewReader(body.String()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	parsed, raw, err := parseBody(req)
	if err != nil {
		t.Fatalf("parseBody() error = %v", err)
	}
	if raw == "" {
		t.Fatalf("expected raw multipart body")
	}
	fields, ok := parsed.(map[string]any)
	if !ok {
		t.Fatalf("parsed body type = %T, want map[string]any", parsed)
	}
	if fields["title"] != "Trail notes" || fields["tag"] != "Planning" {
		t.Fatalf("parsed fields = %#v", fields)
	}
	if req.MultipartForm == nil || len(req.MultipartForm.File["attachment"]) != 1 {
		t.Fatalf("expected multipart file metadata, got %#v", req.MultipartForm)
	}
}
