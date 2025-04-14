package json

import "github.com/admpub/xencoding/filter"

func (e *field) FieldName() string {
	return e.name
}

func OptionFilter(f filter.Filter) Option {
	return func(o *encOpts) {
		o.filter.SetFilter(f)
	}
}

func OptionSelector(f filter.Selector) Option {
	return func(o *encOpts) {
		o.filter.SetSelector(f)
	}
}

func OptionEscapeHTML(escapeHTML bool) Option {
	return func(o *encOpts) {
		o.escapeHTML = escapeHTML
	}
}

func OptionIndent(prefix, indent string) Option {
	return func(p *encOpts) {
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

type Option func(*encOpts)

func MarshalWithOption(v any, opts ...Option) ([]byte, error) {
	e := newEncodeState()
	defer encodeStatePool.Put(e)

	option := encOpts{escapeHTML: true}
	for _, opt := range opts {
		opt(&option)
	}
	err := e.marshal(v, option)
	if err != nil {
		return nil, err
	}
	if len(option.prefix) > 0 || len(option.indent) > 0 {
		return JSONIndent(e.Bytes(), option.prefix, option.indent)
	}
	buf := append([]byte(nil), e.Bytes()...)
	return buf, nil
}

func JSONIndent(b []byte, prefix, indent string) ([]byte, error) {
	b2 := make([]byte, 0, indentGrowthFactor*len(b))
	return appendIndent(b2, b, prefix, indent)
}
