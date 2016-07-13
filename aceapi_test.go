package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"strings"
	"testing"
)

var server *httptest.Server

func testInit() {
	init_config()
	version = "test_version"
	date = "test_date"
	mux := http.NewServeMux()
	mux.Handle("/v1/", &handler{})
	mux.Handle("/updater/", &handler{})

	server = httptest.NewTLSServer(mux)
}

func testClose() {
	server.Close()
}

func TestVersion(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(http.MethodGet, server.URL+"/v1/v", nil)
	req.Header.Add("Token", conf.token)

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := httputil.DumpResponse(res, true)
	if err != nil {
		panic(err)
	}

	s := string(buf)

	if strings.Index(s, "version: test_version\n") < 0 {
		t.Log(s)
		t.Fatal("invalid version")
	}

	if strings.Index(s, "date:    test_date\n") < 0 {
		t.Log(s)
		t.Fatal("invalid date")
	}
}

func TestUpload(t *testing.T) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/upload?dst=1~.txt", nil)
	req.Header.Add("Token", conf.token)
	req.Body, _ = os.Open("aceapi_test.go")
	fi, err := os.Lstat("aceapi_test.go")
	expected := fmt.Sprintf("written: %d\n", fi.Size())

	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	buf, err := httputil.DumpResponse(res, true)
	if err != nil {
		panic(err)
	}

	s := string(buf)

	if strings.Index(s, expected) < 0 {
		t.Log(s)
		t.Fatal("invalid written size")
	}
}

func TestMain(m *testing.M) {
	testInit()
	res := m.Run()
	testClose()
	os.Exit(res)
}
