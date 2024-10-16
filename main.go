package main

import (
	"fmt"
	"math"
	"reflect"
)

func assert(condition bool, message string) {
	if !condition {
		panic(message)
	}
}

func move(src []any, a, b, t int, dst []any) {
	for i := a; i <= b; i++ {
		dst[i] = src[i-a+t]
	}
}

func ttisnumber(v any) bool {
	return v.(float64) != 0
}

func ttisstring(v any) bool {
	return v.(string) != ""
}

func ttisboolean(v any) bool {
	return v.(bool)
}

func ttisfunction(v any) bool {
	return v.(func()) != nil
}

// bit32 extraction
func extract(n, field, width uint32) uint32 {
	return uint32(n>>field) & uint32(math.Pow(2, float64(width))-1)
}

// opList contains information about the instruction, each instruction is defined in this format:
// { OP_NAME, OP_MODE, K_MODE, HAS_AUX }
// OP_MODE specifies what type of registers the instruction uses if any
//		0 = NONE
//		1 = A
//		2 = AB
//		3 = ABC
//		4 = AD
//		5 = AE
// K_MODE specifies if the instruction has a register that holds a constant table index, which will be directly converted to the constant in the 2nd pass
//		0 = NONE
//		1 = AUX
//		2 = C
//		3 = D
//		4 = AUX import
//		5 = AUX boolean low 1 bit
//		6 = AUX number low 24 bits
// HAS_AUX boolean specifies whether the instruction is followed up with an AUX word, which may be used to execute the instruction.

type Operator struct {
	Name   string
	Mode   uint8
	KMode  uint8
	HasAux bool
}

var opList = []Operator{
	{"NOP", 0, 0, false},
	{"BREAK", 0, 0, false},
	{"LOADNIL", 1, 0, false},
	{"LOADB", 3, 0, false},
	{"LOADN", 4, 0, false},
	{"LOADK", 4, 3, false},
	{"MOVE", 2, 0, false},
	{"GETGLOBAL", 1, 1, true},
	{"SETGLOBAL", 1, 1, true},
	{"GETUPVAL", 2, 0, false},
	{"SETUPVAL", 2, 0, false},
	{"CLOSEUPVALS", 1, 0, false},
	{"GETIMPORT", 4, 4, true},
	{"GETTABLE", 3, 0, false},
	{"SETTABLE", 3, 0, false},
	{"GETTABLEKS", 3, 1, true},
	{"SETTABLEKS", 3, 1, true},
	{"GETTABLEN", 3, 0, false},
	{"SETTABLEN", 3, 0, false},
	{"NEWCLOSURE", 4, 0, false},
	{"NAMECALL", 3, 1, true},
	{"CALL", 3, 0, false},
	{"RETURN", 2, 0, false},
	{"JUMP", 4, 0, false},
	{"JUMPBACK", 4, 0, false},
	{"JUMPIF", 4, 0, false},
	{"JUMPIFNOT", 4, 0, false},
	{"JUMPIFEQ", 4, 0, true},
	{"JUMPIFLE", 4, 0, true},
	{"JUMPIFLT", 4, 0, true},
	{"JUMPIFNOTEQ", 4, 0, true},
	{"JUMPIFNOTLE", 4, 0, true},
	{"JUMPIFNOTLT", 4, 0, true},
	{"ADD", 3, 0, false},
	{"SUB", 3, 0, false},
	{"MUL", 3, 0, false},
	{"DIV", 3, 0, false},
	{"MOD", 3, 0, false},
	{"POW", 3, 0, false},
	{"ADDK", 3, 2, false},
	{"SUBK", 3, 2, false},
	{"MULK", 3, 2, false},
	{"DIVK", 3, 2, false},
	{"MODK", 3, 2, false},
	{"POWK", 3, 2, false},
	{"AND", 3, 0, false},
	{"OR", 3, 0, false},
	{"ANDK", 3, 2, false},
	{"ORK", 3, 2, false},
	{"CONCAT", 3, 0, false},
	{"NOT", 2, 0, false},
	{"MINUS", 2, 0, false},
	{"LENGTH", 2, 0, false},
	{"NEWTABLE", 2, 0, true},
	{"DUPTABLE", 4, 3, false},
	{"SETLIST", 3, 0, true},
	{"FORNPREP", 4, 0, false},
	{"FORNLOOP", 4, 0, false},
	{"FORGLOOP", 4, 8, true},
	{"FORGPREP_INEXT", 4, 0, false},
	{"FASTCALL3", 3, 1, true},
	{"FORGPREP_NEXT", 4, 0, false},
	{"DEP_FORGLOOP_NEXT", 0, 0, false},
	{"GETVARARGS", 2, 0, false},
	{"DUPCLOSURE", 4, 3, false},
	{"PREPVARARGS", 1, 0, false},
	{"LOADKX", 1, 1, true},
	{"JUMPX", 5, 0, false},
	{"FASTCALL", 3, 0, false},
	{"COVERAGE", 5, 0, false},
	{"CAPTURE", 2, 0, false},
	{"SUBRK", 3, 7, false},
	{"DIVRK", 3, 7, false},
	{"FASTCALL1", 3, 0, false},
	{"FASTCALL2", 3, 0, true},
	{"FASTCALL2K", 3, 1, true},
	{"FORGPREP", 4, 0, false},
	{"JUMPXEQKNIL", 4, 5, true},
	{"JUMPXEQKB", 4, 5, true},
	{"JUMPXEQKN", 4, 6, true},
	{"JUMPXEQKS", 4, 6, true},
	{"IDIV", 3, 0, false},
	{"IDIVK", 3, 2, false},
}

