package gen

import "fmt"

// Error is what Load, Scan and Prefix report. Every rejection the parser makes
// surfaces as one of these, with the same message the rr CLI prints.
type Error struct {
	Msg string
}

func (e *Error) Error() string { return e.Msg }

// fatalf aborts the current Load. The parser is a deep recursive walk with a
// rejection in nearly every branch, so failures unwind by panic and the
// entry points turn them back into an error.
func fatalf(format string, args ...any) {
	panic(&Error{Msg: fmt.Sprintf(format, args...)})
}

// catch converts a fatalf panic into err. Anything else keeps unwinding.
func catch(err *error) {
	r := recover()
	if r == nil {
		return
	}
	e, ok := r.(*Error)
	if !ok {
		panic(r)
	}
	*err = e
}
