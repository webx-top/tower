//go:build go1.18

package com

import (
	"sort"
	"strings"
)

type Number interface {
	~uint8 | ~int8 | ~uint16 | ~int16 | ~uint32 | ~int32 | ~uint | ~int | ~uint64 | ~int64 | ~float32 | ~float64
}

type Scalar interface {
	Number | ~bool | ~string
}

func SliceExtractCallback[T Scalar](parts []string, cb func(string) T, recv ...*T) {
	recvEndIndex := len(recv) - 1
	if recvEndIndex < 0 {
		return
	}
	for index, value := range parts {
		if index > recvEndIndex {
			break
		}
		*recv[index] = cb(value)
	}
}

func ExtractSlicex[T any](parts []T, recv ...*T) {
	recvEndIndex := len(recv) - 1
	if recvEndIndex < 0 {
		return
	}
	for index, value := range parts {
		if index > recvEndIndex {
			break
		}
		*recv[index] = value
	}
}

type reverseSortIndex[T any] []T

func (s reverseSortIndex[T]) Len() int { return len(s) }
func (s reverseSortIndex[T]) Less(i, j int) bool {
	return j < i
}
func (s reverseSortIndex[T]) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func ReverseSortIndex[T any](values []T) []T {
	sort.Sort(reverseSortIndex[T](values))
	return values
}

func ChunkSlicex[T any](slice []T, size int) (chunkslice [][]T) {
	length := len(slice)
	if size >= length {
		chunkslice = append(chunkslice, slice)
		return
	}
	end := size
	for i := 0; i < length; i += size {
		if end < length {
			chunkslice = append(chunkslice, slice[i:end])
		} else {
			chunkslice = append(chunkslice, slice[i:])
		}
		end += size
	}
	return
}

func InSlicex[T comparable](v T, sl []T) bool {
	for _, vv := range sl {
		if vv == v {
			return true
		}
	}
	return false
}

func IntersectSlicex[T comparable](slice1, slice2 []T) (inslice []T) {
	for _, v := range slice1 {
		if InSlicex(v, slice2) {
			inslice = append(inslice, v)
		}
	}
	return
}

func DiffSlicex[T comparable](slice1, slice2 []T) (diffslice []T) {
	for _, v := range slice1 {
		if !InSlicex(v, slice2) {
			diffslice = append(diffslice, v)
		}
	}
	return
}

func MergeSlicex[T any](slice1, slice2 []T) (c []T) {
	c = append(slice1, slice2...)
	return
}

func ReduceSlicex[T any](slice []T, a func(T) T) (dslice []T) {
	for _, v := range slice {
		dslice = append(dslice, a(v))
	}
	return
}

func RandSlicex[T any](a []T) (b T) {
	randnum := RandInt(len(a))
	b = a[randnum]
	return
}

func JoinNumbers[T Number](slice []T, sep string) string {
	sb := strings.Builder{}
	for i, v := range slice {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(Str(v))
	}
	return sb.String()
}

func MapKeys[K Scalar, V any](m map[K]V) []K {
	r := make([]K, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	return r
}

func SplitSliceByGroup[K Scalar, V any](rows []V, groupValue func(V) K) map[K][]V {
	r := map[K][]V{}
	for _, row := range rows {
		k := groupValue(row)
		if _, ok := r[k]; !ok {
			r[k] = []V{}
		}
		r[k] = append(r[k], row)
	}
	return r
}
