package main

import (
	"context"
	"errors"
	"github.com/fabled-se/lube"
	"github.com/urfave/cli/v2"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

func main() {
	app := cli.NewApp()
	app.EnableBashCompletion = true

	app.Commands = []*cli.Command{
		{
			Name: "deploy",
			Aliases:[]string{"d","dep","depl"},
			Flags: []cli.Flag {
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
			},
			Action: func(c *cli.Context) error {

				ld := lube.NewLubeDeployer(c.String("kubeconfig"),c.String("namespace") )

				deployment := c.Args().Get(0)

				if(len(deployment)<1){
					return errors.New("No deployment specified")
				}

				u, err := url.Parse(deployment)
				if err != nil || u.Scheme == "" || u.Host == "" {
					err = ld.DeployDirectoryRecursive(context.Background(), deployment)
					if(err!=nil){
						return errors.New("Could not deploy directory: "+ deployment)
					}
					return nil
				}

				err = ld.DeployArchiveUrl(context.Background(),u.String())
				if(err!=nil){
					return errors.New("Could not deploy url: "+ deployment)
				}

				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getKubeConfigPath() string{
	if kubepath, ok := os.LookupEnv("KUBECONFIG"); ok{
		return kubepath
	}
	home, err:= os.UserHomeDir()
	if(err!=nil){
		return "~/.kube/config"
	}
	return filepath.Join(home,".kube","config")
}