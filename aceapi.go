package main

import (
	"net/http"
	"net/http/cgi"
	"net/http/httputil"
	"fmt"
	"os/exec"
	"strings"
	"io/ioutil"
	"os"
	"io"
	"crypto/md5"
)

func execCmd(cmdline string) string {

        var out []byte
        var err error

        if out, err = exec.Command("bash", "-c", cmdline).CombinedOutput(); err != nil {
                s := fmt.Sprintf("%s: %s. %s", cmdline, string(out), err)
                return s
        }

        return string(out)
}

func printCrontab(w io.Writer) {
	s := strings.Trim(execCmd("crontab -l"), "\n")
	fmt.Fprintln(w, s)
}

func addCrontab(buf []byte, w io.Writer) {
	ioutil.WriteFile("/tmp/crontab.txt", buf, 0600)
	s := execCmd("crontab /tmp/crontab.txt")
	fmt.Fprintln(w, s)
	printCrontab(w)
}

func dumpReq(req *http.Request, w io.Writer) {
	fmt.Fprintf(w, "%+v\n", req)
	username, password, _ := req.BasicAuth()
	fmt.Fprintln(w, "user:password", username, password)

	buf, err := httputil.DumpRequest(req, true); if err != nil {
		panic(err)
	}
	fmt.Fprintln(w, string(buf))
}

func processFsRequest(r *http.Request, rw http.ResponseWriter) {
	dir, _ := os.Getwd()
	fmt.Fprintf(rw, "cwd: %s\n", dir)
	fmt.Fprintf(rw, "env: %s", os.Environ())
}

type handler struct {
}

func (handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

	if r.URL.Scheme != "https" {
		http.Error(rw, "error: http forbidden", http.StatusForbidden)
		return
	}

	token := r.Header.Get("Token")
	if len(token) == 0 || token != "8eh4ufb3j592g" {
		http.Error(rw, "error: invalid token", http.StatusForbidden)
		return
	}

	path := strings.Replace(r.URL.Path, "/v1", "", 1)

	if path == "" || path == "/" {
		fmt.Fprintf(rw, "req  -- dump request\n")
		fmt.Fprintf(rw, "cron -- crontab\n")
		return
	}

	if strings.HasPrefix(path, "/fs") {
		processFsRequest(r, rw)
		return
	}

	if path == "/req" {
		dumpReq(r, rw)
		return
	}
	
	if path == "/cron" {
		if r.Method == "POST" {
			buf, _ := ioutil.ReadAll(r.Body)
			addCrontab(buf, rw)
		} else {
			printCrontab(rw)
		}
		return
	}

	if path == "/updater/" {
		if r.Method != "POST" {
			http.Error(rw, "error: need binary body", http.StatusMethodNotAllowed)
			return
		}
		buf, _ := ioutil.ReadAll(r.Body)
		if err := ioutil.WriteFile("aceapi-v1", buf, 0755); err != nil {
			http.Error(rw, "error: " + err.Error(), http.StatusInternalServerError)
			return
		}
		
		fmt.Fprintf(rw, "aceapi-v1: %x\n", md5.Sum(buf))
		return
	}

	dumpReq(r, rw)
}

func main() {
	err := cgi.Serve(handler{})
	if err != nil {
		panic(err)
	}
}
