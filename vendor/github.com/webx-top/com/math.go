// Copyright 2014 com authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package com

import (
	"math"
)

// PowInt is int type of math.Pow function.
func PowInt(x int, y int) int {
	num := 1
	for i := 0; i < y; i++ {
		num *= x
	}
	return num
}

// Round 保留几位小数(四舍五入)
func Round(f float64, n int) float64 {
	pow10n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10n)*pow10n) / pow10n
}
