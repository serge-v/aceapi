package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cgi"
	"net/http/httputil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type Config struct {
	Token     string
	TokenFile string
	CacheDir  string
}

var (
	conf        Config
	version     string
	date        string
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
	ioutil.WriteFile(conf.CacheDir+"/crontab.txt", buf, 0600)
	s := execCmd("crontab " + conf.CacheDir + "/crontab.txt")
	fmt.Fprintln(w, s)
	printCrontab(w)
}

func dumpReq(req *http.Request, w io.Writer) {
	fmt.Fprintf(w, "%+v\n", req)
	username, password, _ := req.BasicAuth()
	fmt.Fprintln(w, "user:password", username, password)

	buf, err := httputil.DumpRequest(req, true)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(w, string(buf))
}

func Sha256sum(fileName string) (string, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	sha := sha256.New()
	io.Copy(sha, f)
	sum := sha.Sum(nil)
	return hex.EncodeToString(sum), nil
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
	if len(token) == 0 || token != conf.Token {
		http.Error(rw, "error: invalid token", http.StatusForbidden)
		return
	}

	if path == "" || path == "/" {
		help := `
		Requests:
			GET req                               -- dump request
			POST cron                             -- set crontab
			GET cron                              -- get crontab
			GET ht                                -- dump ../.htaccess
			GET v                                 -- show version
			POST file?dst={path}&mode={mode}      -- upload file
			HEAD file?dst={path}                  -- get file attributes
			POST x                                -- execute command
		Headers:
			Token -- authorization token
		`
		fmt.Fprintf(rw, help)
		return
	}

	if path == "/req" {
		dumpReq(r, rw)
		return
	}

	if path == "/v" {
		fmt.Fprintln(rw, "version:", version)
		fmt.Fprintln(rw, "date:   ", date)
		dir, _ := os.Getwd()
		fmt.Fprintf(rw, "cwd: %s\n", dir)
		fmt.Fprintf(rw, "env: %s\n", os.Environ())
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
			http.Error(rw, "error: "+err.Error(), http.StatusOK)
			return
		}

		fmt.Fprintf(rw, "aceapi-v1: %x\n", sha256.Sum256(buf))
		return
	}

	if path == "/file" {
		if r.Method != "POST" {
			http.Error(rw, "error: post method required", http.StatusMethodNotAllowed)
			return
		}

		dst := r.URL.Query().Get("dst")
		if dst == "" {
			http.Error(rw, "error: no dst=fname parameter", http.StatusBadRequest)
			return
		}

		str := r.URL.Query().Get("mode")
		var mode os.FileMode = 0

		if len(str) > 0 {
			n, err := strconv.ParseInt(str, 8, 32)
			if err != nil {
				http.Error(rw, "error: cannot parse mode parameter", http.StatusBadRequest)
				return
			}
			if n < 0 || n > 0777 {
				http.Error(rw, "error: invalid file mode", http.StatusBadRequest)
				return
			}
			mode = os.FileMode(n)
		}

		f, err := os.Create(dst)
		if err != nil {
			http.Error(rw, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		written, err := io.Copy(f, r.Body)
		if err != nil {
			http.Error(rw, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if mode > 0 {
			if err := os.Chmod(dst, mode); err != nil {
				http.Error(rw, "error: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		sha, err := Sha256sum(dst)
		if err != nil {
			http.Error(rw, "error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(rw, "written: %d\nsha: %s\n", written, sha)
		println("============", written)
		return
	}

	http.Error(rw, "error: invalid url", http.StatusBadRequest)
}

func initConfig() {
	conf.CacheDir = os.Getenv("HOME") + "/.cache/aceapi"
	if _, err := os.Stat(conf.CacheDir); os.IsNotExist(err) {
		if err := os.Mkdir(conf.CacheDir, 0700); err != nil {
			panic(err)
		}
	}

	conf.TokenFile = os.Getenv("HOME") + "/.config/aceapi/token.txt"
	buf, err := ioutil.ReadFile(conf.TokenFile)
	if err != nil {
		panic("cannot read token from " + conf.TokenFile)
	}
	conf.Token = strings.Trim(string(buf), "\r\n ")
}

func main() {
	flag.Parse()
	if *showVersion {
		fmt.Println("version:", version)
		fmt.Println("date:   ", date)
		return
	}

	initConfig()
	err := cgi.Serve(handler{})
	if err != nil {
		panic(err)
	}
}
