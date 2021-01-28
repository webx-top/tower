package logger

var _ Logger = &Base{}

type (
	// Logger is the interface that declares Echo's logging system.
	Logger interface {
		Debug(...interface{})
		Debugf(string, ...interface{})

		Info(...interface{})
		Infof(string, ...interface{})

		Warn(...interface{})
		Warnf(string, ...interface{})

		Error(...interface{})
		Errorf(string, ...interface{})

		Fatal(...interface{})
		Fatalf(string, ...interface{})
	}

	LevelSetter interface {
		SetLevel(string)
	}

	Base struct {
	}
)

func (b *Base) Debug(...interface{}) {
}

func (b *Base) Debugf(string, ...interface{}) {
}

func (b *Base) Info(...interface{}) {
}

func (b *Base) Infof(string, ...interface{}) {
}

func (b *Base) Warn(...interface{}) {
}

func (b *Base) Warnf(string, ...interface{}) {
}

func (b *Base) Error(...interface{}) {
}

func (b *Base) Errorf(string, ...interface{}) {
}

func (b *Base) Fatal(...interface{}) {
}

func (b *Base) Fatalf(string, ...interface{}) {
}

func (b *Base) SetLevel(string) {
}
