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
	"flag"
)

type config struct {
	token string
}

var (
	conf config
	version string
	date string
	showVersion = flag.Bool("v", false, "show version")
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
	fmt.Fprintf(rw, "env: %s\n", os.Environ())
}

type handler struct {
}

func (handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {

//	if r.URL.Scheme != "https" {
//		http.Error(rw, "error: " + r.URL.Scheme + " forbidden", http.StatusForbidden)
//		return
//	}

	path := strings.Replace(r.URL.Path, "/v1", "", 1)

	if path == "/a" {
//		username, password, ok := r.BasicAuth()
		rw.Header().Add("WWW-Authenticate", `Basic realm="myrealm"`)
		http.Error(rw, "error: no auth", http.StatusUnauthorized)
		return
	}

	token := r.Header.Get("Token")
	if len(token) == 0 || token != conf.token {
		http.Error(rw, "error: invalid token", http.StatusForbidden)
		return
	}

	if path == "" || path == "/" {
		fmt.Fprintf(rw, "req      -- dump request\n")
		fmt.Fprintf(rw, "cron     -- crontab\n")
		fmt.Fprintf(rw, "ht       -- dump ../.htaccess\n")
		fmt.Fprintf(rw, "v        -- show version\n")
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

	if path == "/v" {
		fmt.Fprintln(rw, "version:", version)
		fmt.Fprintln(rw, "date:   ", date)
		return
	}

	if path == "/x" {
		if r.Method != "POST" {
			http.Error(rw, "error: need binary body", http.StatusMethodNotAllowed)
			return
		}
		buf, _ := ioutil.ReadAll(r.Body)
		fmt.Fprintln(rw, execCmd(string(buf)))
		return
	}

	if path == "/df" {
		fmt.Fprintln(rw, execCmd("df -h --local ${DOCUMENT_ROOT} 2>/dev/null"))
		return
	}

	if path == "/ht" {
		f, err := os.Open("../.htaccess")
		if err != nil {
			http.Error(rw, "error: cannot read .htaccess", http.StatusNotFound)
			return
		}
		io.Copy(rw, f)
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
			http.Error(rw, "error: post method required", http.StatusMethodNotAllowed)
			return
		}
		buf, _ := ioutil.ReadAll(r.Body)
		if err := ioutil.WriteFile("aceapi-v1", buf, 0744); err != nil {
			http.Error(rw, "error: " + err.Error(), http.StatusOK)
			return
		}

		fmt.Fprintf(rw, "aceapi-v1: %x\n", md5.Sum(buf))
		return
	}

	if path == "/upload" {
		if r.Method != "POST" {
			http.Error(rw, "error: post method required", http.StatusMethodNotAllowed)
			return
		}
		
		dst := r.URL.Query().Get("dst")
		if dst == "" {
			http.Error(rw, "error: no dst=fname parameter", http.StatusBadRequest)
			return
		}

		f, err := os.Create(dst)
		if err != nil {
			http.Error(rw, "error: " + err.Error(), http.StatusInternalServerError)
			return
		}

		written, err := io.Copy(f, r.Body)
		if err != nil {
			http.Error(rw, "error: " + err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(rw, "written: %d\n", written)
		return
	}

	dumpReq(r, rw)
}

func init_config() {
	buf, err := ioutil.ReadFile("token.txt")
	if err != nil {
		panic("cannot read token")
	}
	conf.token = strings.Trim(string(buf), "\r\n ")
}

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println("version:", version)
		fmt.Println("date:   ", date)
		return
	}
	
	init_config()
	err := cgi.Serve(handler{})
	if err != nil {
		panic(err)
	}
}
