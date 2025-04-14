package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/admpub/log"
	"github.com/webx-top/com"
)

const (
	HttpPanicMessage = "http: panic serving"
)

var (
	BinPrefix    = "tower-app-"
	AppBin       = ""
	findPackage  = regexp.MustCompile(`:[\s]*cannot find package "([^"]+)" in any of:`)
	findPackage2 = regexp.MustCompile(`:[\s]*unrecognized import path "([^"]+)"[\s]*\(`)
	movePackage  = regexp.MustCompile(`can't load package: package [^:]+: code in directory ([^\s]+) expects import "([^"]+)"`)
)

type App struct {
	OfflineMode         bool
	Cmds                map[string]*exec.Cmd
	RunParams           []string
	BuildParams         []string
	MainFile            string
	Port                string
	Ports               map[string]int64
	BuildDir            string
	Name                string
	Root                string
	keyPressListened    bool
	LastError           string
	PortParamName       string //端口参数名称(用于指定应用程序监听的端口，例如：webx.exe -p 8080，这里的-p就是端口参数名)
	SwitchToNewPort     bool
	DisabledBuild       bool
	BeforeBuildGenerate bool
	BuildStart          *sync.Once
	AppRestart          *sync.Once
	DisabledLogRequest  bool
	PkgMirrors          map[string]string
	Env                 []string

	portBinFiles map[string]string
	buildErr     error
	startErr     error
	restartErr   error
	_goVersion   string
	ctx          context.Context
}

type StderrCapturer struct {
	app *App
}

func (a StderrCapturer) Write(p []byte) (n int, err error) {
	s := string(p)
	httpError := strings.Contains(s, HttpPanicMessage)

	if httpError {
		a.app.LastError = s
		os.Stdout.Write([]byte("----------- Application Error -----------\n"))
		n, err = os.Stdout.Write(p)
		os.Stdout.Write([]byte("-----------------------------------------\n"))
	} else {
		n, err = os.Stdout.Write(p)
	}
	return
}

func NewApp(ctx context.Context, mainFile, port, buildDir, portParamName string) (app App) {
	app.ctx = ctx
	app.Cmds = make(map[string]*exec.Cmd)
	goPath := os.Getenv(`GOPATH`)
	if len(goPath) > 0 && !strings.HasSuffix(mainFile, `.go`) {
		var err error
		goPath, err = filepath.Abs(goPath)
		if err != nil {
			panic(err.Error())
		}
		app.MainFile = strings.TrimPrefix(mainFile, string(append([]byte(filepath.Join(goPath, `src`)), filepath.Separator)))
	} else {
		app.MainFile = mainFile
	}
	app.BuildDir = buildDir
	app.PortParamName = portParamName
	app.ParseMutiPort(port)
	app.Port = app.UseRandPort()
	wd, _ := os.Getwd()
	app.Name = filepath.Base(wd)
	app.Root = filepath.Dir(mainFile)
	app.BuildStart = &sync.Once{}
	app.AppRestart = &sync.Once{}
	app.portBinFiles = make(map[string]string)
	app.PkgMirrors = make(map[string]string)
	app.RunParams = []string{}
	app.BuildParams = []string{}
	return
}

func (a *App) DisabledVisitPort() bool {
	return len(a.Port) == 0 || len(a.PortParamName) == 0
}

func (a *App) ParseMutiPort(port string) {
	p := strings.Split(port, `,`)
	a.Ports = make(map[string]int64)
	for _, v := range p {
		r := strings.Split(v, `-`)
		if len(r) > 1 {
			i, _ := strconv.Atoi(r[0])
			j, _ := strconv.Atoi(r[1])
			for ; i <= j; i++ {
				port := fmt.Sprintf("%v", i)
				a.Ports[port] = 0
			}
		} else {
			a.Ports[r[0]] = 0
		}
	}
}

func (a *App) SupportMutiPort() bool {
	return a.Ports != nil && len(a.Ports) > 1 && a.PortParamName != ``
}

