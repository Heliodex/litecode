package litecode

import (
	"errors"
	"strings"
)

func table_clear(args Args) (r Rets, err error) {
	t := args.GetTable()
	if t.readonly {
		return nil, errors.New("attempt to modify a readonly table")
	}

	t.array = nil
	t.node = nil
	return
}

func table_clone(args Args) (r Rets, err error) {
	t := args.GetTable()

	var a2 *[]any
	var h2 *map[any]any

	if t.array != nil {
		a := make([]any, len(*t.array))
		copy(a, *t.array)
		a2 = &a
	}

	if t.node == nil {
		h := make(map[any]any, len(*t.node))
		for k, v := range *t.node {
			h[k] = v
		}
		h2 = &h
	}

	return Rets{&Table{
		array: a2,
		node:  h2,
	}}, nil
}

func table_concat(args Args) (r Rets, err error) {
	t := args.GetTable()
	sep := args.GetString("")
	i := args.GetNumber(1)
	j := args.GetNumber(float64(t.Len()))

	if i > j {
		return Rets{""}, nil
	}

	b := strings.Builder{}
	for ; i <= j; i++ {
		v, ok := t.Get(i).(string)
		if !ok {
			return nil, errors.New("attempt to concatenate non-string value")
		}

		b.WriteString(v)
		if i < j {
			b.WriteString(sep)
		}
	}

	return Rets{b.String()}, nil
}

func table_create(args Args) (r Rets, err error) {
	s := int(args.GetNumber())
	if s < 0 {
		return nil, errors.New("index out of range")
	}

	if len(args.List) == 1 {
		a := make([]any, 0, s)
		return Rets{&Table{
			array: &a,
		}}, nil
	}

	a := make([]any, s)
	value := args.GetAny()
	for i := range a {
		a[i] = value
	}
	return Rets{&Table{
		array: &a,
	}}, nil
}

func table_find(args Args) (r Rets, err error) {
	haystack := args.GetTable()
	needle := args.GetAny()
	init := args.GetNumber(1)
	if init < 1 {
		return nil, errors.New("index out of range")
	}

	if haystack.array != nil {
		for i := int(init); i < len(*haystack.array); i++ {
			v := (*haystack.array)[i-1]
			if needle == v {
				return Rets{float64(i)}, nil
			}
		}
	}
	if haystack.node != nil {
		for k, v := range *haystack.node {
			if needle == v {
				return Rets{k}, nil
			}
		}
	}
	return Rets{nil}, nil
}

func table_freeze(args Args) (r Rets, err error) {
	t := args.GetTable()

	t.readonly = true
	return Rets{t}, nil
}

func bumpelements(t *Table, start int) {
	// fmt.Println("BEFORE", start)
	// fmt.Println(t)
	// fmt.Println()

	fstart := float64(start)
	var keys float64
	for i := fstart; ; i++ {
		v := t.Get(i)
		// fmt.Println("starting with", i, v)
		if v == nil {
			keys = i - 1
			break
		}
	}

	// fmt.Println("keys", keys)

	for k := keys; k >= fstart; k-- {
		// fmt.Println("moving key", k+1, "=", t.Get(k))
		t.ForceSet(k+1, t.Get(k))
	}

	// fmt.Println("AFTER")
	// fmt.Println(t)
	// fmt.Println()
}

func table_insert(args Args) (r Rets, err error) {
	t := args.GetTable()
	if t.readonly {
		return nil, errors.New("attempt to modify a readonly table")
	}

	n := t.Len()

	var pos int

	switch len(args.List) {
	case 2:
		pos = n + 1
	case 3:
		pos = int(args.GetNumber()) // 2nd argument is the position

		// fmt.Println("bumping elements up", pos)
		// move elements up if necessary
		if n > 0 && 1 <= pos && pos <= n {
			bumpelements(t, pos)
		}
	default:
		return nil, errors.New("wrong number of arguments to 'insert'")
	}

	v := args.GetAny()
	t.ForceSet(float64(pos), v)

	return
}

func table_isfrozen(args Args) (r Rets, err error) {
	t := args.GetTable()

	return Rets{t.readonly}, nil
}

