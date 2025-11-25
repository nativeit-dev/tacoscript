package parser

import "github.com/nativeit-dev/tacoscript/tasks"

type TaskFieldParseFn func(t tasks.CoreTask, path string, val interface{}) error

type TaskField struct {
	ParseFn   TaskFieldParseFn
	FieldName string
}

type TaskFieldsParserConfig map[string]TaskField
