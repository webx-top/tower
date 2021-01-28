/*

   Copyright 2016 Wenhui Shen <www.webx.top>

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

package echo

import (
	"errors"

	"github.com/webx-top/validation"
)

// Validator is the interface that wraps the Validate method.
type Validator interface {
	Validate(i interface{}, args ...string) ValidateResult
}

type ValidateResult interface {
	Ok() bool
	Error() error
	Field() string
	Raw() interface{}

	//setter
	SetError(error) ValidateResult
	SetField(string) ValidateResult
	SetRaw(interface{}) ValidateResult
}

func NewValidateResult() ValidateResult {
	return &ValidatorResult{}
}

type ValidatorResult struct {
	error
	field string
	raw   interface{}
}

func (v *ValidatorResult) Ok() bool {
	return v.error == nil
}

func (v *ValidatorResult) Error() error {
	return v.error
}

func (v *ValidatorResult) Field() string {
	return v.field
}

func (v *ValidatorResult) Raw() interface{} {
	return v.raw
}

func (v *ValidatorResult) SetError(err error) ValidateResult {
	v.error = err
	return v
}

func (v *ValidatorResult) SetField(field string) ValidateResult {
	v.field = field
	return v
}

func (v *ValidatorResult) SetRaw(raw interface{}) ValidateResult {
	v.raw = raw
	return v
}

var (
	DefaultNopValidate     Validator = &NopValidation{}
	defaultValidatorResult           = NewValidateResult()
	ErrNoSetValidator                = errors.New(`The validator is not set`)
)

type NopValidation struct {
}

func (v *NopValidation) Validate(_ interface{}, _ ...string) ValidateResult {
	return defaultValidatorResult
}

func NewValidation() Validator {
	return &Validation{
		validator: validation.New(),
	}
}

type Validation struct {
	validator *validation.Validation
}

// Validate 此处支持两种用法：
// 1. Validate(表单字段名, 表单值, 验证规则名)
// 2. Validate(结构体实例, 要验证的结构体字段1，要验证的结构体字段2)
// Validate(结构体实例) 代表验证所有带“valid”标签的字段
func (v *Validation) Validate(i interface{}, args ...string) ValidateResult {
	e := NewValidateResult()
	var err error
	switch m := i.(type) {
	case string:
		field := m
		var value, rule string
		switch len(args) {
		case 2:
			rule = args[1]
			fallthrough
		case 1:
			value = args[0]
		}
		if len(rule) == 0 {
			return e
		}
		_, err = v.validator.ValidSimple(field, value, rule)
	default:
		_, err = v.validator.Valid(i, args...)
	}
	if err != nil {
		return e.SetError(err)
	}
	if v.validator.HasError() {
		vErr := v.validator.Errors[0].WithField()
		e.SetError(vErr)
		e.SetField(vErr.Field)
		e.SetRaw(v.validator.Errors)
		v.validator.Errors = nil
	}
	return e
}
