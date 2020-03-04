package webapi

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/OnlpLab/yap/app"
)

var ApiCommands = []*commander.Command{
	APIServerStartCmd(),
}

func AllCommands() *commander.Command {
	cmd := &commander.Command{
		// Name:        os.Args[0],
		Subcommands: ApiCommands,
		Flag:        *flag.NewFlagSet("api", flag.ExitOnError),
	}
	for _, api := range cmd.Subcommands {
		api.Run = app.NewAppWrapCommand(api.Run)
		api.Flag.IntVar(&app.CPUs, app.NUM_CPUS_FLAG, 0, "Max CPUS to use (runtime.GOMAXPROCS); 0 = all")
		api.Flag.StringVar(&app.CPUProfile, "cpuprofile", "", "write cpu profile to file")
	}
	return cmd
}