const (
	LUA_MULTRET                = -1
	LUA_GENERALISED_TERMINATOR = -2
)

type LuauSettings struct {
	VectorCtor       func(...float32) any
	VectorSize       uint8
	Extensions       map[uint32]any
	AllowProxyErrors bool
	DecodeOp         func(op uint32) uint32
}

var luau_settings = LuauSettings{
	VectorCtor: func(...float32) any {
		panic("vectorCtor was not provided")
	},
	VectorSize:       4,
	Extensions:       nil,
	AllowProxyErrors: false,
	DecodeOp: func(op uint32) uint32 {
		// println("decoding op", op)
		return op
	},
}

type KVal = any

type Inst struct {
	A       uint32
	B       uint32
	C       uint32
	D       uint32
	E       uint32
	K       KVal
	K0      KVal
	K1      KVal
	K2      KVal
	KC      uint32
	KN      bool
	aux     uint32
	kmode   uint8
	opcode  uint8
	opmode  uint8
	opname  string
	usesAux bool
	value   uint32
}

type Varargs struct {
	len  uint32
	list []KVal
}

type Proto struct {
	maxstacksize uint8
	numparams    uint8
	nups         uint8
	isvararg     bool
	linedefined  uint32
	debugname    string

	sizecode  uint32
	code      []Inst
	debugcode []uint8

	sizek uint32
	k     map[uint32]KVal

	sizep  uint32
	protos []uint32

	lineinfoenabled     bool
	instructionlineinfo []uint32

	bytecodeid uint32
}

type Deserialise struct {
	stringList []string
	protoList  []Proto

	mainProto Proto

	typesVersion uint8
}

