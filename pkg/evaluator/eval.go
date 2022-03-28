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
	"context"
	"errors"
	"sync"

	supervisorapi "kubeops.dev/supervisor/apis/supervisor/v1alpha1"
	"kubeops.dev/supervisor/pkg/shared"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	obj *supervisorapi.Recommendation
	kc  client.Client
}

func New(obj *supervisorapi.Recommendation, kc client.Client) *Evaluator {
	return &Evaluator{
		obj: obj,
		kc:  kc,
	}
}

func (e *Evaluator) EvaluateSuccessfulOperation(ctx context.Context) (bool, error) {
	success, err := e.evaluateRule(ctx, e.obj.Spec.Rules.Success)
	if err != nil {
		return false, err
	}
	if success {
		return true, nil
	}

	inProgress, err := e.evaluateRule(ctx, e.obj.Spec.Rules.InProgress)
	if err != nil {
		return false, err
	}
	if inProgress {
		return false, nil
	}

	failed, err := e.evaluateRule(ctx, e.obj.Spec.Rules.Failed)
	if err != nil {
		return false, err
	}
	if failed {
		return false, errors.New("operation failed")
	}
	return false, nil
}

func (e *Evaluator) evaluateRule(ctx context.Context, rule string) (bool, error) {
	program, err := getProgramForRule(rule)
	if err != nil {
		return false, err
	}

	res, err := e.evalProgram(ctx, program)
	if err != nil {
		return false, err
	}
	if res {
		return true, nil
	}
	return false, nil
}

func (e *Evaluator) evalProgram(ctx context.Context, program cel.Program) (bool, error) {
	gvk, err := shared.GetGVK(e.obj.Spec.Operation)
	if err != nil {
		return false, err
	}
	unObj := &unstructured.Unstructured{}
	unObj.SetGroupVersionKind(gvk)
	if e.obj.Status.CreatedOperationRef == nil {
		return false, errors.New("created operation is missing in status")
	}

	key := client.ObjectKey{Name: e.obj.Status.CreatedOperationRef.Name, Namespace: e.obj.Namespace}
	err = e.kc.Get(ctx, key, unObj)
	if err != nil {
		return false, err
	}
	obj := make(map[string]interface{})
	obj[defaultCELVar] = unObj.UnstructuredContent()

	val, _, err := program.Eval(obj)
	if err != nil {
		return false, err
	}
	if val.Value() == true {
		return true, nil
	}
	return false, nil
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

	ast, iss := env.Parse(rule)
	if iss.Err() != nil {
		return nil, err
	}

	checked, iss := env.Check(ast)
	if iss.Err() != nil {
		return nil, err
	}
	program, err = env.Program(checked)
	if err != nil {
		return nil, err
	}
	evalProgram[rule] = program
	return program, nil
}
