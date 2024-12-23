package main

import "fmt"

func invalidNumArgs(fn string, nx int, tx string) string {
	return fmt.Sprintf("missing argument #%d to '%s' (%s expected)", nx, fn, luautype[tx])
}

func invalidArgType(i int, fn string, tx, tg string) string {
	return fmt.Sprintf("invalid argument #%d to '%s' (%s expected, got %s)", i, fn, luautype[tx], luautype[tg])
}

type Args struct {
	args []any
	name string
	co   *Coroutine
	pos  int
}

type (
	Ret  any
	Rets []any
)

func getArg[T any](args *Args, optionalValue []T) T {
	var possibleArg any

	if args.pos >= len(args.args) {
		if len(optionalValue) == 0 {
			panic(invalidNumArgs(args.name, args.pos, typeOf(args.args[args.pos-1])))
		}
		possibleArg = optionalValue[0]
	} else {
		possibleArg = args.args[args.pos]
	}

	args.pos++
	arg, ok := possibleArg.(T)
	if !ok {
		panic(invalidArgType(args.pos, args.name, typeOf(arg), typeOf(possibleArg)))
	}
	return arg
}

func (a *Args) CheckNextArg() {
	if a.pos >= len(a.args) {
		panic(invalidNumArgs(a.name, a.pos, typeOf(a.args[a.pos-1])))
	}
}

func (a *Args) GetBool(optionalValue ...bool) bool {
	return getArg(a, optionalValue)
}

func (a *Args) GetNumber(optionalValue ...float64) float64 {
	return getArg(a, optionalValue)
}

func (a *Args) GetString(optionalValue ...string) string {
	return getArg(a, optionalValue)
}

func (a *Args) GetTable(optionalValue ...*Table) *Table {
	return getArg(a, optionalValue)
}

func (a *Args) GetFunction(optionalValue ...*Function) *Function {
	return getArg(a, optionalValue)
}

func (a *Args) GetCoroutine(optionalValue ...*Coroutine) *Coroutine {
	return getArg(a, optionalValue)
}

func (a *Args) GetBuffer(optionalValue ...*Buffer) *Buffer {
	return getArg(a, optionalValue)
}

func (a *Args) GetAny(optionalValue ...any) any {
	return getArg(a, optionalValue)
}

// Reflection don't scale
func MakeFn(name string, fn func(args Args) Rets) [2]any {
	fn2 := Function(func(co *Coroutine, vargs ...any) []any {
		return fn(Args{vargs, name, co, 0})
	})
	return [2]any{name, &fn2}
}

func MakeFn1(name string, fn func(args Args) Ret) [2]any {
	fn2 := Function(func(co *Coroutine, vargs ...any) []any {
		return []any{fn(Args{vargs, name, co, 0})}
	})
	return [2]any{name, &fn2}
}

func MakeFn0(name string, fn func(args Args)) [2]any {
	fn2 := Function(func(co *Coroutine, vargs ...any) []any {
		fn(Args{vargs, name, co, 0})
		return []any{}
	})
	return [2]any{name, &fn2}
}
