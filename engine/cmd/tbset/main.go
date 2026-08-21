//go:build windows

package main

import (
	"fmt"
	"os"

	"github.com/khiemnguyen/remotune/engine/internal/wintune"
)

func main() {
	var m wintune.TaskbarManager

	show := func(tag string) {
		s, err := m.GetState()
		if err != nil {
			fmt.Printf("%s: read failed: %v\n", tag, err)
			return
		}
		fmt.Printf("%-10s live=%-5v persisted=%-5v agreed=%-5v abmState=0x%02X\n",
			tag, s.Live, s.Persisted, s.Agreed(), s.ABMState)
	}

	if len(os.Args) < 2 {
		show("current")
		return
	}

	var want bool
	switch os.Args[1] {
	case "on":
		want = true
	case "off":
		want = false
	default:
		fmt.Fprintln(os.Stderr, "usage: tbset [on|off]")
		os.Exit(2)
	}

	show("before")
	res, err := m.SetAutoHide(want)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set failed: %v\n", err)
		show("after")
		os.Exit(1)
	}
	fmt.Printf("set ok: changed=%v verified=%v\n", res.Changed, res.Verified)
	show("after")
}
