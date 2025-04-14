package xml

import (
	"bytes"

	"github.com/admpub/xencoding/filter"
)

func (f *fieldInfo) FieldName() string {
	return f.name
}

func OptionFilter(f filter.Filter) Option {
	return func(p *printer) {
		p.filter.SetFilter(f)
	}
}

func OptionSelector(f filter.Selector) Option {
	return func(p *printer) {
		p.filter.SetSelector(f)
	}
}

func OptionIndent(prefix, indent string) Option {
	return func(p *printer) {
		p.prefix = prefix
		p.indent = indent
	}
}

func MarshalFilter(v any, f filter.Filter) ([]byte, error) {
	return MarshalWithOption(v, OptionFilter(f))
}

func MarshalSelector(v any, f filter.Selector) ([]byte, error) {
	return MarshalWithOption(v, OptionSelector(f))
}

type Option func(*printer)

func MarshalWithOption(v any, opts ...Option) ([]byte, error) {
	var b bytes.Buffer
	enc := NewEncoder(&b)
	for _, opt := range opts {
		opt(&enc.p)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
