package common

type ContextKey int

const (
	DBHandle ContextKey = iota
	BuckContext
)

func (ck ContextKey) String() string {
	switch ck {
	case DBHandle:
		return "database handle"
	case BuckContext:
		return "buck context"
	}
	return "invalid context type"
}
