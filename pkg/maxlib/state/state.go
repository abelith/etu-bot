package state

const ClearState = "clear"

type State string

type Context struct {
	id    int
	state State
	m     map[string]string
}

func New(id int) *Context {
	return &Context{
		id: id,
		m:  make(map[string]string),
	}
}

func (sc *Context) SetState(state State) {
	sc.state = state
}

func (sc *Context) GetState() State {
	return sc.state
}

func (sc *Context) Set(key, val string) {
	sc.m[key] = val
}

func (sc *Context) Get(key string) (string, bool) {
	val, ok := sc.m[key]
	return val, ok
}

func (sc *Context) Clear() {
	sc.state = ClearState
}