func (a *App) UseRandPort() string {
	var lastRunTime []int64
	lastRunPorts := make(map[int64]string, 0)
	for port, runningTime := range a.Ports {
		if runningTime == 0 || a.IsRunning(port) == false || isFreePort(port) {
			return port
		}
		lastRunTime = append(lastRunTime, runningTime)
		lastRunPorts[runningTime] = port
	}
	quickSort(lastRunTime, 0, len(lastRunTime)-1)
	for _, runningTime := range lastRunTime {
		return lastRunPorts[runningTime]
	}
	return a.Port
}

func (a *App) Start(ctx context.Context, build bool, args ...string) error {
	a.BuildStart.Do(func() {
		if build {
			a.buildErr = a.Build()
			if a.buildErr != nil {
				log.Error("== Fail to build " + a.Name + ": " + a.buildErr.Error())
				a.startErr = a.buildErr
				a.BuildStart = &sync.Once{}
				return
			}
		}
		port := a.Port
		if len(args) > 0 {
			port = args[0]
		}
		a.startErr = a.Run(port)
		if a.startErr != nil {
			a.startErr = errors.New("== Fail to run " + a.Name + ": " + a.startErr.Error())
			a.BuildStart = &sync.Once{}
			return
		}
		a.RestartOnReturn(ctx)
		a.BuildStart = &sync.Once{}
	})

	return a.startErr
}

func (a *App) Restart(ctx context.Context) error {
	a.AppRestart.Do(func() {
		log.Warn(`== Restart the application.`)
		a.Clean()
		a.Stop(a.Port)
		a.restartErr = a.Start(ctx, true)
		a.AppRestart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return a.restartErr
}

func (a *App) BinFile(args ...string) (f string) {
	binFileName := AppBin
	if len(args) > 0 {
		binFileName = args[0]
	}
	if len(a.BuildDir) > 0 {
		f = filepath.Join(a.BuildDir, binFileName)
	} else {
		f = binFileName
	}
	if runtime.GOOS == "windows" {
		f += ".exe"
	}
	return
}

func (a *App) Stop(port string, args ...string) {
	if !a.IsRunning(port) {
		return
	}
	log.Info("== Stopping " + a.Name)
	cmd := a.GetCmd(port)
	if cmd == nil || cmd.Process == nil {
		return
	}
	err := cmd.Process.Kill()
	if err != nil {
		log.Error(err)
	}
	cmd = nil
	if port == a.Port && a.DisabledBuild {
		return
	}
	bin := a.BinFile(args...)
	err = os.Remove(bin)
	if err == nil {
		a.Ports[port] = 0
		return
	}
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			err = os.Remove(bin)
			if err != nil {
				if os.IsNotExist(err) {
					a.Ports[port] = 0
					return
				}
				log.Error(err)
			} else {
				log.Info(`== Remove ` + bin + `: Success.`)
				a.Ports[port] = 0
				return
			}
		}
	}()
}

func (a *App) Clean(excludePorts ...string) {
	excludePort := a.Port
	if len(excludePorts) > 0 {
		excludePort = excludePorts[0]
	}
	for port, cmd := range a.Cmds {
		if port == excludePort || !CmdIsRunning(cmd) {
			continue
		}
		if cmd == nil || cmd.Process == nil {
			continue
		}
		log.Info("== Stopping app at port: " + port)
		err := cmd.Process.Kill()
		if err != nil {
			log.Error(err)
		}
		cmd = nil
		if bin, ok := a.portBinFiles[port]; ok && bin != "" {
			err := os.Remove(bin)
			if err == nil {
				a.Ports[port] = 0
				continue
			}
			go func(port string) {
				for i := 0; i < 10; i++ {
					time.Sleep(time.Second * time.Duration(i+1))
					err = os.Remove(bin)
					if err != nil {
						log.Error(err)
					} else {
						log.Info(`== Remove ` + bin + `: Success.`)
						a.Ports[port] = 0
						return
					}
				}
			}(port)
		}
	}
}

func (a *App) GetCmd(args ...string) (cmd *exec.Cmd) {
	var port string
	if len(args) > 0 {
		port = args[0]
	} else {
		port = a.Port
	}
	cmd = a.Cmds[port]
	return
}

func (a *App) SetCmd(port string, cmd *exec.Cmd) {
	a.Cmds[port] = cmd
}

