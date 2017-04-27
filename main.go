package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/input"
	shellquote "github.com/kballard/go-shellquote"
)

// ConfigsModel ...
type ConfigsModel struct {
	WorkDir  string
	Browsers string
	Options  string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		WorkDir:  os.Getenv("workdir"),
		Browsers: os.Getenv("browsers"),
		Options:  os.Getenv("options"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- WorkDir: %s", configs.WorkDir)
	log.Printf("- Browsers: %s", configs.Browsers)
	log.Printf("- Options: %s", configs.Options)
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfDirExists(configs.WorkDir); err != nil {
		return fmt.Errorf("WorkDir: %s", err)
	}

	if err := input.ValidateIfNotEmpty(configs.Browsers); err != nil {
		return fmt.Errorf("Browsers: %s", err)
	}

	return nil
}

func fail(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func checkProgramInstalledPath(clcommand string) (string, error) {
	cmd := exec.Command("which", clcommand)
	cmd.Stderr = os.Stderr
	outBytes, err := cmd.Output()
	outStr := string(outBytes)
	return strings.TrimSpace(outStr), err
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		fail("Issue with input: %s", err)
	}

	fmt.Println()
	log.Infof("Searching for karma binary")

	workDir, err := pathutil.AbsPath(configs.WorkDir)
	if err != nil {
		fail("Failed to expand WorkDir (%s), error: %s", configs.WorkDir, err)
	}

	karmaBinPth := filepath.Join(workDir, "node_modules", ".bin", "karma")
	if exist, err := pathutil.IsPathExists(karmaBinPth); err != nil {
		fail("Failed to check if karma bin exist at: %s, error: %s", karmaBinPth, err)
	} else if !exist {
		log.Printf("karma bin not found in node_modules")

		if pth, err := checkProgramInstalledPath("karma"); err == nil && pth != "" {
			log.Printf("Using system installed karma...")

			karmaBinPth = pth
		} else {
			log.Printf("Installing karma...")

			cmd := command.New("npm", "install", "karma-jasmine")

			cmd.SetStdout(os.Stdout)
			cmd.SetStderr(os.Stderr)

			log.Donef("$ %s", cmd.PrintableCommandArgs())

			if err := cmd.Run(); err != nil {
				fail("Failed to install karma runner, error: %s", err)
			}
		}
	} else {
		log.Printf("Using karma in node_modules...")
	}

	fmt.Println()
	log.Infof("Running karma-jasmine tests")

	cmdSlice := []string{karmaBinPth, "start", "--single-run", "--browsers", configs.Browsers}

	if configs.Options != "" {
		options, err := shellquote.Split(configs.Options)
		if err != nil {
			fail("Failed to shell split Options (%s), error: %s", configs.Options, err)
		}

		cmdSlice = append(cmdSlice, options...)
	}

	cmd := command.New(cmdSlice[0], cmdSlice[1:]...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	cmd.SetDir(workDir)

	log.Donef("$ %s", cmd.PrintableCommandArgs())

	if err := cmd.Run(); err != nil {
		fail("cordova failed, error: %s", err)
	}
}
