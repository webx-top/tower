package com

import (
	"fmt"
	"math"
	"strings"
)

func Float2int(v interface{}) int {
	s := fmt.Sprintf("%f", Float64(v))
	i := strings.SplitN(s, `.`, 2)[0]
	return Int(i)
}

func Float2uint(v interface{}) uint {
	s := fmt.Sprintf("%f", Float64(v))
	i := strings.SplitN(s, `.`, 2)[0]
	return Uint(i)
}

func Float2int64(v interface{}) int64 {
	s := fmt.Sprintf("%f", Float64(v))
	i := strings.SplitN(s, `.`, 2)[0]
	return Int64(i)
}

func Float2uint64(v interface{}) uint64 {
	s := fmt.Sprintf("%f", Float64(v))
	i := strings.SplitN(s, `.`, 2)[0]
	return Uint64(i)
}

func NumberTrim(number string, precision int, separator ...string) string {
	s := number
	if len(s) == 0 {
		if precision <= 0 {
			return `0`
		}
		return `0.` + strings.Repeat(`0`, precision)
	}
	p := strings.LastIndex(s, `.`)
	if p < 0 {
		if precision > 0 {
			s += `.` + strings.Repeat(`0`, precision)
		}
		return numberWithSeparator(s, separator...)
	}
	if precision <= 0 {
		return numberWithSeparator(s[0:p], separator...)
	}
	r := s[p+1:]
	if len(r) >= precision {
		return numberWithSeparator(s[0:p]+`.`+r[0:precision], separator...)
	}
	return numberWithSeparator(s, separator...)
}

func NumberTrimZero(number string) string {
	p := strings.LastIndex(number, `.`)
	if p < 0 {
		return number
	}
	d := strings.TrimRight(number[p+1:], `0`)
	if len(d) == 0 {
		return number[0:p]
	}
	return number[0:p] + `.` + d
}

func numberWithSeparator(r string, separator ...string) string {
	d := `,`
	var trimZero bool
	length := len(separator)
	if length > 0 {
		d = separator[0]
		if length > 1 && strings.EqualFold(separator[1], `trimZero`) {
			trimZero = true
		}
	}
	p := strings.LastIndex(r, `.`)
	var (
		i int
		v string
	)
	if p <= 0 {
		if len(d) == 0 {
			return r
		}
		i = len(r)
	} else {
		i = p
		v = r[i:]
		if trimZero {
			d := strings.TrimRight(r[i+1:], `0`)
			if len(d) == 0 {
				v = ``
			} else {
				v = `.` + d
			}
		}
		if len(d) == 0 {
			return r[0:i] + v
		}
	}
	j := int(math.Ceil(float64(i) / float64(3)))
	s := make([]string, j)
	for i > 0 && j > 0 {
		j--
		start := i - 3
		if start < 0 {
			start = 0
		}
		s[j] = r[start:i]
		i = start
	}
	if s[0] == `-` { // 负数时
		return `-` + strings.Join(s[1:], d) + v
	}
	return strings.Join(s, d) + v
}

func NumberFormat(number interface{}, precision int, separator ...string) string {
	r := fmt.Sprintf(`%.*f`, precision, Float64(number))
	return numberWithSeparator(r, separator...)
}

var mumFormatDefaultArgs = []string{`,`, `trimZero`}

// NumFormat 数字格式化。默认裁剪小数部分右侧的0
func NumFormat(number interface{}, precision int, separator ...string) string {
	length := len(separator)
	switch length {
	case 0:
		separator = mumFormatDefaultArgs
	case 1:
		separator = append(separator, `trimZero`)
	}
	return NumberFormat(number, precision, separator...)
}
