package oneandone

import (
	"errors"
	"fmt"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/helper/communicator"
	"github.com/mitchellh/packer/packer"
	"log"
)

const BuilderId = "packer.oneandone"

type Builder struct {
	config *Config
	runner multistep.Runner
}

func (b *Builder) Prepare(raws ...interface{}) ([]string, error) {
	c, warnings, errs := NewConfig(raws...)
	if errs != nil {
		return warnings, errs
	}
	b.config = c

	return warnings, nil
}

func (b *Builder) Run(ui packer.Ui, hook packer.Hook, cache packer.Cache) (packer.Artifact, error) {

	state := new(multistep.BasicStateBag)

	state.Put("config", b.config)
	state.Put("hook", hook)
	state.Put("ui", ui)

	steps := []multistep.Step{
		&StepCreateSSHKey{
			Debug:        b.config.PackerDebug,
			DebugKeyPath: fmt.Sprintf("oneandone_%s", b.config.SnapshotName),
		},
		new(stepCreateServer),
		&communicator.StepConnect{
			Config:    &b.config.Comm,
			Host:      commHost,
			SSHConfig: sshConfig,
		},
		&common.StepProvision{},
		new(stepTakeSnapshot),
	}

	if b.config.PackerDebug {
		b.runner = &multistep.DebugRunner{
			Steps:   steps,
			PauseFn: common.MultistepDebugFn(ui),
		}
	} else {
		b.runner = &multistep.BasicRunner{Steps: steps}
	}

	b.runner.Run(state)

	if rawErr, ok := state.GetOk("error"); ok {
		return nil, rawErr.(error)
	}

	if temp, ok := state.GetOk("snapshot_name"); ok {
		b.config.SnapshotName = temp.(string)
	}

	artifact := &Artifact{
		snapshotName: b.config.SnapshotName,
	}

	if id, ok := state.GetOk("snapshot_id"); ok {
		artifact.snapshotId = id.(string)
	} else {
		return nil, errors.New("Image creation has failed.")
	}

	return artifact, nil
}

func (b *Builder) Cancel() {
	if b.runner != nil {
		log.Println("Cancelling the step runner...")
		b.runner.Cancel()
	}
}
