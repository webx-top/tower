package filter

import (
	"reflect"
	"strings"
)

type Filter interface {
	Filter(string, reflect.Value) bool
}

type Selector interface {
	Select(string, reflect.Value) bool
}

type excludeFilters map[string]struct{}

func (f excludeFilters) Add(names ...string) excludeFilters {
	for _, name := range names {
		f[name] = struct{}{}
	}
	return f
}

func (f excludeFilters) Filter(name string, v reflect.Value) bool {
	_, ok := f[name]
	return ok
}

type includeFilters map[string]struct{}

func (f includeFilters) Add(names ...string) includeFilters {
	for _, name := range names {
		f[name] = struct{}{}
	}
	return f
}

func (f includeFilters) Select(name string, v reflect.Value) bool {
	_, ok := f[name]
	if ok {
		return ok
	}
	parts := strings.Split(name, `.`)
	if len(parts) == 1 {
		_, ok = f[name+`.*`]
		return ok
	}
	for index := range parts[1:] {
		prefix := strings.Join(parts[:index+1], `.`) + `.*`
		_, ok = f[prefix]
		if ok {
			return ok
		}
	}
	return ok
}

func Include(names ...string) Selector {
	if len(names) == 0 {
		return nil
	}
	f := make(includeFilters)
	return f.Add(names...)
}

func Exclude(names ...string) Filter {
	if len(names) == 0 {
		return nil
	}
	f := make(excludeFilters)
	return f.Add(names...)
}
