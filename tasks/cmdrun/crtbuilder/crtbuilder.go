package crtbuilder

import (
	"fmt"

	"github.com/nativeit-dev/tacoscript/conv"
	"github.com/nativeit-dev/tacoscript/tasks"
	"github.com/nativeit-dev/tacoscript/tasks/cmdrun"
	"github.com/nativeit-dev/tacoscript/tasks/shared/builder"
	"github.com/nativeit-dev/tacoscript/tasks/shared/builder/parser"
)

type TaskBuilder struct {
}

var cmdRunTaskParamsFnMap = parser.TaskFieldsParserConfig{
	tasks.NameField: parser.TaskField{
		ParseFn: func(task tasks.CoreTask, path string, val interface{}) error {
			t := task.(*cmdrun.Task)
			t.Named.Name = fmt.Sprint(val)
			return nil
		},
		FieldName: "Name",
	},
	tasks.NamesField: parser.TaskField{
		ParseFn: func(task tasks.CoreTask, path string, val interface{}) error {
			var err error
			t := task.(*cmdrun.Task)
			t.Named.Names, err = conv.ConvertToValues(val)
			return err
		},
		FieldName: "Names",
	},
	tasks.EnvField: parser.TaskField{
		ParseFn: func(task tasks.CoreTask, path string, val interface{}) error {
			var err error
			t := task.(*cmdrun.Task)
			t.Envs, err = conv.ConvertToKeyValues(val, path)
			return err
		},
		FieldName: "Env",
	},
}

func (tb TaskBuilder) Build(typeName, path string, params interface{}) (t tasks.CoreTask, err error) {
	task := &cmdrun.Task{
		TypeName: typeName,
		Path:     path,
	}

	errs := builder.Build(typeName, path, params, task, cmdRunTaskParamsFnMap)

	return task, errs.ToError()
}
