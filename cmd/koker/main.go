package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/ntk148v/koker/pkg/constants"
	"github.com/ntk148v/koker/pkg/containers"
	"github.com/ntk148v/koker/pkg/images"
	"github.com/ntk148v/koker/pkg/network"
	"github.com/ntk148v/koker/pkg/utils"
)

var version = "v0.0.1"

func main() {
	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// Default level is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	// NOTE(kiennt26): Pretty logging, log a human-friendly,
	// colorized output because I like it!
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if os.Getuid() != 0 {
		log.Fatal().Msg("You need root privileges to run `koker`")
	}

	if err := utils.InitKokerDirs(); err != nil {
		log.Fatal().Err(err).Msg("Unable to create requisite directories")
	}

	rand.Seed(time.Now().UnixNano())

	app := &cli.App{
		Name:                 "koker",
		Version:              version,
		EnableBashCompletion: true,
		Authors: []*cli.Author{
			{
				Name:  "Kien Nguyen-Tuan",
				Email: "kiennt2609@gmail.com",
			},
		},
		Usage: "Kien's mini Docker",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Disable logging altogether (quiet mode)",
				Value:   false,
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"D"},
				Usage:   "Set log level to debug. You will see step-by-step what were executed",
				Value:   false,
			},
		},
		Before: func(ctx *cli.Context) error {
			quiet := ctx.Bool("quiet")
			if quiet {
				zerolog.SetGlobalLevel(zerolog.Disabled)
			} else {
				debug := ctx.Bool("debug")
				if debug {
					zerolog.SetGlobalLevel(zerolog.DebugLevel)
				}
			}

			// Load image registry
			if err := images.LoadRepository(); err != nil {
				log.Fatal().Err(err).Msg("Unable to load image registry")
			}

			return nil
		},
	}

	defer images.SaveRepository()

	containerCmd := &cli.Command{
		Name:    "container",
		Usage:   "Manage container",
		Aliases: []string{"c"},
		Subcommands: []*cli.Command{
			{
				Name:      "run",
				Usage:     "Run a command in a new container",
				ArgsUsage: "IMAGE [COMMAND]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "hostname",
						Usage: "Container hostname",
					},
					&cli.IntFlag{
						Name:    "mem",
						Aliases: []string{"m"},
						Usage:   "Memory limit in MB",
						Value:   -1,
					},
					&cli.IntFlag{
						Name:    "swap",
						Aliases: []string{"sw"},
						Usage:   "Swap limit in MB",
						Value:   -1,
					},
					&cli.Float64Flag{
						Name:    "cpus",
						Aliases: []string{"c"},
						Usage:   "Number of CPU cores to restrict to",
						Value:   -1.0,
					},
					&cli.IntFlag{
						Name:    "pids",
						Aliases: []string{"p"},
						Usage:   "Number of max processes to allow",
						Value:   -1,
					},
				},
				Action: func(ctx *cli.Context) error {
					// Create and setup network bridge
					if ok, _ := network.CheckBridgeUp(constants.KokerBridgeName); !ok {
						if err := network.SetupBridge(constants.KokerBridgeName,
							constants.KokerBridgeDefaultIP+"/16"); err != nil {
							return errors.Wrap(err, "unable to create default bridge")
						}
					}

					args := ctx.Args()
					if !args.Present() {
						return errors.New("missing required arguments")
					}
					image := args.Get(0)

					var commands []string
					if len(args.Slice()) >= 2 {
						commands = args.Slice()[1:]
					}

					c := containers.NewContainer(utils.GenUID())

					// Init container
					if err := c.Run(image, commands, ctx.String("hostname"), ctx.Int("mem"), ctx.Int("swap"),
						ctx.Int("pids"), ctx.Float64("cpus"), ctx.Bool("quiet"), ctx.Bool("debug")); err != nil {
						return fmt.Errorf("error initializing container: %v", err)
					}
					return nil
				},
			},
			{
				Name:     "child",
				HideHelp: true,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "hostname",
						Usage: "Container hostname",
					},
					&cli.IntFlag{
						Name:    "mem",
						Aliases: []string{"m"},
						Usage:   "Memory limit in MB",
						Value:   -1,
					},
					&cli.IntFlag{
						Name:    "swap",
						Aliases: []string{"sw"},
						Usage:   "Swap limit in MB",
						Value:   -1,
					},
					&cli.Float64Flag{
						Name:    "cpus",
						Aliases: []string{"c"},
						Usage:   "Number of CPU cores to restrict to",
						Value:   -1.0,
					},
					&cli.IntFlag{
						Name:    "pids",
						Aliases: []string{"p"},
						Usage:   "Number of max processes to allow",
						Value:   -1,
					},
				},
				Action: func(ctx *cli.Context) error {
					args := ctx.Args()
					container := args.Get(0)

					var commands []string
					if len(args.Slice()) >= 2 {
						commands = args.Slice()[1:]
					}

					c := containers.NewContainer(container)
					if err := c.LoadConfig(); err != nil {
						return err
					}

					// Execute command
					if err := c.ExecuteCommand(commands, ctx.String("hostname"), ctx.Int("mem"), ctx.Int("swap"),
						ctx.Int("pids"), ctx.Float64("cpus")); err != nil {
						return fmt.Errorf("error executing container command: %v", err)
					}
					return nil
				},
			},
			{
				Name:  "rm",
				Usage: "Remove a container",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the removal of a running container (uses SIGKILL)",
						Value:   false,
					},
				},
				Action: func(ctx *cli.Context) error {
					// Remove container
					return nil
				},
			},
			{
				Name:  "ls",
				Usage: "List running containers",
				Flags: []cli.Flag{},
				Action: func(ctx *cli.Context) error {
					// List all running containers
					cs, err := containers.ListAllContainers()
					if err != nil {
						return errors.Wrap(err, "unable to list all containers")
					}
					return utils.GenTemplate("container", constants.ContainersTemplate, cs)
				},
			},
		},
	}

	imageCmd := &cli.Command{
		Name:    "image",
		Usage:   "Manage images",
		Aliases: []string{"i"},
		Subcommands: []*cli.Command{
			{
				Name:  "ls",
				Usage: "List all available images",
				Flags: []cli.Flag{},
				Action: func(ctx *cli.Context) error {
					// List all images
					is, err := images.ListAllImages()
					if err != nil {
						return errors.Wrap(err, "unable to list all images")
					}
					return utils.GenTemplate("image", constants.ImagesTemplate, is)
				},
			},
			{
				Name:  "pull",
				Usage: "Pull an image or a repository from a registry",
				Flags: []cli.Flag{},
				Action: func(ctx *cli.Context) error {
					return nil
				},
			},
			{
				Name:  "rm",
				Usage: "Remove a image",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the removal of the image",
						Value:   false,
					},
				},
				Action: func(ctx *cli.Context) error {
					// Remove image
					return nil
				},
			},
		},
	}

	app.Commands = []*cli.Command{
		containerCmd,
		imageCmd,
	}
	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("Something went wrong")
	}
}
