package config

var Conf = &Config{
	App:   &App{},
	Proxy: &Proxy{},
	Admin: &Admin{},
	Watch: &Watch{},
}

type App struct {
	ExecFile      *string `json:"exec"` //非编译模式下有效
	MainFile      *string `json:"main"` //编译模式下有效
	Port          *string `json:"port"`
	PortParamName *string `json:"portParamName"`
	BuildDir      *string `json:"buildDir"`
	RunParams     *string `json:"params"`
}

func (a *App) Fixed() {
	if a.ExecFile == nil {
		s := ``
		a.ExecFile = &s
	}
	if a.MainFile == nil {
		s := ``
		a.MainFile = &s
	}
	if a.Port == nil {
		s := ``
		a.Port = &s
	}
	if a.PortParamName == nil {
		s := ``
		a.PortParamName = &s
	}
	if a.BuildDir == nil {
		s := ``
		a.BuildDir = &s
	}
	if a.RunParams == nil {
		s := ``
		a.RunParams = &s
	}
}

type Proxy struct {
	Port   *string `json:"port"`
	Engine *string `json:"engine"`
}

func (p *Proxy) Fixed() {
	if p.Engine == nil {
		s := ``
		p.Engine = &s
	}
	if p.Port == nil {
		s := ``
		p.Port = &s
	}
}

type Watch struct {
	FileExtension *string `json:"fileExtension"`
	OtherDir      *string `json:"otherDir"` //编译模式下有效
	IgnoredPath   *string `json:"ignoredPath"`
}

func (w *Watch) Fixed() {
	if w.FileExtension == nil {
		s := ``
		w.FileExtension = &s
	}
	if w.OtherDir == nil {
		s := ``
		w.OtherDir = &s
	}
	if w.IgnoredPath == nil {
		s := ``
		w.IgnoredPath = &s
	}
}

type Admin struct {
	Password *string `json:"password"`
	IPs      *string `json:"ips"`
}

func (a *Admin) Fixed() {
	if a.Password == nil {
		s := ``
		a.Password = &s
	}
	if a.IPs == nil {
		s := ``
		a.IPs = &s
	}
}

type Config struct {
	App        *App    `json:"app"`
	Proxy      *Proxy  `json:"proxy"`
	Admin      *Admin  `json:"admin"`
	Watch      *Watch  `json:"watch"`
	Verbose    *bool   `json:"verbose"`
	ConfigFile *string `json:"-"`
	LogLevel   *string `json:"logLevel"`
	LogRequest *bool   `json:"logRequest"`
	AutoClear  *bool   `json:"autoClear"`
	Offline    *bool   `json:"offline"`
}

func (c *Config) Fixed() {
	if c.App == nil {
		c.App = &App{}
	}
	c.App.Fixed()

	if c.Proxy == nil {
		c.Proxy = &Proxy{}
	}
	c.Proxy.Fixed()

	if c.Admin == nil {
		c.Admin = &Admin{}
	}
	c.Admin.Fixed()

	if c.Watch == nil {
		c.Watch = &Watch{}
	}
	c.Watch.Fixed()

	if c.ConfigFile == nil {
		s := ``
		c.ConfigFile = &s
	}
	if c.LogLevel == nil {
		s := ``
		c.LogLevel = &s
	}
	if c.Verbose == nil {
		s := false
		c.Verbose = &s
	}
	if c.LogRequest == nil {
		s := false
		c.LogRequest = &s
	}
	if c.AutoClear == nil {
		s := false
		c.AutoClear = &s
	}
	if c.Offline == nil {
		s := false
		c.Offline = &s
	}
}