func (a *App) Run(port string) (err error) {
	bin := a.BinFile()
	_, err = os.Stat(bin)
	if err != nil {
		return
	}
	ableSwitch := true
	disabledVisitPort := a.DisabledVisitPort()
	if !disabledVisitPort {
		log.Info("== Running at port " + port + ": " + a.Name)
		ableSwitch = a.Port != port
		a.Port = port //记录被使用的端口，避免下次使用
	} else {
		log.Info("== Running " + a.Name)
		cmd := a.GetCmd(port)
		bin := a.portBinFiles[port]
		if cmd != nil && len(bin) > 0 {
			defer func() {
				if !CmdIsRunning(cmd) {
					return
				}
				if cmd == nil || cmd.Process == nil {
					return
				}
				log.Info("== Stopping app: " + bin)
				err := cmd.Process.Kill()
				if err != nil {
					log.Error(err)
				}
				err = os.Remove(bin)
				if err == nil {
					return
				}

				go func() {
					for i := 0; i < 10; i++ {
						time.Sleep(time.Second * time.Duration(i+1))
						err = os.Remove(bin)
						if err != nil {
							if os.IsNotExist(err) {
								return
							}
							log.Error(err)
						} else {
							log.Info(`== Remove ` + bin + `: Success.`)
							return
						}
					}
				}()
			}()
		}
	}

	var cmd *exec.Cmd
	a.portBinFiles[port] = bin
	a.Ports[port] = time.Now().Unix()
	params := []string{}
	if !disabledVisitPort && a.SupportMutiPort() {
		params = append(params, com.ParseArgs(a.PortParamName)...)
		params = append(params, port)
	}
	params = append(params, a.RunParams...)
	cmd = exec.CommandContext(a.ctx, bin, params...)
	a.SetCmd(a.Port, cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = StderrCapturer{a}
	cmd.Env = append(os.Environ(), a.Env...)
	var hasError bool
	go func() {
		err := cmd.Run()
		if err != nil {
			if a.Port == port {
				log.Error(`== cmd.Run Error:`, err)
			}
			hasError = true
		}
	}()
	if !disabledVisitPort {
		err = dialAddress("127.0.0.1:"+a.Port, 60, func() bool {
			return !hasError
		})
	}
	if err == nil && ableSwitch {
		a.SwitchToNewPort = true
		if a.OfflineMode {
			a.Clean()
		}
	}
	return
}

func (a *App) fetchPkg(matches [][]string, isRetry bool, args ...string) bool {
	alldl := true
	currt := filepath.ToSlash(a.BuildDir)
	for _, match := range matches {
		pkg := match[1]
		if strings.Contains(currt, pkg) {
			continue
		}
		moveTo := pkg
		for rule, rep := range a.PkgMirrors {
			re, err := regexp.Compile(rule)
			if err != nil {
				log.Error(err)
				continue
			}
			pkg = re.ReplaceAllString(pkg, rep)
		}
		fromDir := pkg
		if len(pkg) > 10 {
			switch pkg[0:10] {
			case `golang.org`:
				pkg = strings.TrimPrefix(pkg, `golang.org/x/`)
				repertory := strings.SplitN(pkg, `/`, 2)[0]
				pkg = `github.com/golang/` + repertory
				moveTo = `golang.org/x/` + repertory
			case `github.com`:
				arr := strings.SplitN(pkg, `/`, 4)
				pkg = strings.Join(arr[0:3], `/`)
			}
		}
		cmdArgs := []string{`get`}
		cmdArgs = append(cmdArgs, args...)
		var hasVerb bool
		for _, _arg := range cmdArgs {
			if _arg == `-v` {
				hasVerb = true
				break
			}
		}
		if !hasVerb {
			cmdArgs = append(cmdArgs, `-v`)
		}
		cmdArgs = append(cmdArgs, pkg)
		cmd := exec.CommandContext(a.ctx, "go", cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Env = append(os.Environ(), a.Env...)
		err := cmd.Run()
		if err != nil && !isRetry {
			matches2 := findPackage2.FindAllStringSubmatch(err.Error(), -1)
			if len(matches2) > 0 {
				if a.fetchPkg(matches2, true, args...) {
					err = nil
				}
			}
		}
		if err != nil {
			log.Error(err)
			match := movePackage.FindStringSubmatch(err.Error())
			if len(match) > 0 {
				fromPath := match[1]
				moveTo := match[2]
				goPath := os.Getenv(`GOPATH`)
				toPath := filepath.Join(goPath, `src`, moveTo)
				err = os.MkdirAll(toPath, os.ModePerm)
				if err != nil {
					log.Error(err, `: `, toPath)
					alldl = false
					continue
				}
				err = com.CopyDir(fromPath, toPath)
				if err != nil {
					log.Error(err, `: `, fromPath, ` => `, toPath)
					alldl = false
					continue
				}
			} else {
				alldl = false
			}
		}
		if moveTo != fromDir {
			goPath := os.Getenv(`GOPATH`)
			fromPath := filepath.Join(goPath, `src`, fromDir)
			if !com.IsDir(fromPath) {
				continue
			}
			toPath := filepath.Join(goPath, `src`, moveTo)
			err = os.MkdirAll(toPath, os.ModePerm)
			if err != nil {
				log.Error(err, `: `, toPath)
				alldl = false
				continue
			}
			err = com.CopyDir(fromPath, toPath)
			if err != nil {
				log.Error(err, `: `, fromPath, ` => `, toPath)
				alldl = false
				continue
			}
		}
	}
	return alldl
}

func (a *App) goVersion() (string, error) {
	if len(a._goVersion) > 0 {
		return a._goVersion, nil
	}
	b, err := exec.CommandContext(a.ctx, "go", "version").CombinedOutput()
	if err != nil {
		return "", err
	}
	v := string(b)
	v = strings.TrimPrefix(v, `go version go`)
	v = strings.SplitN(v, ` `, 2)[0]
	a._goVersion = v
	return v, nil
}

func (a *App) Build() (err error) {
	if a.DisabledBuild {
		return nil
	}
	log.Info("== Building " + a.Name)
	AppBin = BinPrefix + strconv.FormatInt(time.Now().Unix(), 10)
	build := func() (string, error) {
		if a.BeforeBuildGenerate {
			cmd := exec.CommandContext(a.ctx, "go", "generate")
			cmd.Run()
		}
		args := []string{"build"}
		args = append(args, a.BuildParams...)
		args = append(args, []string{"-o", a.BinFile(), a.MainFile}...)
		cmd := exec.CommandContext(a.ctx, "go", args...)
		var b bytes.Buffer
		cmd.Stderr = &b
		cmd.Stdout = os.Stdout
		cmd.Env = append(os.Environ(), a.Env...)
		err := cmd.Run()
		out := b.String()
		return out, err
	}
	out, err := build()
	var lastOut string
	for i := 0; err != nil && len(out) > 0 && lastOut != out && i < 10; i++ {
		matches := findPackage.FindAllStringSubmatch(out, -1)
		if len(matches) > 0 {
			if a.fetchPkg(matches, false) {
				lastOut = out
				out, err = build()
				continue
			}
		}
		break
	}
	if err != nil && len(out) > 0 {
		msg := strings.Replace(out, "# command-line-arguments\n", "", 1)
		log.Errorf("----------- Build Error -----------\n%s-----------------------------------", msg)
		return errors.New(err.Error() + `: ` + msg)
	}
	log.Info("== Build completed.")
	return nil
}

func (a *App) IsRunning(args ...string) bool {
	return CmdIsRunning(a.GetCmd(args...))
}

func CmdIsRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState == nil
}

func CmdIsQuit(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState != nil
}

func (a *App) IsQuit(args ...string) bool {
	return CmdIsQuit(a.GetCmd(args...))
}

func (a *App) RestartOnReturn(ctx context.Context) {
	if a.keyPressListened {
		return
	}
	a.keyPressListened = true

	// Listen to keypress of "return" and restart the app automatically
	go func() {
		in := bufio.NewReader(os.Stdin)
		for {
			input, err := in.ReadString('\n')
			if err != nil && err != io.EOF {
				log.Error(`watchingSignal:`, err.Error())
				return
			}
			if input == "\n" {
				a.Restart(ctx)
			}
		}
	}()

	// Listen to "^C" signal and stop the app properly
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		defer func() {
			fmt.Println("")
			a.Stop(a.Port)
			os.Exit(0)
		}()
		for {
			select {
			case <-sig: // wait for the "^C" signal
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}