func luau_deserialise(stream []byte) Deserialise {
	println("deserialising")
	cursor := uint32(0)

	readByte := func() uint8 {
		byte := stream[cursor]
		cursor += 1
		return byte
	}

	readWord := func() uint32 {
		word := uint32(stream[cursor]) | uint32(stream[cursor+1])<<8 | uint32(stream[cursor+2])<<16 | uint32(stream[cursor+3])<<24
		cursor += 4
		return word
	}

	readFloat := func() float32 {
		float := math.Float32frombits(readWord())
		cursor += 4
		return float
	}

	readDouble := func() float64 {
		double := math.Float64frombits(uint64(readWord()) | uint64(readWord())<<32)
		cursor += 8
		return double
	}

	readVarInt := func() uint32 {
		result := uint32(0)

		for i := range 4 {
			value := readByte()
			result = result | (uint32(value) << (i * 7))
			if value&0x80 == 0 {
				break
			}
		}

		return result
	}

	readString := func() string {
		size := readVarInt()

		if size == 0 {
			return ""
		}

		str := make([]byte, size)
		for i := range str {
			str[i] = readByte()
		}

		return string(str)
	}

	luauVersion := readByte()
	typesVersion := uint8(0)
	if luauVersion == 0 {
		panic("the provided bytecode is an error message")
	} else if luauVersion < 3 || luauVersion > 6 {
		panic("the version of the provided bytecode is unsupported")
	} else if luauVersion >= 4 {
		typesVersion = readByte()
	}

	stringCount := readVarInt()
	stringList := make([]string, stringCount)

	for i := range stringList {
		stringList[i] = readString()
	}

	readInstruction := func(codeList []Inst) bool {
		value := luau_settings.DecodeOp(readWord())
		opcode := uint8(value & 0xFF)

		opinfo := opList[opcode]
		opname := opinfo.Name
		opmode := opinfo.Mode
		kmode := opinfo.KMode
		usesAux := opinfo.HasAux

		inst := Inst{
			opcode:  opcode,
			opname:  opname,
			opmode:  opmode,
			kmode:   kmode,
			usesAux: usesAux,
		}

		codeList = append(codeList, inst)

		if opmode == 1 { /* A */
			inst.A = uint32(value>>8) & 0xFF
		} else if opmode == 2 { /* AB */
			inst.A = uint32(value>>8) & 0xFF
			inst.B = uint32(value>>16) & 0xFF
		} else if opmode == 3 { /* ABC */
			inst.A = uint32(value>>8) & 0xFF
			inst.B = uint32(value>>16) & 0xFF
			inst.C = uint32(value>>24) & 0xFF
		} else if opmode == 4 { /* AD */
			inst.A = uint32(value>>8) & 0xFF
			temp := uint32(value>>16) & 0xFFFF

			if temp < 0x8000 {
				inst.D = temp
			} else {
				inst.D = temp - 0x10000
			}
		} else if opmode == 5 { /* AE */
			temp := uint32(value>>8) & 0xFFFFFF

			if temp < 0x800000 {
				inst.E = temp
			} else {
				inst.E = temp - 0x1000000
			}
		}

		if usesAux {
			aux := readWord()
			inst.aux = aux

			codeList = append(codeList, Inst{value: aux, opname: "auxvalue"})
		}

		return usesAux
	}

	checkkmode := func(inst Inst, k map[uint32]KVal) {
		kmode := inst.kmode

		if kmode == 1 { /* AUX */
			inst.K = k[inst.aux]
		} else if kmode == 2 { /* C */
			inst.K = k[inst.C]
		} else if kmode == 3 { /* D */
			inst.K = k[inst.D]
		} else if kmode == 4 { /* AUX import */
			extend := inst.aux
			count := extend >> 30
			// id0 := (extend >> 20) & 0x3FF

			// inst.K0 = k[id0]
			// inst.KC = count
			if count == 2 {
				id1 := (extend >> 10) & 0x3FF

				inst.K1 = k[id1]
			} else if count == 3 {
				id1 := (extend >> 10) & 0x3FF
				// id2 := (extend >> 0) & 0x3FF

				inst.K1 = k[id1]
				// inst.K2 = k[id2]
			}
		} else if kmode == 5 { /* AUX boolean low 1 bit */
			inst.K = extract(inst.aux, 0, 1) == 1
			inst.KN = extract(inst.aux, 31, 1) == 1
		} else if kmode == 6 { /* AUX number low 24 bits */
			inst.K = k[extract(inst.aux, 0, 24)] // TODO: 1-based indexing
			inst.KN = extract(inst.aux, 31, 1) == 1
		} else if kmode == 7 { /* B */
			inst.K = k[inst.B] // TODO: 1-based indexing
		} else if kmode == 8 { /* AUX number low 16 bits */
			inst.K = inst.aux & 0xF
		}
	}

	readProto := func(bytecodeid uint32) Proto {
		maxstacksize := readByte()
		numparams := readByte()
		nups := readByte()
		isvararg := readByte() != 0

		if luauVersion >= 4 {
			readByte() //-- flags
			typesize := readVarInt()
			cursor += typesize
		}

		sizecode := readVarInt()
		codelist := make([]Inst, sizecode)

		skipnext := false
		for range codelist {
			if skipnext {
				skipnext = false
				continue
			}

			skipnext = readInstruction(codelist)
		}

		debugcodelist := make([]uint8, sizecode)
		for i := range sizecode {
			debugcodelist[i] = codelist[i].opcode
		}

		sizek := readVarInt()
		klist := make(map[uint32]KVal, sizek)

		for i := range klist {
			kt := readByte()
			var k KVal

			if kt == 0 { /* Nil */
				k = nil
			} else if kt == 1 { /* Bool */
				k = readByte() != 0
			} else if kt == 2 { /* Number */
				k = readDouble()
			} else if kt == 3 { /* String */
				k = stringList[readVarInt()-1] // TODO: 1-based indexing
			} else if kt == 4 { /* Function */
				k = readWord()
			} else if kt == 5 { /* Table */
				dataLength := readVarInt()
				k = make([]uint32, dataLength)

				for i := range dataLength {
					k.(map[uint32]KVal)[i] = readVarInt() // TODO: 1-based indexing
				}
			} else if kt == 6 { /* Closure */
				k = readVarInt()
			} else if kt == 7 { /* Vector */
				x, y, z, w := readFloat(), readFloat(), readFloat(), readFloat()

				if luau_settings.VectorSize == 4 {
					k = luau_settings.VectorCtor(x, y, z, w)
				} else {
					k = luau_settings.VectorCtor(x, y, z)
				}
			}

			klist[i] = k
		}

		// -- 2nd pass to replace constant references in the instruction
		for i := range sizecode {
			checkkmode(codelist[i], klist)
		}

		sizep := readVarInt()
		protolist := make([]uint32, sizep)

		for i := range sizep {
			protolist[i] = readVarInt() + 1 // TODO: 1-based indexing
		}

		linedefined := readVarInt()

		debugnameindex := readVarInt()
		var debugname string

		if debugnameindex != 0 {
			debugname = stringList[debugnameindex-1] // TODO: 1-based indexing
		} else {
			debugname = "(??)"
		}

		// -- lineinfo
		lineinfoenabled := readByte() != 0
		var instructionlineinfo []uint32

		if lineinfoenabled {
			linegaplog2 := readByte()

			intervals := uint32((sizecode-1)>>linegaplog2) + 1

			lineinfo := make([]uint32, sizecode)
			abslineinfo := make([]uint32, intervals)

			lastoffset := uint32(0)
			for j := range sizecode {
				lastoffset += uint32(readByte()) // TODO: type convs?
				lineinfo[j] = lastoffset
			}

			lastline := uint32(0)
			for j := range intervals {
				lastline += readWord()
				abslineinfo[j] = lastline % (uint32(math.Pow(2, 32))) // TODO: 1-based indexing
			}

			instructionlineinfo = make([]uint32, sizecode)

			for i := range sizecode {
				// -- p->abslineinfo[pc >> p->linegaplog2] + p->lineinfo[pc];
				instructionlineinfo = append(instructionlineinfo, abslineinfo[i>>(linegaplog2)]+lineinfo[i]) // TODO: 1-based indexing
			}
		}

		// -- debuginfo
		if readByte() != 0 {
			sizel := readVarInt()
			for range sizel {
				readVarInt()
				readVarInt()
				readVarInt()
				readByte()
			}
			sizeupvalues := readVarInt()
			for range sizeupvalues {
				readVarInt()
			}
		}

		return Proto{
			maxstacksize: maxstacksize,
			numparams:    numparams,
			nups:         nups,
			isvararg:     isvararg,
			linedefined:  linedefined,
			debugname:    debugname,

			sizecode:  sizecode,
			code:      codelist,
			debugcode: debugcodelist,

			sizek: sizek,
			k:     klist,

			sizep:  sizep,
			protos: protolist,

			lineinfoenabled:     lineinfoenabled,
			instructionlineinfo: instructionlineinfo,

			bytecodeid: bytecodeid,
		}
	}

	// userdataRemapping (not used in VM, left unused)
	if typesVersion == 3 {
		index := readByte()

		for index != 0 {
			readVarInt()

			index = readByte()
		}
	}

	protoCount := readVarInt()
	protoList := make([]Proto, protoCount)

	for i := range protoCount {
		protoList[i] = readProto(i - 1)
	}

	mainProto := protoList[readVarInt()+1]

	assert(cursor == uint32(len(stream)), "deserialiser cursor position mismatch")

	mainProto.debugname = "(main)"

	return Deserialise{
		stringList: stringList,
		protoList:  protoList,

		mainProto: mainProto,

		typesVersion: typesVersion,
	}
}

