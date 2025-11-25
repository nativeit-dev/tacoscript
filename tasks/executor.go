package tasks

import (
	"context"
	"fmt"

	"github.com/nativeit-dev/tacoscript/tasks/shared/executionresult"
)

type Executor interface {
	Execute(ctx context.Context, task CoreTask) executionresult.ExecutionResult
}

type ExecutorRouter struct {
	Executors map[string]Executor
}

func (er ExecutorRouter) GetExecutor(task CoreTask) (Executor, error) {
	e, ok := er.Executors[task.GetTypeName()]
	if !ok {
		return nil, fmt.Errorf("cannot find executor for task %s", task.GetTypeName())
	}

	return e, nil
}