func table_maxn(args Args) (r Rets, err error) {
	t := args.GetTable()

	var maxn float64

	// array kvs
	if t.array != nil {
		for i, v := range *t.array {
			if fi := float64(i + 1); v != nil && fi > maxn {
				maxn = fi
			}
		}
	}

	// hash kvs
	if t.node != nil {
		for k, v := range *t.node {
			if fk, ok := k.(float64); ok && v != nil && fk > maxn {
				maxn = fk
			}
		}
	}

	return Rets{maxn}, nil
}

func table_move(args Args) (r Rets, err error) {
	src := args.GetTable()
	a, b, t := args.GetNumber(), args.GetNumber(), args.GetNumber()
	dst := args.GetTable(src)
	if dst.readonly {
		return nil, errors.New("attempt to modify a readonly table")
	}

	for i := a; i <= b; i++ {
		dst.ForceSet(t+i-a, src.Get(i))
	}

	return Rets{dst}, nil
}

func table_pack(args Args) (r Rets, err error) {
	n := float64(len(args.List))
	t := &Table{
		node: &map[any]any{"n": n},
	}
	for i, v := range args.List {
		t.SetArray(i+1, v)
	}

	return Rets{t}, nil
}

func table_remove(args Args) (r Rets, err error) {
	t := args.GetTable()
	if t.readonly {
		return nil, errors.New("attempt to modify a readonly table")
	}

	l := float64(t.Len())
	pos := args.GetNumber(l)

	p := t.Get(pos)
	if uint(pos) == uint(l) {
		t.ForceSet(pos, nil)
	} else if 0 < pos && pos < l {
		for i := pos; i < l; i++ {
			t.ForceSet(i, t.Get(i+1))
		}
		t.ForceSet(l, nil)
	}
	return Rets{p}, nil
}

// ltablib.cpp
type comp func(a, b any) (bool, error) // ton, compton, aint no city quite like miiine

func sort_swap(t *Table, i, j int) {
	arr := *t.array
	// n := t.asize
	// LUAU_ASSERT(unsigned(i) < unsigned(n) && unsigned(j) < unsigned(n)) // contract maintained in sort_less after predicate call

	// no barrier required because both elements are in the array before and after the swap
	arr[i], arr[j] = arr[j], arr[i]
}

func sort_less(t *Table, i, j int, c comp) (res bool, err error) {
	arr, n := *t.array, len(*t.array)
	// LUAU_ASSERT(unsigned(i) < unsigned(n) && unsigned(j) < unsigned(n)) // contract maintained in sort_less after predicate call

	res, err = c(arr[i], arr[j])

	// predicate call may resize the table, which is invalid
	if len(*t.array) != n {
		return false, errors.New("table modified during sorting")
	}
	return
}

func sort_siftheap(t *Table, l, u int, c comp, root int) (err error) {
	// LUAU_ASSERT(l <= u)
	count := u - l + 1

	// process all elements with two children
	for root*2+2 < count {
		left, right := root*2+1, root*2+2
		next := root
		if r, err := sort_less(t, l+next, l+left, c); err != nil {
			return err
		} else if r {
			next = left
		}
		if r, err := sort_less(t, l+next, l+left, c); err != nil {
			return err
		} else if r {
			next = left
		}
		if r, err := sort_less(t, l+next, l+right, c); err != nil {
			return err
		} else if r {
			next = right
		}

		if next == root {
			break
		}

		sort_swap(t, l+root, l+next)
		root = next
	}

	// process last element if it has just one child
	if lastleft := root*2 + 1; lastleft == count-1 {
		if r, err := sort_less(t, l+root, l+lastleft, c); err != nil {
			return err
		} else if r {
			sort_swap(t, l+root, l+lastleft)
		}
	}

	return
}

func sort_heap(t *Table, l, u int, c comp) {
	// LUAU_ASSERT(l <= u)
	count := u - l + 1

	for i := count/2 - 1; i >= 0; i-- {
		sort_siftheap(t, l, u, c, i)
	}

	for i := count - 1; i > 0; i-- {
		sort_swap(t, l, l+i)
		sort_siftheap(t, l, l+i-1, c, 0)
	}
}