func luau_load(stream []byte, env map[uint32]any) (func(...any) []bool, func()) {
	module := luau_deserialise(stream)

	protolist := module.protoList
	mainProto := module.mainProto

	alive := true
	luau_close := func() {
		alive = false
	}

	type Debugging struct {
		pc     int
		top    int
		name   string
		reason string
	}

	type Upval struct {
		value any
		index any
		store any
	}

	var luau_wrapclosure func(module Deserialise, proto Proto, upvals []KVal) func(...any) []bool
	luau_wrapclosure = func(module Deserialise, proto Proto, upvals []KVal) func(...any) []bool {
		println("wrapping closure")
		luau_execute := func(
			// debugging Debugging,
			stack []any,
			protos []uint32,
			code []Inst,
			varargs Varargs,
		) []bool {
			println("executing")

			// if "pc" means "program counter" then this makes a lot more sense than I thought
			top, pc, open_upvalues, generalised_iterators := -1, 1, []*Upval{}, map[Inst]any{}
			constants := proto.k
			debugopcodes := proto.debugcode
			extensions := luau_settings.Extensions

			handlingBreak := false
			inst, op := Inst{}, uint8(0)
			for alive { // TODO: check go scope bruh
				if !handlingBreak {
					inst = code[pc]
					op = inst.opcode
				}

				handlingBreak = false

				// debugging.pc = pc
				// debugging.top = top
				// debugging.name = inst.opname

				pc += 1

				if op == 0 { /* NOP */
					// -- Do nothing
				} else if op == 1 { /* BREAK */
					pc -= 1
					op = debugopcodes[pc]
					handlingBreak = true
				} else if op == 2 { /* LOADNIL */
					stack[inst.A] = nil
				} else if op == 3 { /* LOADB */
					stack[inst.A] = inst.B == 1
					pc += int(inst.C) // TODO: casting overflow?
				} else if op == 4 { /* LOADN */
					stack[inst.A] = inst.D
				} else if op == 5 { /* LOADK */
					stack[inst.A] = inst.K
				} else if op == 6 { /* MOVE */
					stack[inst.A] = stack[inst.B]
				} else if op == 7 { /* GETGLOBAL */
					kv := inst.K.(uint32)

					stack[inst.A] = extensions[kv]
					if stack[inst.A] == nil {
						stack[inst.A] = env[kv]
					}

					pc += 1 // -- adjust for aux
				} else if op == 8 { /* SETGLOBAL */
					kv := inst.K.(uint32)
					env[kv] = stack[inst.A]

					pc += 1 // -- adjust for aux
				} else if op == 9 { /* GETUPVAL */
					uv := upvals[inst.B+1].(Upval)
					stack[inst.A] = uv.store.([]KVal)[uv.index.(uint32)]
				} else if op == 10 { /* SETUPVAL */
					uv := upvals[inst.B+1].(Upval)
					uv.store.([]KVal)[uv.index.(uint32)] = stack[inst.A]
				} else if op == 11 { /* CLOSEUPVALS */
					for i, uv := range open_upvalues {
						if uv.index.(uint32) >= inst.A {
							uv.value = uv.store.([]KVal)[uv.index.(uint32)]
							uv.store = uv
							uv.index = "value" // -- self reference
							open_upvalues[i] = nil
						}
					}
				} else if op == 12 { /* GETIMPORT */
					count := inst.KC
					k0 := inst.K0.(uint32)
					imp := extensions[k0].(map[uint32]KVal)
					if imp == nil {
						imp = env[k0].(map[uint32]KVal)
					}

					if count == 1 {
						stack[inst.A] = imp
					} else if count == 2 {
						stack[inst.A] = imp[inst.K1.(uint32)]
					} else if count == 3 {
						stack[inst.A] = imp[inst.K1.(uint32)].(map[uint32]KVal)[inst.K2.(uint32)]
					}
				} else if op == 13 { /* GETTABLE */
					stack[inst.A] = stack[inst.B].(map[uint32]KVal)[stack[inst.C].(uint32)]
				} else if op == 14 { /* SETTABLE */
					stack[inst.B].(map[uint32]KVal)[stack[inst.C].(uint32)] = stack[inst.A]
				} else if op == 15 { /* GETTABLEKS */
					index := inst.K.(uint32)
					stack[inst.A] = stack[inst.B].(map[uint32]KVal)[index]

					pc += 1 // -- adjust for aux
				} else if op == 16 { /* SETTABLEKS */
					index := inst.K.(uint32)
					stack[inst.B].(map[uint32]KVal)[index] = stack[inst.A]

					pc += 1 // -- adjust for aux
				} else if op == 17 { /* GETTABLEN */
					stack[inst.A] = stack[inst.B].(map[uint32]KVal)[inst.C]
				} else if op == 18 { /* SETTABLEN */
					stack[inst.B].(map[uint32]KVal)[inst.C] = stack[inst.A]
				} else if op == 19 { /* NEWCLOSURE */
					newPrototype := protolist[protos[inst.D]]

					nups := newPrototype.nups
					upvalues := make([]KVal, nups)
					stack[inst.A] = luau_wrapclosure(module, newPrototype, upvalues)

					for i := range nups {
						pseudo := code[pc]

						pc += 1

						t := pseudo.A

						if t == 0 { /* value */
							upvalue := Upval{
								value: stack[pseudo.B],
								index: "value", // -- self reference
							}
							upvalue.store = upvalue

							upvalues[i] = upvalue
						} else if t == 1 { /* reference */
							index := pseudo.B
							prev := open_upvalues[index]

							if prev == nil {
								prev = &Upval{
									index: index,
									store: stack,
								}
								open_upvalues[index] = prev
							}

							upvalues[i] = prev
						} else if t == 2 { /* upvalue */
							upvalues[i] = upvals[pseudo.B]
						}
					}
				} else if op == 20 { /* NAMECALL */
					A := inst.A
					B := inst.B

					kv := inst.K.(uint32)

					sb := stack[B]

					stack[A] = sb // TODO: 1-based indexing

					pc += 1 // -- adjust for aux

					stack[A-1] = sb.(map[uint32]KVal)[kv]
				} else if op == 21 { /* CALL */
					A, B, C := inst.A, inst.B, inst.C

					var params uint32
					if B == 0 {
						params = uint32(top) - A
					} else {
						params = B - 1
					}

					fn := stack[A].(func(...any) []any)
					ret_list := fn(stack[A+1 : A+params]...)

					ret_num := uint32(len(ret_list))

					if C == 0 {
						top = int(A + ret_num - 1)
					} else {
						ret_num = C - 1
					}

				} else if op == 22 { /* RETURN */
					A, B := int(inst.A), int(inst.B)
					b := (B - 1)
					nresults := int(0)

					if b == LUA_MULTRET {
						nresults = top - A + 1
					} else {
						nresults = B - 1
					}

					return any(stack).([]bool)[A : A+nresults-1] // TODO: 1-based indexing
				} else if op == 23 { /* JUMP */
					pc += int(inst.D) // TODO: casting overflow?
				} else if op == 24 { /* JUMPBACK */
					pc += int(inst.D) // TODO: casting overflow?
				} else if op == 25 { /* JUMPIF */
					if stack[inst.A] != nil {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 26 { /* JUMPIFNOT */
					if stack[inst.A] == nil {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 27 { /* JUMPIFEQ */
					if stack[inst.A] == stack[inst.aux] {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 28 { /* JUMPIFLE */
					if stack[inst.A].(int) <= stack[inst.aux].(int) {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 29 { /* JUMPIFLT */
					if stack[inst.A].(int) < stack[inst.aux].(int) {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 30 { /* JUMPIFNOTEQ */
					if stack[inst.A] == stack[inst.aux] {
						pc += 1
					} else {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 31 { /* JUMPIFNOTLE */
					if stack[inst.A].(int) <= stack[inst.aux].(int) {
						pc += 1
					} else {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 32 { /* JUMPIFNOTLT */
					if stack[inst.A].(int) < stack[inst.aux].(int) {
						pc += 1
					} else {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 33 { /* ADD */
					stack[inst.A] = stack[inst.B].(float64) + stack[inst.C].(float64)
				} else if op == 34 { /* SUB */
					stack[inst.A] = stack[inst.B].(float64) - stack[inst.C].(float64)
				} else if op == 35 { /* MUL */
					stack[inst.A] = stack[inst.B].(float64) * stack[inst.C].(float64)
				} else if op == 36 { /* DIV */
					stack[inst.A] = stack[inst.B].(float64) / stack[inst.C].(float64)
				} else if op == 37 { /* MOD */
					stack[inst.A] = math.Mod(stack[inst.B].(float64), stack[inst.C].(float64))
				} else if op == 38 { /* POW */
					stack[inst.A] = math.Pow(stack[inst.B].(float64), stack[inst.C].(float64))
				} else if op == 39 { /* ADDK */
					stack[inst.A] = stack[inst.B].(float64) + inst.K.(float64)
				} else if op == 40 { /* SUBK */
					stack[inst.A] = stack[inst.B].(float64) - inst.K.(float64)
				} else if op == 41 { /* MULK */
					stack[inst.A] = stack[inst.B].(float64) * inst.K.(float64)
				} else if op == 42 { /* DIVK */
					stack[inst.A] = stack[inst.B].(float64) / inst.K.(float64)
				} else if op == 43 { /* MODK */
					stack[inst.A] = math.Mod(stack[inst.B].(float64), inst.K.(float64))
				} else if op == 44 { /* POWK */
					stack[inst.A] = math.Pow(stack[inst.B].(float64), inst.K.(float64))
				} else if op == 45 { /* AND */
					value := stack[inst.B]
					if value != nil {
						stack[inst.A] = stack[inst.C]
						if stack[inst.A] == nil {
							stack[inst.A] = false
						}
					} else {
						stack[inst.A] = value
					}
				} else if op == 46 { /* OR */
					value := stack[inst.B]
					if value != nil {
						stack[inst.A] = value
					} else {
						stack[inst.A] = stack[inst.C]
						if stack[inst.A] == nil {
							stack[inst.A] = false
						}
					}
				} else if op == 47 { /* ANDK */
					value := stack[inst.B]
					if value != nil {
						stack[inst.A] = inst.K
						if stack[inst.A] == nil {
							stack[inst.A] = false
						}
					} else {
						stack[inst.A] = value
					}
				} else if op == 48 { /* ORK */
					value := stack[inst.B]
					if value != nil {
						stack[inst.A] = value
					} else {
						stack[inst.A] = inst.K
						if stack[inst.A] == nil {
							stack[inst.A] = false
						}
					}
				} else if op == 49 { /* CONCAT */
					// TODO: optimise w/ stringbuilders
					s := ""
					for i := inst.B; i <= inst.C; i++ {
						s += stack[i-1].(string) // TODO: 1-based indexing
					}
					stack[inst.A] = s
				} else if op == 50 { /* NOT */
					stack[inst.A] = !stack[inst.B].(bool)
				} else if op == 51 { /* MINUS */
					stack[inst.A] = -stack[inst.B].(float64)
				} else if op == 52 { /* LENGTH */
					stack[inst.A] = len(stack[inst.B].([]any)) // TODO: 1-based indexing
				} else if op == 53 { /* NEWTABLE */
					stack[inst.A] = make(map[uint32]KVal)

					pc += 1 // -- adjust for aux
				} else if op == 54 { /* DUPTABLE */
					template := inst.K.(map[uint32]uint32)
					serialized := make(map[uint32]KVal)
					for _, id := range template {
						serialized[constants[id].(uint32)] = nil // TODO: 1-based indexing
					}
				} else if op == 55 { /* SETLIST */
					A, B := int(inst.A), int(inst.B)
					c := int(inst.C) - 1

					if c == LUA_MULTRET {
						c = top - B + 1
					}

					move(stack, B, B+c-1, int(inst.aux), stack[A].([]any))

					pc += 1 // -- adjust for aux
				} else if op == 56 { /* FORNPREP */
					A := inst.A

					limit := stack[A]
					if !ttisnumber(limit) {
						number := limit.(float64)

						if number == 0 { // TODO: check nils
							panic("invalid 'for' limit (number expected)")
						}

						stack[A] = number
						limit = number
					}

					step := stack[A+1]
					if !ttisnumber(step) {
						number := step.(float64)

						if number == 0 { // TODO: check nils
							panic("invalid 'for' step (number expected)")
						}

						stack[A+1] = number
						step = number
					}

					index := stack[A+2]
					if !ttisnumber(index) {
						number := index.(float64)

						if number == 0 { // TODO: check nils
							panic("invalid 'for' index (number expected)")
						}

						stack[A+2] = number
						index = number
					}

					if step.(float64) > 0 {
						if index.(float64) > limit.(float64) {
							pc += int(inst.D) // TODO: casting overflow?
						}
					} else if limit.(float64) > index.(float64) {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 57 { /* FORNLOOP */
					A := inst.A
					limit := stack[A]
					step := stack[A+1]
					index := stack[A+2]

					stack[A+2] = index.(float64)

					if step.(float64) > 0 {
						if index.(float64) <= limit.(float64) {
							pc += int(inst.D) // TODO: casting overflow?
						}
					} else if limit.(float64) <= index.(float64) {
						pc += int(inst.D) // TODO: casting overflow?
					}
				} else if op == 58 { /* FORGLOOP */
					A := inst.A
					// res := inst.K.(uint32)

					top = int(A + 6)

					// it := stack[A]

					// bruh
					panic("TODO: implement iterators")
				} else if op == 59 { /* FORGPREP_INEXT */
					if !ttisfunction(stack[inst.A]) {
						// yaaaaaaaaaaay reflection (i'm dying inside)
						panic(fmt.Sprintf("attempt to iterate over a %s value", reflect.TypeOf(stack[inst.A]))) // -- FORGPREP_INEXT encountered non-function value
					}

					pc += int(inst.D) // TODO: casting overflow?
				} else if op == 60 { /* FASTCALL3 */
					/* Skipped */
					pc += 1 // adjust for aux
				} else if op == 61 { /* FORGPREP_NEXT */
					if !ttisfunction(stack[inst.A]) {
						panic(fmt.Sprintf("attempt to iterate over a %s value", reflect.TypeOf(stack[inst.A]))) // -- FORGPREP_NEXT encountered non-function value
					}

					pc += int(inst.D) // TODO: casting overflow?
				} else if op == 63 { /* GETVARARGS */
					A := inst.A
					b := int(inst.B) - 1

					if b == LUA_MULTRET {
						b = int(varargs.len)
						top = int(A) + b - 1
					}

					move(varargs.list, 1, b, int(A), stack)
				} else if op == 64 { /* DUPCLOSURE */
					newPrototype := protolist[inst.K.(uint32)] // TODO: 1-based indexing

					nups := newPrototype.nups
					upvalues := make([]KVal, nups)
					stack[inst.A] = luau_wrapclosure(module, newPrototype, upvalues)

					for i := range nups {
						pseudo := code[pc]
						pc += 1

						t := pseudo.A
						if t == 0 { /* value */
							upvalue := Upval{
								value: stack[pseudo.B],
								index: "value", // -- self reference
							}
							upvalue.store = upvalue

							upvalues[i] = upvalue

							// -- references dont get handled by DUPCLOSURE
						} else if t == 2 { /* upvalue */
							upvalues[i] = upvals[pseudo.B]
						}
					}
				} else if op == 65 { /* PREPVARARGS */
					/* Handled by wrapper */
				} else if op == 66 { /* LOADKX */
					kv := inst.K.(uint32)
					stack[inst.A] = kv

					pc += 1 // -- adjust for aux
				} else if op == 67 { /* JUMPX */
					pc += int(inst.E) // TODO: casting overflow?
				} else if op == 68 { /* FASTCALL */
					/* Skipped */
				} else if op == 69 { /* COVERAGE */
					inst.E += 1
				} else if op == 70 { /* CAPTURE */
					/* Handled by CLOSURE */
					panic("encountered unhandled CAPTURE")
				} else if op == 71 { /* SUBRK */
					stack[inst.A] = inst.K.(float64) - stack[inst.C].(float64)
				} else if op == 72 { /* DIVRK */
					stack[inst.A] = inst.K.(float64) / stack[inst.C].(float64)
				} else if op == 73 { /* FASTCALL1 */
					/* Skipped */
				} else if op == 74 { /* FASTCALL2 */
					/* Skipped */
					pc += 1 // adjust for aux
				} else if op == 75 { /* FASTCALL2K */
					/* Skipped */
					pc += 1 // adjust for aux
				} else if op == 76 { /* FORGPREP */
					// ohhh no
					iterator := stack[inst.A]

					if !ttisfunction(iterator) {
						loopInstruction := code[pc+int(inst.D)] // TODO: casting overflow?
						if generalised_iterators[loopInstruction] == nil {
							gen_iterator := func(args ...any) []any {
								println("generating iterator", args)
								const max = 200
								panic("TODO: implement iterators") // gotta be the hardest part
							}

							// TODO: coroutines
							generalised_iterators[loopInstruction] = gen_iterator
						}
					}
				} else if op == 77 { /* JUMPXEQKNIL */
					kn := inst.KN

					if (stack[inst.A] == nil) != kn {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 78 { /* JUMPXEQKB */
					kv := inst.K.(bool)
					kn := inst.KN
					ra := stack[inst.A]

					if ttisboolean(ra) && (ra.(bool) == kv) != kn {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 79 { /* JUMPXEQKN */
					kv := inst.K.(uint32)
					kn := inst.KN
					ra := stack[inst.A].(uint32)

					if (ra == kv) != kn {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 80 { /* JUMPXEQKS */
					kv := inst.K.(uint32)
					kn := inst.KN
					ra := stack[inst.A].(uint32)

					if (ra == kv) != kn {
						pc += int(inst.D) // TODO: casting overflow?
					} else {
						pc += 1
					}
				} else if op == 81 { /* IDIV */
					stack[inst.A] = stack[inst.B].(uint32) / stack[inst.C].(uint32)
				} else if op == 82 { /* IDIVK */
					stack[inst.A] = stack[inst.B].(uint32) / inst.K.(uint32)
				} else {
					panic(fmt.Sprintf("Unsupported Opcode: %s op: %d", inst.opname, op))
				}
			}

			for i, uv := range open_upvalues {
				uv.value = uv.store.([]KVal)[uv.index.(uint32)]
				uv.store = uv
				uv.index = "value" // -- self reference
				open_upvalues[i] = nil
			}

			for i := range generalised_iterators {
				// TODO: close the coroutines using channels or something I have no idea
				generalised_iterators[i] = nil
			}
			return []bool{}
		}

		wrapped := func(args ...any) []bool {
			passed := make([]any, len(args))
			stack := make([]any, proto.maxstacksize)
			varargs := Varargs{
				len:  0,
				list: make([]KVal, 0),
			}

			// TODO: test table.move impl
			move(passed, 1, int(proto.numparams), 0, stack)

			// TODO: check len(passed) vs passed.n
			n := uint8(len(passed))
			if proto.numparams < uint8(n) {
				start := proto.numparams + 1
				l := n - proto.numparams
				varargs.len = uint32(l)
				move(passed, int(start), int(start+l-1), 1, varargs.list)
			}

			passed = nil

			// debugging := Debugging{pc: 0, name: "NONE"}
			// TODO: oh no, this doesn't translate well from Luau at all
			result := luau_execute( /* debugging, */ stack, proto.protos, proto.code, varargs)

			// IF IT PANICS IT PANICS GRAAHHHH
			return result[0:1]
		}

		return wrapped
	}

	return luau_wrapclosure(module, mainProto, []KVal{}), luau_close
}

func main() {
	println("Hello, World!")
}
