package filter

import "reflect"

type Config struct {
	filter   Filter
	selector Selector
	path     string
}

func (c *Config) SetFilter(f Filter) {
	c.filter = f
}

func (c *Config) SetSelector(f Selector) {
	c.selector = f
}

func (c *Config) Reset() {
	c.path = ``
}

type FieldName interface {
	FieldName() string
}

func (c *Config) MakeFilters() (reset func(), filters []func(f FieldName, v reflect.Value) bool) {
	if c.selector != nil || c.filter != nil {
		path := c.path
		var pa func(f FieldName) string
		if len(path) > 0 {
			pa = func(f FieldName) string {
				return path + "." + f.FieldName()
			}
		} else {
			pa = func(f FieldName) string {
				return f.FieldName()
			}
			reset = c.Reset
		}
		if c.selector != nil {
			filters = append(filters, func(f FieldName, v reflect.Value) bool {
				c.path = pa(f)
				//println(`selector:`,c.path)
				return !c.selector.Select(c.path, v)
			})
		}
		if c.filter != nil {
			filters = append(filters, func(f FieldName, v reflect.Value) bool {
				c.path = pa(f)
				//println(`filter:`,c.path)
				return c.filter.Filter(c.path, v)
			})
		}
	}
	return
}

func (c *Config) CallFilter(filters []func(f FieldName, v reflect.Value) bool, f FieldName, v reflect.Value) bool {
	for _, filter := range filters {
		if filter(f, v) {
			return true
		}
	}
	return false
}
