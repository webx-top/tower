//go:build go1.18

package com

import "reflect"

func IfTrue[V any](condition bool, yesValue V, noValue V) V {
	if condition {
		return yesValue
	}
	return noValue
}

func IfFalse[V any](condition bool, yesValue V, noValue V) V {
	if !condition {
		return yesValue
	}
	return noValue
}

func IfZero[V any](condition any, yesValue V, noValue V) V {
	if reflect.ValueOf(condition).IsZero() {
		return yesValue
	}
	return noValue
}
