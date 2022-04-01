/*
Copyright AppsCode Inc. and Contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package evaluator

import (
	"errors"
	"sync"

	supervisorapi "kubeops.dev/supervisor/apis/supervisor/v1alpha1"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"gomodules.xyz/pointer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/klog/v2"
)

const (
	defaultCELVar = "self"
)

var (
	evalProgram map[string]cel.Program
	mutex       sync.Mutex
)

func init() {
	evalProgram = make(map[string]cel.Program)
	mutex = sync.Mutex{}
}

type Evaluator struct {
	obj   *unstructured.Unstructured
	rules supervisorapi.OperationPhaseRules
}

func New(obj *unstructured.Unstructured, rules supervisorapi.OperationPhaseRules) *Evaluator {
	return &Evaluator{
		obj:   obj,
		rules: rules,
	}
}

func (e *Evaluator) EvaluateSuccessfulOperation() (*bool, error) {
	success, err := e.evaluateRule(e.rules.Success)
	if err != nil {
		return nil, err
	}
	if success {
		return pointer.BoolP(true), nil
	}

	inProgress, err := e.evaluateRule(e.rules.InProgress)
	if err != nil {
		return nil, err
	}
	if inProgress {
		return nil, nil
	}

	failed, err := e.evaluateRule(e.rules.Failed)
	if err != nil {
		return nil, err
	}
	if failed {
		return pointer.BoolP(false), nil
	}
	return nil, nil
}

func (e *Evaluator) evaluateRule(rule string) (bool, error) {
	program, err := getProgramForRule(rule)
	if err != nil {
		return false, err
	}

	res, err := e.evalProgram(program)
	if err != nil {
		return false, err
	}
	if res {
		return true, nil
	}
	return false, nil
}

func (e *Evaluator) evalProgram(program cel.Program) (success bool, err error) {
	// Configure error recovery for unexpected panics during evaluation.
	defer func() {
		if r := recover(); r != nil {
			klog.Info("recovered from panic, given rules is invalid")
			success = false
			err = errors.New("given rules is invalid")
			return
		}
	}()

	res := make(map[string]interface{})
	res[defaultCELVar] = e.obj.UnstructuredContent()

	val, _, err := program.Eval(res)
	if err != nil {
		return false, err
	}
	if val.Value() == true {
		success = true
	}
	return
}

func getProgramForRule(rule string) (cel.Program, error) {
	mutex.Lock()
	defer mutex.Unlock()
	program, found := evalProgram[rule]
	if found {
		return program, nil
	}
	env, err := cel.NewEnv(
		cel.Declarations(
			decls.NewVar(defaultCELVar, decls.Dyn)))
	if err != nil {
		return nil, err
	}

	ast, iss := env.Compile(rule)
	if iss != nil && iss.Err() != nil {
		return nil, err
	}

	checked, iss := env.Check(ast)
	if iss != nil && iss.Err() != nil {
		return nil, err
	}
	program, err = env.Program(checked)
	if err != nil {
		return nil, err
	}
	evalProgram[rule] = program
	return program, nil
}
