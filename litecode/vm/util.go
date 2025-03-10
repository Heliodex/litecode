package vm

import (
	"crypto/sha3"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const Ext = ".luau"

func luauCompile(path string, o uint8) (bytecode []byte, err error) {
	cmd := exec.Command("luau-compile", "--binary", fmt.Sprintf("-O%d", o), path)
	return cmd.Output()
}

// simpler compilation, deserialisation, and loading API
type compiled struct {
	deserialised
	filepath       string
	compiler       *Compiler
	requireHistory []string
}

// Compiler allows programs to be compiled and deserialised with a cache and given optimisation level.
type Compiler struct {
	cache map[[32]byte]deserialised
	o     uint8
}

// NewCompiler creates a new compiler with the given optimisation level.
func NewCompiler(o uint8) Compiler {
	return Compiler{
		cache: make(map[[32]byte]deserialised),
		o:     o,
	}
}

func (c Compiler) deserialise(b []byte, path string) (compiled, error) {
	d, err := deserialise(b)
	if err != nil {
		return compiled{}, fmt.Errorf("error deserialising bytecode: %w", err)
	}

	return compiled{
		deserialised: d,
		filepath:     path,
		compiler:     &c,
	}, nil
}

// Compile compiles a program at a specific path to bytecode and returns its deserialised form.
func (c Compiler) Compile(path string) (p compiled, err error) {
	// hash path instead of bytecode
	hash := sha3.Sum256([]byte(path))
	if d, ok := c.cache[hash]; ok {
		return compiled{
			deserialised: d,
			filepath:     path,
			compiler:     &c,
		}, nil
	}

	pathext := path + Ext
	// find if file at path exists
	if _, err := os.Stat(pathext); err != nil {
		if _, err := os.Stat(path); err != nil {
			return compiled{}, errors.New("error finding file")
		}
		// init.luau directory
		pathext = path
	}

	b, err := luauCompile(pathext, c.o)
	if err != nil {
		return compiled{}, fmt.Errorf("error compiling file: %w", err)
	}

	p, err = c.deserialise(b, path)
	if err != nil {
		return
	}

	c.cache[hash] = p.deserialised
	return
}

func (p compiled) Load(env Env) (co Coroutine, cancel func()) {
	return loadmodule(p, env, map[string]Rets{}, nil)
}
