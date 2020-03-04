// +build !appengine

package main

import (
	"fmt"
	"os"
	_ "net/http/pprof"

	"github.com/gonuts/commander"
	"github.com/OnlpLab/yap/app"
	"github.com/OnlpLab/yap/webapi"
)

var cmd = &commander.Command{
	UsageLine: os.Args[0] + " app|api",
	Short:     "invoke yap as a standalone app or as an api server",
}

func init() {
	cmd.Subcommands = append(app.AllCommands().Subcommands, webapi.AllCommands().Subcommands...)
	//cmd.Subcommands = app.AllCommands().Subcommands
}

func exit(err error) {
	fmt.Printf("**error**: %v\n", err)
	os.Exit(1)
}

func main() {
	if err := cmd.Dispatch(os.Args[1:]); err != nil {
		exit(err)
	}

	return
}
