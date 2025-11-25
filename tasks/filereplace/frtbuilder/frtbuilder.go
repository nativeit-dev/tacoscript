package frtbuilder

import (
	tasks "github.com/nativeit-dev/tacoscript/tasks"
	"github.com/nativeit-dev/tacoscript/tasks/filereplace"
	"github.com/nativeit-dev/tacoscript/tasks/shared/builder"
)

type TaskBuilder struct {
}

func (tb TaskBuilder) Build(typeName, path string, params interface{}) (t tasks.CoreTask, err error) {
	task := &filereplace.Task{
		TypeName: typeName,
		Path:     path,
	}

	errs := builder.Build(typeName, path, params, task, nil)

	return task, errs.ToError()
}
