package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func filename(f string) string {
	return fmt.Sprintf("test/%s", f)
}

func litecode(t *testing.T, f string, o int) string {
	cmd := exec.Command("luau-compile", "--binary", fmt.Sprintf("-O%d", o), filename(f))
	bytecode, err := cmd.Output()
	if err != nil {
		t.Error("error running luau-compile:", err)
		return ""
	}

	deserialised := luau_deserialise(bytecode)

	b := strings.Builder{}
	luau_print := Function(func(co *Coroutine, args ...any) (ret []any) {
		// b.WriteString(fmt.Sprint(args...))
		for i, arg := range args {
			b.WriteString(tostring(arg))
			if i < len(args)-1 {
				b.WriteString("\t")
			}
		}
		b.WriteString("\r\n") // yeah
		return
	})

	co, _ := luau_load(deserialised, map[any]any{
		"print": &luau_print,
	})
	co.Resume()

	return b.String()
}

func luau(f string) (string, error) {
	cmd := exec.Command("luau", filename(f))
	o, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(o), nil
}

func TestConformance(t *testing.T) {
	files, err := os.ReadDir("test")
	if err != nil {
		t.Error("error reading test directory:", err)
		return
	}

	// onlyTest := "nprint.luau"

	for _, f := range files {
		name := f.Name()
		// if name != onlyTest {
		// 	continue
		// }

		fmt.Println(" -- Testing", name, "--")

		og, err := luau(name)
		if err != nil {
			t.Error("error running luau:", err)
			return
		}

		outputs := []string{
			litecode(t, name, 0),
			litecode(t, name, 1),
			litecode(t, name, 2),
		}
		fmt.Println()

		for i, o := range outputs {
			if o != og {
				t.Errorf("%d output mismatch:\n-- Expected\n%s\n-- Got\n%s", i, og, o)
				fmt.Println()

				// print mismatch
				oLines := strings.Split(strings.TrimSpace(o), "\n")
				ogLines := strings.Split(strings.TrimSpace(og), "\n")
				for i, line := range ogLines {
					if line != oLines[i] {
						t.Errorf("mismatched line: \n%s\n%v\n%s\n%v", line, []byte(line), oLines[i], []byte(oLines[i]))
					}
				}

				return
			}
		}

		fmt.Println(og)
	}

	fmt.Println("-- Done! --")
	fmt.Println()
}
