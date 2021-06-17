package param

type Transfer interface {
	Transform(value interface{}, row Store) interface{}
	Destination() string
}

func NewTransfers() *Transfers {
	return &Transfers{}
}

// Transfers {oldField:Transfer}
type Transfers map[string]Transfer

func (t *Transfers) Add(name string, transfer Transfer) *Transfers {
	(*t)[name] = transfer
	return t
}

func (t *Transfers) AddFunc(oldField string, fn func(value interface{}, row Store) interface{}, newField ...string) *Transfers {
	tr := NewTransform().SetFunc(fn)
	if len(newField) > 0 {
		tr.SetKey(newField[0])
	}
	(*t)[oldField] = tr
	return t
}

func (t *Transfers) Delete(names ...string) *Transfers {
	for _, name := range names {
		if _, ok := (*t)[name]; ok {
			delete(*t, name)
		}
	}
	return t
}

func (t *Transfers) AsMap() Transfers {
	return *t
}

func (t *Transfers) Transform(row Store) Store {
	return row.Transform(t.AsMap())
}

func NewTransform() *Transform {
	return &Transform{}
}

type Transform struct {
	Key  string                                         // new field
	Func func(value interface{}, row Store) interface{} `json:"-" xml:"-"`
}

func (t *Transform) Transform(value interface{}, row Store) interface{} {
	if t.Func == nil {
		return value
	}
	return t.Func(value, row)
}

func (t *Transform) Destination() string {
	return t.Key
}

func (t *Transform) Set(key string, fn func(value interface{}, row Store) interface{}) *Transform {
	t.SetKey(key)
	t.SetFunc(fn)
	return t
}

func (t *Transform) SetKey(key string) *Transform {
	t.Key = key
	return t
}

func (t *Transform) SetFunc(fn func(value interface{}, row Store) interface{}) *Transform {
	t.Func = fn
	return t
}
