package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

const defaultConfig = `[dns]
host        = "127.0.0.1"
port        = 5301
origin      = "seed.example.com"
ns          = "ns1.example.com"
email       = "hostmaster.example.com"
max_records = 24

[node]
# mode = "dynamic" requires ` + "`url`" + ` and ` + "`secret`" + ` to receive the online peers list from the node for every ` + "`interval`" + `,
# you can comment out ` + "`url`" + ` and ` + "`secret`" + ` if you just want to serve a static list of ` + "`peers`" + `,
# alive_only = true serves only peers reachable on p2p_port, including hardcoded peers,
# if it is set to false it will serve ` + "`peers`" + ` from the static list without checking reachability.
mode        = "dynamic" # or "static"
peers       = ["1.2.3.4"]
alive_only  = true # or false
url         = "http://127.0.0.1:3413"
secret      = "/home/user/.grin/main/.api_secret"
interval    = 60
p2p_port    = 3414
check_timeout = 3
min_user_agent = "MW/Grin 5.4.0"
`

func main() {
	app := &cli.App{
		Name:      "grin-seeder",
		Usage:     "DNS seed server for the Grin network",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "seed.toml",
				Usage:   "path to config file",
			},
		},
		Action: func(ctx *cli.Context) error {
			cfg, err := loadConfig(ctx.String("config"))
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			ps := newPeerState(&cfg.Node)
			go runMonitor(&cfg.Node, ps)
			runDNS(cfg, ps)
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:      "generate-config",
				Usage:     "Generate an example config file",
				ArgsUsage: "[path]",
				Description: "Writes an example config to the given path. " +
					"If path is a directory, seed.toml is created inside it. " +
					"Defaults to ./seed.toml.",
				Action: func(ctx *cli.Context) error {
					return generateConfig(ctx.Args().First())
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateConfig(arg string) error {
	dest := "seed.toml"

	if arg != "" {
		info, err := os.Stat(arg)
		if err == nil && info.IsDir() {
			dest = filepath.Join(arg, "seed.toml")
		} else if strings.HasSuffix(arg, string(os.PathSeparator)) {
			dest = filepath.Join(arg, "seed.toml")
		} else {
			dest = arg
		}
	}

	if err := os.WriteFile(dest, []byte(defaultConfig), 0644); err != nil {
		return err
	}
	fmt.Printf("config written to %s\n", dest)
	return nil
}
