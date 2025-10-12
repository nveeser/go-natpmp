// Package flags provides the Config type and the parsing to
// accurately replicate the arguments of the standard natpmpc client.
package flags

import (
	"flag"
)

type Config struct {
	Help    bool
	Gateway IPValue
	Port    int
	AddSpec PortSpec
}

func (c *Config) ParseArgs(fs *flag.FlagSet, args []string) error {
	if fs == nil {
		fs = flag.NewFlagSet("", flag.ExitOnError)
	}
	fs.BoolVar(&c.Help, "h", false, "show this message")
	fs.IntVar(&c.Port, "P", 0, "Port to use for NAT-PMP Protocol")
	fs.Var(&c.AddSpec, "a", "port specification <public port> <private port> <Protocol> [Lifetime]")
	fs.Var(&c.Gateway, "g", "gateway address")

	var positionalArgs []string
	var err error
	for {
		if err := fs.Parse(args); err != nil {
			return err
		}
		args = args[len(args)-fs.NArg():]
		if args, err = c.AddSpec.consume(args); err != nil {
			return err
		}
		if len(args) == 0 {
			break
		}
		positionalArgs = append(positionalArgs, args[0])
		args = args[1:]
	}
	return fs.Parse(positionalArgs)
}
