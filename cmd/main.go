package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true

	app.Commands = []*cli.Command{
		{
			Name:    "deploy",
			Aliases: []string{"d", "dep", "depl"},
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "kubeconfig",
					Aliases: []string{"k"},
					Value:   getKubeConfigPath(),
					Usage:   "path to kubeconfig file",
					EnvVars: []string{"KUBECONFIG"},
				},
				&cli.StringFlag{
					Name:    "namespace",
					Aliases: []string{"n"},
					Value:   "default",
					Usage:   "path to deployment namespace",
				},
				&cli.StringFlag{
					Name:    "token",
					Aliases: []string{"t"},
					Usage:   "git token to download deployments",
					EnvVars: []string{"LUBE_GIT_TOKEN"},
				},
			},
			Action: func(c *cli.Context) error {
				// TODO - fix deployment
				//ld := lube.NewLubeDeployer(c.String("kubeconfig"))

				//err := ld.DeployArchive(context.Background(), u.String(), c.String("token"))
				//if err != nil {
				//	return fmt.Errorf("could not deploy: %s", deployment)
				//}

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getKubeConfigPath() string {
	if kubepath, ok := os.LookupEnv("KUBECONFIG"); ok {
		return kubepath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.kube/config"
	}
	return filepath.Join(home, ".kube", "config")
}
