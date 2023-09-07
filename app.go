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
	KeyPress            bool
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
}

type StderrCapturer struct {
	app *App
}

func (this StderrCapturer) Write(p []byte) (n int, err error) {
	s := string(p)
	httpError := strings.Contains(s, HttpPanicMessage)

	if httpError {
		this.app.LastError = s
		os.Stdout.Write([]byte("----------- Application Error -----------\n"))
		n, err = os.Stdout.Write(p)
		os.Stdout.Write([]byte("-----------------------------------------\n"))
	} else {
		n, err = os.Stdout.Write(p)
	}
	return
}

func NewApp(mainFile, port, buildDir, portParamName string) (app App) {
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

func (this *App) DisabledVisitPort() bool {
	return len(this.Port) == 0 || len(this.PortParamName) == 0
}

func (this *App) ParseMutiPort(port string) {
	p := strings.Split(port, `,`)
	this.Ports = make(map[string]int64)
	for _, v := range p {
		r := strings.Split(v, `-`)
		if len(r) > 1 {
			i, _ := strconv.Atoi(r[0])
			j, _ := strconv.Atoi(r[1])
			for ; i <= j; i++ {
				port := fmt.Sprintf("%v", i)
				this.Ports[port] = 0
			}
		} else {
			this.Ports[r[0]] = 0
		}
	}
}

func (this *App) SupportMutiPort() bool {
	return this.Ports != nil && len(this.Ports) > 1 && this.PortParamName != ``
}

func (this *App) UseRandPort() string {
	var lastRunTime []int64
	lastRunPorts := make(map[int64]string, 0)
	for port, runningTime := range this.Ports {
		if runningTime == 0 || this.IsRunning(port) == false || isFreePort(port) {
			return port
		}
		lastRunTime = append(lastRunTime, runningTime)
		lastRunPorts[runningTime] = port
	}
	quickSort(lastRunTime, 0, len(lastRunTime)-1)
	for _, runningTime := range lastRunTime {
		return lastRunPorts[runningTime]
	}
	return this.Port
}

func (this *App) Start(ctx context.Context, build bool, args ...string) error {
	this.BuildStart.Do(func() {
		if build {
			this.buildErr = this.Build()
			if this.buildErr != nil {
				log.Error("== Fail to build " + this.Name + ": " + this.buildErr.Error())
				this.startErr = this.buildErr
				this.BuildStart = &sync.Once{}
				return
			}
		}
		port := this.Port
		if len(args) > 0 {
			port = args[0]
		}
		this.startErr = this.Run(port)
		if this.startErr != nil {
			this.startErr = errors.New("== Fail to run " + this.Name + ": " + this.startErr.Error())
			this.BuildStart = &sync.Once{}
			return
		}
		this.RestartOnReturn(ctx)
		this.BuildStart = &sync.Once{}
	})

	return this.startErr
}

func (this *App) Restart(ctx context.Context) error {
	this.AppRestart.Do(func() {
		log.Warn(`== Restart the application.`)
		this.Clean()
		this.Stop(this.Port)
		this.restartErr = this.Start(ctx, true)
		this.AppRestart = &sync.Once{} // Assign new Once to allow calling Start again.
	})

	return this.restartErr
}

func (this *App) BinFile(args ...string) (f string) {
	binFileName := AppBin
	if len(args) > 0 {
		binFileName = args[0]
	}
	if len(this.BuildDir) > 0 {
		f = filepath.Join(this.BuildDir, binFileName)
	} else {
		f = binFileName
	}
	if runtime.GOOS == "windows" {
		f += ".exe"
	}
	return
}

func (this *App) Stop(port string, args ...string) {
	if !this.IsRunning(port) {
		return
	}
	log.Info("== Stopping " + this.Name)
	cmd := this.GetCmd(port)
	err := cmd.Process.Kill()
	if err != nil {
		log.Error(err)
	}
	cmd = nil
	if port == this.Port && this.DisabledBuild {
		return
	}
	bin := this.BinFile(args...)
	err = os.Remove(bin)
	if err == nil {
		this.Ports[port] = 0
		return
	}
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Second)
			err = os.Remove(bin)
			if err != nil {
				if os.IsNotExist(err) {
					this.Ports[port] = 0
					return
				}
				log.Error(err)
			} else {
				log.Info(`== Remove ` + bin + `: Success.`)
				this.Ports[port] = 0
				return
			}
		}
	}()
}

func (this *App) Clean(excludePorts ...string) {
	excludePort := this.Port
	if len(excludePorts) > 0 {
		excludePort = excludePorts[0]
	}
	for port, cmd := range this.Cmds {
		if port == excludePort || !CmdIsRunning(cmd) {
			continue
		}
		log.Info("== Stopping app at port: " + port)
		err := cmd.Process.Kill()
		if err != nil {
			log.Error(err)
		}
		cmd = nil
		if bin, ok := this.portBinFiles[port]; ok && bin != "" {
			err := os.Remove(bin)
			if err == nil {
				this.Ports[port] = 0
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
						this.Ports[port] = 0
						return
					}
				}
			}(port)
		}
	}
}

