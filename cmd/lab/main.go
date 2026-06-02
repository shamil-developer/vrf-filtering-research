package main

import (
	"context"
	"log"
	"os"
	"runtime"

	"github.com/shamil-developer/vrf-filtering-research/internal/chain"
	"github.com/shamil-developer/vrf-filtering-research/internal/handlers"
	"github.com/vishvananda/netns"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Lab struct {
		Namespace string `yaml:"namespace"`
	} `yaml:"lab"`

	Steps []chain.Step `yaml:"steps"`
}

func main() {

	data, err := os.ReadFile(
		"lab.yaml",
	)
	if err != nil {
		log.Fatal(err)
	}

	var cfg Config

	if err := yaml.Unmarshal(
		data,
		&cfg,
	); err != nil {
		log.Fatal(err)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hostNS, err := netns.Get()
	if err != nil {
		log.Fatal(err)
	}
	defer hostNS.Close()

	_ = netns.DeleteNamed(
		cfg.Lab.Namespace,
	)

	labNS, err := netns.NewNamed(
		cfg.Lab.Namespace,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer labNS.Close()

	if err := netns.Set(
		hostNS,
	); err != nil {
		log.Fatal(err)
	}

	tools := &chain.Tools{
		Data: map[string]any{
			"host-ns": hostNS,
			"lab-ns":  labNS,
		},
	}

	c := chain.New(
		tools,
		map[string]chain.Handler{
			"print":            handlers.NewPrint(),
			"create_namespace": handlers.NewCreateNamespace(),
			"create_interface": handlers.NewCreateInterface(),
			"move_interface":   handlers.NewMoveInterface(),
			"attach_interface": handlers.NewAttachInterface(),
			"assign_ip":        handlers.NewAssignIP(),
			"interface_up":     handlers.NewInterfaceUp(),
		},
	)

	if err := c.Run(
		context.Background(),
		cfg.Steps,
	); err != nil {
		log.Fatal(err)
	}
}
