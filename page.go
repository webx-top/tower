package main

import (
	"html"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/webx-top/reverseproxy"
)

var (
	errorTemplate      *template.Template
	regexMemAddrSuffix = regexp.MustCompile(` [(+]0x[a-z0-9]+[)]?$`)
	regexIP4Prefix     = regexp.MustCompile(`.+\d+\.\d+.\d+.\d+\:\d+\: `)
)

func init() {
	templatePath := filepath.Join(SelfDir(), "page.html")

	_, err := os.Stat(templatePath)
	if err == nil {
		errorTemplate, err = template.ParseFiles(templatePath)
	} else {
		errorTemplate = template.New(`defaultPage`)
		_, err = errorTemplate.Parse(defaultPageHTML)
	}

	if err != nil {
		panic(err)
	}
}

func RenderError(ctx reverseproxy.Context, app *App, message string) {
	info := ErrorInfo{Title: "Error", Message: template.HTML(message)}
	info.Prepare()

	renderPage(ctx, info)
}

func RenderBuildError(ctx reverseproxy.Context, app *App, message string) {
	info := ErrorInfo{Title: "Build Error", Message: template.HTML(message)}
	info.Prepare()

	renderPage(ctx, info)
}

const SnippetLineNumbers = 13

func RenderAppError(ctx reverseproxy.Context, app *App, errMessage string) {
	info := ErrorInfo{Title: "Application Error"}
	message, trace, appIndex := extractAppErrorInfo(errMessage)

	// from: 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Validation Error
	//   to: Validation Error
	message[0] = regexIP4Prefix.ReplaceAllString(message[0], "")
	if !strings.Contains(message[0], "runtime error") {
		message[0] = "panic: " + message[0]
	}

	info.Message = template.HTML(strings.Join(message, "\n"))
	info.Trace = trace
	info.ShowTrace = true

	// from: test/server1.go:16 (0x211e)
	//	 to: [test/server1.go, 16]
	if appIndex >= 0 && appIndex < len(trace) {
		tr := trace[appIndex]
		info.SnippetPath = tr.File
		var err error
		info.Snippet, err = extractAppSnippet(tr.File, tr.Line)
		info.ShowSnippet = err == nil
	}

	info.Prepare()
	renderPage(ctx, info)
}

func renderPage(ctx reverseproxy.Context, info ErrorInfo) {
	ctx.SetHeader(`Content-Type`, `text/html;charset=utf-8`)
	err := errorTemplate.Execute(ctx.ResponseWriter(), info)
	if err != nil {
		panic(err)
	}
}

// Example input
// 2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Panic !!
// /usr/local/Cellar/go/1.0.3/src/pkg/net/http/server.go:589 (0x31ed9)
// _func_004: buf.Write(debug.Stack())
// /usr/local/Cellar/go/1.0.3/src/pkg/runtime/proc.c:1443 (0x10b83)
// panic: reflect·call(d->fn, d->args, d->siz);
// /Users/user/tower/test/server1.go:16 (0x211e)
// Panic: panic(errors.New("Panic !!"))

// Example output
// message:
//	[2013/02/12 18:24:15 http: panic serving 127.0.0.1:54114: Panic !!]
// trace:
//  [
//	 [test/server1.go:16 (0x211e), Panic: panic(errors.New("Panic !!"))]
//	]
func extractAppErrorInfo(errMessage string) (message []string, trace []Trace, appIndex int) {
	// from: /Users/user/tower/test/server1.go:16 (0x211e)
	// 		   Panic: panic(errors.New("Panic !!"))
	//   to: <n>//Users/user/tower/test/server1.go:16 (0x211e)<n>Panic: panic(errors.New("Panic !!"))
	errMessage = strings.Replace(errMessage, "\n\t", `<nt>`, -1)
	errMessage = strings.Replace(errMessage, "\n", `<n>`, -1)

	wd, _ := os.Getwd()
	wd = filepath.ToSlash(wd) + "/"
	appIndex = -1
	for _, line := range strings.Split(errMessage, `<n>`) {
		if len(line) == 0 { //另一个Goroutine开始
			continue
		}
		lines := strings.Split(line, `<nt>`)
		if !strings.HasSuffix(lines[0], `:`) && len(message) == 0 {
			message = lines
		}
		if len(lines) < 2 {
			continue
		}

		t := Trace{Func: lines[0], File: lines[1]}
		if strings.Index(t.File, wd) != -1 {
			if appIndex == -1 {
				appIndex = len(trace)
			}
			t.AppFile = true
		}
		t.File = strings.Replace(t.File, wd, "", 1)
		// from: /Users/user/tower/test/server1.go:16 (0x211e)
		// or from:  /Users/user/tower/test/server1.go:16 +0x211e
		//   to: /Users/user/tower/test/server1.go:16
		t.File = regexMemAddrSuffix.ReplaceAllString(t.File, "")
		t.File = strings.TrimSpace(t.File)
		if p := strings.LastIndex(t.File, `:`); p > 0 {
			t.Line, _ = strconv.Atoi(t.File[p+1:])
			t.File = t.File[0:p]
		}
		trace = append(trace, t)
	}
	return
}

func extractAppSnippet(appFile string, curLineNum int) (snippet []Snippet, err error) {
	content, err := ioutil.ReadFile(appFile)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	for lineNum := curLineNum - SnippetLineNumbers/2; lineNum <= curLineNum+SnippetLineNumbers/2; lineNum++ {
		if len(lines) >= lineNum {
			c := html.EscapeString(lines[lineNum-1])
			c = strings.Replace(c, "\t", "&nbsp;&nbsp;&nbsp;&nbsp;", -1)
			c = strings.Replace(c, " ", "&nbsp;", -1)
			snippet = append(snippet, Snippet{lineNum, template.HTML(c), lineNum == curLineNum})
		}
	}
	return
}

type ErrorInfo struct {
	Title   string
	Time    string
	Message template.HTML

	Trace     []Trace
	ShowTrace bool

	SnippetPath string
	Snippet     []Snippet
	ShowSnippet bool
}

type Snippet struct {
	Number  int
	Code    template.HTML
	Current bool
}

type Trace struct {
	Line    int
	File    string
	Func    string
	AppFile bool
}

func (this *ErrorInfo) Prepare() {
	this.TrimMessage()
	this.Time = time.Now().Format("15:04:05")
}

func (this *ErrorInfo) TrimMessage() {
	html := strings.Join(strings.Split(string(this.Message), "\n"), "<br/>")
	this.Message = template.HTML(html)
}