func (this *App) GetCmd(args ...string) (cmd *exec.Cmd) {
	var port string
	if len(args) > 0 {
		port = args[0]
	} else {
		port = this.Port
	}
	cmd, _ = this.Cmds[port]
	return
}

func (this *App) SetCmd(port string, cmd *exec.Cmd) {
	this.Cmds[port] = cmd
}

func (this *App) Run(port string) (err error) {
	bin := this.BinFile()
	_, err = os.Stat(bin)
	if err != nil {
		return
	}
	ableSwitch := true
	disabledVisitPort := this.DisabledVisitPort()
	if !disabledVisitPort {
		log.Info("== Running at port " + port + ": " + this.Name)
		ableSwitch = this.Port != port
		this.Port = port //记录被使用的端口，避免下次使用
	} else {
		log.Info("== Running " + this.Name)
		cmd := this.GetCmd(port)
		bin := this.portBinFiles[port]
		if cmd != nil && len(bin) > 0 {
			defer func() {
				if !CmdIsRunning(cmd) {
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
	this.portBinFiles[port] = bin
	this.Ports[port] = time.Now().Unix()
	params := []string{}
	if !disabledVisitPort && this.SupportMutiPort() {
		params = append(params, this.PortParamName)
		params = append(params, port)
	}
	params = append(params, this.RunParams...)
	cmd = exec.Command(bin, params...)
	this.SetCmd(this.Port, cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = StderrCapturer{this}
	cmd.Env = append(os.Environ(), this.Env...)
	var hasError bool
	go func() {
		err := cmd.Run()
		if err != nil {
			if this.Port == port {
				log.Error(`== cmd.Run Error:`, err)
			}
			hasError = true
		}
	}()
	if !disabledVisitPort {
		err = dialAddress("127.0.0.1:"+this.Port, 60, func() bool {
			return !hasError
		})
	}
	if err == nil && ableSwitch {
		this.SwitchToNewPort = true
		if this.OfflineMode {
			this.Clean()
		}
	}
	return
}

func (this *App) fetchPkg(matches [][]string, isRetry bool, args ...string) bool {
	alldl := true
	currt := filepath.ToSlash(this.BuildDir)
	for _, match := range matches {
		pkg := match[1]
		if strings.Contains(currt, pkg) {
			continue
		}
		moveTo := pkg
		for rule, rep := range this.PkgMirrors {
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
		cmd := exec.Command("go", cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Env = append(os.Environ(), this.Env...)
		err := cmd.Run()
		if err != nil && !isRetry {
			matches2 := findPackage2.FindAllStringSubmatch(err.Error(), -1)
			if len(matches2) > 0 {
				if this.fetchPkg(matches2, true, args...) {
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

func (this *App) goVersion() (string, error) {
	if len(this._goVersion) > 0 {
		return this._goVersion, nil
	}
	b, err := exec.Command("go", "version").CombinedOutput()
	if err != nil {
		return "", err
	}
	v := string(b)
	v = strings.TrimPrefix(v, `go version go`)
	v = strings.SplitN(v, ` `, 2)[0]
	this._goVersion = v
	return v, nil
}

func (this *App) Build() (err error) {
	if this.DisabledBuild {
		return nil
	}
	log.Info("== Building " + this.Name)
	AppBin = BinPrefix + strconv.FormatInt(time.Now().Unix(), 10)
	build := func() (string, error) {
		if this.BeforeBuildGenerate {
			cmd := exec.Command("go", "generate")
			cmd.Run()
		}
		args := []string{"build"}
		args = append(args, this.BuildParams...)
		args = append(args, []string{"-o", this.BinFile(), this.MainFile}...)
		cmd := exec.Command("go", args...)
		var b bytes.Buffer
		cmd.Stderr = &b
		cmd.Stdout = os.Stdout
		cmd.Env = append(os.Environ(), this.Env...)
		err := cmd.Run()
		out := b.String()
		return out, err
	}
	out, err := build()
	var lastOut string
	for i := 0; err != nil && len(out) > 0 && lastOut != out && i < 10; i++ {
		matches := findPackage.FindAllStringSubmatch(out, -1)
		if len(matches) > 0 {
			if this.fetchPkg(matches, false) {
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

func (this *App) IsRunning(args ...string) bool {
	return CmdIsRunning(this.GetCmd(args...))
}

func CmdIsRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState == nil
}

func CmdIsQuit(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.ProcessState != nil
}

func (this *App) IsQuit(args ...string) bool {
	return CmdIsQuit(this.GetCmd(args...))
}

func (this *App) RestartOnReturn(ctx context.Context) {
	if this.KeyPress {
		return
	}
	this.KeyPress = true

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
				this.Restart(ctx)
			}
		}
	}()

	// Listen to "^C" signal and stop the app properly
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		defer func() {
			fmt.Println("")
			this.Stop(this.Port)
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
