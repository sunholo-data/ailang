package main

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE: `ailang models <role|publish|source>`.
//
// One place to answer "what runs this role?" — the question that used to need a
// four-file audit (models.yml + mission-control.sh + config.cloud.yaml + the
// hardcoded executor defaults).

func modelsCommand(args []string) error {
	if len(args) == 0 {
		printModelsHelp()
		return nil
	}
	switch args[0] {
	case "role":
		return modelsRole(args[1:])
	case "publish":
		return modelsPublish(args[1:])
	case "source":
		return modelsSource()
	case "help", "--help", "-h":
		printModelsHelp()
		return nil
	default:
		return fmt.Errorf("unknown models subcommand %q (want: role, publish, source)", args[0])
	}
}

func printModelsHelp() {
	fmt.Println(`ailang models — the model registry (models.yml)

  ailang models role <role> [--lane local|cloud]
      Print the ordered fallback chain for a role, with each entry's harness
      and the exact model string that harness receives.

  ailang models source
      Which registry is this binary using, and what version. Answers the
      question the startup provenance line answers, on demand.

  ailang models publish [--dry-run]
      Validate the repo's models.yml and publish it to the config bucket
      (compare-and-swap). --dry-run validates without writing.`)
}

// modelsRole is the human audit tool AND the mission driver's read path (D3(a)).
func modelsRole(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang models role <role> [--lane local|cloud]")
	}
	role := args[0]
	lane := modelreg.LaneCloud
	for i := 1; i < len(args); i++ {
		if (args[i] == "--lane" || args[i] == "-lane") && i+1 < len(args) {
			switch args[i+1] {
			case "local":
				lane = modelreg.LaneLocal
			case "cloud":
				lane = modelreg.LaneCloud
			default:
				return fmt.Errorf("unknown lane %q (want local or cloud)", args[i+1])
			}
		}
	}

	if err := modelreg.InitModelsConfig(); err != nil {
		return err
	}
	chain, err := modelreg.GlobalModelsConfig.ResolveRole(role, lane)
	if err != nil {
		return err
	}
	for _, e := range chain {
		// Machine-readable and stable: the mission driver reads field 2.
		fmt.Printf("%s\t%s\t%s\n", e.FriendlyName, e.ModelName, e.Executor)
	}
	return nil
}

func modelsSource() error {
	if err := modelreg.InitModelsConfig(); err != nil {
		return err
	}
	fmt.Println(modelreg.LoadedSource.String())
	return nil
}