func sort_rec(t *Table, l, u, limit int, c comp) (err error) {
	// sort range [l..u] (inclusive, 0-based)
	for l < u {
		// if the limit has been reached, quick sort is going over the permitted nlogn complexity, so we fall back to heap sort
		if limit == 0 {
			sort_heap(t, l, u, c)
			return
		} else if r, err := sort_less(t, u, l, c); err != nil {
			return err
		} else if r { // a[u] < a[l]?
			// sort elements a[l], a[(l+u)/2] and a[u]
			// note: this simultaneously acts as a small sort and a median selector
			sort_swap(t, u, l) // swap a[l] - a[u]
		}
		if u-l == 1 {
			break // only 2 elements
		}

		m := l + ((u - l) >> 1) // midpoint
		if r, err := sort_less(t, m, l, c); err != nil {
			return err
		} else if r { // a[m]<a[l]?
			sort_swap(t, m, l)
		} else if r, err := sort_less(t, u, m, c); err != nil {
			return err
		} else if r { // a[u]<a[m]?
			sort_swap(t, m, u)
		}
		if u-l == 2 {
			break // only 3 elements
		}

		// here l, m, u are ordered; m will become the new pivot
		p := u - 1
		sort_swap(t, m, u-1) // pivot is now (and always) at u-1

		// a[l] <= P == a[u-1] <= a[u], only need to sort from l+1 to u-2
		i := l
		j := u - 1
		for {
			// invariant: a[l..i] <= P <= a[j..u]
			// repeat ++i until a[i] >= P
			i++
			for {
				r, err := sort_less(t, i, p, c)
				if err != nil {
					return err
				} else if !r {
					break
				} else if i >= u {
					return errors.New("invalid order function for sorting")
				}
				i++
			}

			// repeat --j until a[j] <= P
			j--
			for {
				r, err := sort_less(t, p, j, c)
				if err != nil {
					return err
				} else if !r {
					break
				} else if j <= l {
					return errors.New("invalid order function for sorting")
				}
				j--
			}
			if j < i {
				break
			}
			sort_swap(t, i, j)
		}

		// swap pivot a[p] with a[i], which is the new midpoint
		sort_swap(t, p, i)

		// adjust limit to allow 1.5 log2N recursive steps
		limit = (limit >> 1) + (limit >> 2)

		// a[l..i-1] <= a[i] == P <= a[i+1..u]
		// sort smaller half recursively; the larger half is sorted in the next loop iteration
		if i-l < u-i {
			err := sort_rec(t, l, i-1, limit, c)
			if err != nil {
				return err
			}
			l = i + 1
		} else {
			err := sort_rec(t, i+1, u, limit, c)
			if err != nil {
				return err
			}
			u = i - 1
		}
	}

	return
}

func table_sort(args Args) (r Rets, err error) {
	t := args.GetTable()
	if t.readonly {
		return nil, errors.New("attempt to modify a readonly table")
	}

	var c comp
	if len(args.List) == 1 {
		c = jumpLt
	} else {
		fn := args.GetFunction()
		c = func(a, b any) (bool, error) {
			res, err := (*fn)(args.Co, a, b)
			if err != nil {
				return false, err
			}
			return res[0].(bool), nil
		}
	}

	if n := t.Len(); n > 0 {
		return nil, sort_rec(t, 0, n-1, n, c)
	}
	return
}

func table_unpack(args Args) (r Rets, err error) {
	list := args.GetTable()
	i := args.GetNumber(1)
	e := args.GetNumber(float64(list.Len()))
	if i > e {
		return // empty range
	}

	ui, uj := int(i), int(e)
	if uj <= len(*list.array) {
		return (*list.array)[ui-1 : uj], nil
	}

	r = make(Rets, uj-ui+1)
	for k := i; k <= e; k++ {
		r[int(k)-ui] = list.Get(k)
	}

	return
}

var libtable = NewTable([][2]any{
	MakeFn("clear", table_clear),
	MakeFn("clone", table_clone),
	MakeFn("concat", table_concat),
	MakeFn("create", table_create),
	MakeFn("find", table_find),
	MakeFn("freeze", table_freeze),
	MakeFn("insert", table_insert),
	MakeFn("isfrozen", table_isfrozen),
	MakeFn("maxn", table_maxn),
	MakeFn("move", table_move),
	MakeFn("pack", table_pack),
	MakeFn("remove", table_remove),
	MakeFn("sort", table_sort),
	MakeFn("unpack", table_unpack),
})
