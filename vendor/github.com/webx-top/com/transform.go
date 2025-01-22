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

package com

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
)

func AsType(typ string, val interface{}) interface{} {
	switch typ {
	case `string`:
		return String(val)
	case `bytes`, `[]byte`:
		return Bytes(val)
	case `bool`:
		return Bool(val)
	case `float64`:
		return Float64(val)
	case `float32`:
		return Float32(val)
	case `int8`:
		return Int8(val)
	case `int16`:
		return Int16(val)
	case `int`:
		return Int(val)
	case `int32`:
		return Int32(val)
	case `int64`:
		return Int64(val)
	case `uint8`:
		return Uint8(val)
	case `uint16`:
		return Uint16(val)
	case `uint`:
		return Uint(val)
	case `uint32`:
		return Uint32(val)
	case `uint64`:
		return Uint64(val)
	default:
		return val
	}
}

func Bytes(val interface{}) []byte {
	switch v := val.(type) {
	case []byte:
		return v
	case nil:
		return nil
	case string:
		return []byte(v)
	default:
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err := enc.Encode(val)
		if err != nil {
			return nil
		}
		return buf.Bytes()
	}
}

func Int64(i interface{}) int64 {
	switch v := i.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case int16:
		return int64(v)
	case int8:
		return int64(v)

	case uint:
		return int64(v)
	case uint64:
		if v <= math.MaxInt64 {
			return int64(v)
		}
		return 0
	case uint32:
		return int64(v)
	case uint16:
		return int64(v)
	case uint8:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		out, _ := strconv.ParseInt(v, 10, 64)
		return out
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseInt(in, 10, 64)
		if err != nil {
			log.Printf("string[%s] covert int64 fail. %s", in, err)
			return 0
		}
		return out
	}
}

func Int(i interface{}) int {
	switch v := i.(type) {
	case int:
		return v
	case int64:
		if v >= math.MinInt && v <= math.MaxInt {
			return int(v)
		}
		return 0
	case int32:
		return int(v)
	case int16:
		return int(v)
	case int8:
		return int(v)

	case uint:
		if v <= math.MaxInt {
			return int(v)
		}
		return 0
	case uint64:
		if v <= math.MaxInt {
			return int(v)
		}
		return 0
	case uint32:
		return int(v)
	case uint16:
		return int(v)
	case uint8:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case string:
		out, _ := strconv.Atoi(v)
		return out
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.Atoi(in)
		if err != nil {
			log.Printf("string[%s] covert int fail. %s", in, err)
			return 0
		}
		return out
	}
}

func Int32(i interface{}) int32 {
	switch v := i.(type) {
	case int:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v)
		}
		return 0
	case int64:
		if v >= math.MinInt32 && v <= math.MaxInt32 {
			return int32(v)
		}
		return 0
	case int32:
		return v
	case int16:
		return int32(v)
	case int8:
		return int32(v)

	case uint:
		if v <= math.MaxInt32 {
			return int32(v)
		}
		return 0
	case uint64:
		if v <= math.MaxInt32 {
			return int32(v)
		}
		return 0
	case uint32:
		if v <= math.MaxInt32 {
			return int32(v)
		}
		return 0
	case uint16:
		return int32(v)
	case uint8:
		return int32(v)
	case float32:
		return int32(v)
	case float64:
		return int32(v)
	case string:
		out, _ := strconv.ParseInt(v, 10, 32)
		return int32(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseInt(in, 10, 32)
		if err != nil {
			log.Printf("string[%s] covert int32 fail. %s", in, err)
			return 0
		}
		return int32(out)
	}
}

func Int16(i interface{}) int16 {
	switch v := i.(type) {
	case int:
		if v >= math.MinInt16 && v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case int64:
		if v >= math.MinInt16 && v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case int32:
		if v >= math.MinInt16 && v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case int16:
		return v
	case int8:
		return int16(v)

	case uint:
		if v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case uint64:
		if v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case uint32:
		if v <= math.MaxInt16 {
			return int16(v)
		}
		return 0
	case uint16:
		if v <= math.MaxInt16 {
			return int16(v)
		}
		return int16(v)
	case uint8:
		return int16(v)
	case float32:
		return int16(v)
	case float64:
		return int16(v)
	case string:
		out, _ := strconv.ParseInt(v, 10, 16)
		return int16(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseInt(in, 10, 16)
		if err != nil {
			log.Printf("string[%s] covert int16 fail. %s", in, err)
			return 0
		}
		return int16(out)
	}
}

func Int8(i interface{}) int8 {
	switch v := i.(type) {
	case int:
		if v >= math.MinInt8 && v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case int64:
		if v >= math.MinInt8 && v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case int32:
		if v >= math.MinInt8 && v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case int16:
		if v >= math.MinInt8 && v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case int8:
		return v

	case uint:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case uint64:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case uint32:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case uint16:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case uint8:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case float32:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case float64:
		if v <= math.MaxInt8 {
			return int8(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseInt(v, 10, 8)
		return int8(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseInt(in, 10, 8)
		if err != nil {
			log.Printf("string[%s] covert int8 fail. %s", in, err)
			return 0
		}
		return int8(out)
	}
}

func Uint64(i interface{}) uint64 {
	switch v := i.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int32:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int16:
		if v < 0 {
			return 0
		}
		return uint64(v)
	case int8:
		if v < 0 {
			return 0
		}
		return uint64(v)

	case uint:
		return uint64(v)
	case uint64:
		return v
	case uint32:
		return uint64(v)
	case uint16:
		return uint64(v)
	case uint8:
		return uint64(v)
	case float32:
		if v > 0 && v <= math.MaxUint64 {
			return uint64(v)
		}
		return 0
	case float64:
		if v > 0 && v <= math.MaxUint64 {
			return uint64(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseUint(v, 10, 64)
		return out
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseUint(in, 10, 64)
		if err != nil {
			log.Printf("string[%s] covert uint64 fail. %s", in, err)
			return 0
		}
		return out
	}
}

func Uint(i interface{}) uint {
	switch v := i.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int64:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int32:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int16:
		if v < 0 {
			return 0
		}
		return uint(v)
	case int8:
		if v < 0 {
			return 0
		}
		return uint(v)

	case uint:
		return v
	case uint64:
		if v > math.MaxUint {
			return 0
		}
		return uint(v)
	case uint32:
		return uint(v)
	case uint16:
		return uint(v)
	case uint8:
		return uint(v)
	case float32:
		if v > 0 {
			return uint(v)
		}
		return 0
	case float64:
		if v > 0 {
			return uint(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseUint(v, 10, 0)
		return uint(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseUint(in, 10, 0)
		if err != nil {
			log.Printf("string[%s] covert uint fail. %s", in, err)
			return 0
		}
		return uint(out)
	}
}

func Uint32(i interface{}) uint32 {
	switch v := i.(type) {
	case int:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case int64:
		if v < 0 || v > math.MaxUint32 {
			return 0
		}
		return uint32(v)
	case int32:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case int16:
		if v < 0 {
			return 0
		}
		return uint32(v)
	case int8:
		if v < 0 {
			return 0
		}
		return uint32(v)

	case uint:
		return uint32(v)
	case uint64:
		if v > math.MaxUint32 {
			return 0
		}
		return uint32(v)
	case uint32:
		return v
	case uint16:
		return uint32(v)
	case uint8:
		return uint32(v)
	case float32:
		if v > 0 {
			return uint32(v)
		}
		return 0
	case float64:
		if v > 0 {
			return uint32(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseUint(v, 10, 32)
		return uint32(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseUint(in, 10, 32)
		if err != nil {
			log.Printf("string[%s] covert uint32 fail. %s", in, err)
			return 0
		}
		return uint32(out)
	}
}

func Uint16(i interface{}) uint16 {
	switch v := i.(type) {
	case int:
		if v < 0 || v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case int64:
		if v < 0 || v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case int32:
		if v < 0 || v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case int16:
		if v < 0 {
			return 0
		}
		return uint16(v)
	case int8:
		if v < 0 {
			return 0
		}
		return uint16(v)

	case uint:
		if v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case uint64:
		if v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case uint32:
		if v > math.MaxUint16 {
			return 0
		}
		return uint16(v)
	case uint16:
		return v
	case uint8:
		return uint16(v)
	case float32:
		if v > 0 && v <= math.MaxUint16 {
			return uint16(v)
		}
		return 0
	case float64:
		if v > 0 && v <= math.MaxUint16 {
			return uint16(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseUint(v, 10, 16)
		return uint16(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseUint(in, 10, 16)
		if err != nil {
			log.Printf("string[%s] covert uint16 fail. %s", in, err)
			return 0
		}
		return uint16(out)
	}
}

func Uint8(i interface{}) uint8 {
	switch v := i.(type) {
	case int:
		if v < 0 || v > math.MaxUint8 {
			return 0
		}
		return uint8(v)
	case int64:
		if v < 0 || v > math.MaxUint8 {
			return 0
		}
		return uint8(v)
	case int32:
		if v < 0 || v > math.MaxUint8 {
			return 0
		}
		return uint8(v)
	case int16:
		if v < 0 || v > math.MaxUint8 {
			return 0
		}
		return uint8(v)
	case int8:
		if v < 0 {
			return 0
		}
		return uint8(v)

	case uint:
		if v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case uint64:
		if v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case uint32:
		if v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case uint16:
		if v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case uint8:
		return v
	case float32:
		if v > 0 && v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case float64:
		if v > 0 && v <= math.MaxUint8 {
			return uint8(v)
		}
		return 0
	case string:
		out, _ := strconv.ParseUint(v, 10, 8)
		return uint8(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseUint(in, 10, 8)
		if err != nil {
			log.Printf("string[%s] covert uint8 fail. %s", in, err)
			return 0
		}
		return uint8(out)
	}
}

func Float32(i interface{}) float32 {
	switch v := i.(type) {
	case float32:
		return v
	case float64:
		if v > math.MaxFloat32 {
			return 0
		}
		return float32(v)

	case int:
		return float32(v)
	case int64:
		return float32(v)
	case int32:
		return float32(v)
	case int16:
		return float32(v)
	case int8:
		return float32(v)

	case uint:
		return float32(v)
	case uint64:
		return float32(v)
	case uint32:
		return float32(v)
	case uint16:
		return float32(v)
	case uint8:
		return float32(v)
	case string:
		out, _ := strconv.ParseFloat(v, 32)
		return float32(out)
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseFloat(in, 32)
		if err != nil {
			log.Printf("string[%s] covert float32 fail. %s", in, err)
			return 0
		}
		return float32(out)
	}
}

func Float64(i interface{}) float64 {
	switch v := i.(type) {
	case float32:
		return float64(v)
	case float64:
		return v

	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case int16:
		return float64(v)
	case int8:
		return float64(v)

	case uint:
		return float64(v)
	case uint64:
		return float64(v)
	case uint32:
		return float64(v)
	case uint16:
		return float64(v)
	case uint8:
		return float64(v)
	case string:
		out, _ := strconv.ParseFloat(v, 64)
		return out
	case nil:
		return 0
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return 0
		}
		out, err := strconv.ParseFloat(in, 64)
		if err != nil {
			log.Printf("string[%s] covert float64 fail. %s", in, err)
			return 0
		}
		return out
	}
}

func Bool(i interface{}) bool {
	switch v := i.(type) {
	case bool:
		return v
	case nil:
		return false
	case string:
		if len(v) == 0 {
			return false
		}
		switch v {
		case `Y`:
			return true
		case `N`:
			return false
		default:
			out, err := strconv.ParseBool(v)
			if err != nil {
				log.Printf("string[%s] covert bool fail. %s", v, err)
				return false
			}
			return out
		}
	default:
		in := fmt.Sprint(i)
		if len(in) == 0 {
			return false
		}
		switch v {
		case `Y`:
			return true
		case `N`:
			return false
		default:
			out, err := strconv.ParseBool(in)
			if err != nil {
				log.Printf("string[%s] covert bool fail. %s", in, err)
				return false
			}
			return out
		}
	}
}

func Str(i interface{}) string {
	return ToStr(i)
}

func String(v interface{}) string {
	return Str(v)
}

// SeekRangeNumbers 遍历范围数值，支持设置步进值。格式例如：1-2,2-3:2
func SeekRangeNumbers(expr string, fn func(int) bool) {
	expa := strings.SplitN(expr, ":", 2)
	step := 1
	switch len(expa) {
	case 2:
		if i, e := strconv.Atoi(strings.TrimSpace(expa[1])); e == nil {
			step = i
		}
		fallthrough
	case 1:
		for _, exp := range strings.Split(strings.TrimSpace(expa[0]), `,`) {
			exp = strings.TrimSpace(exp)
			if len(exp) == 0 {
				continue
			}
			expb := strings.SplitN(exp, `-`, 2)
			var minN, maxN int
			switch len(expb) {
			case 2:
				maxN, _ = strconv.Atoi(strings.TrimSpace(expb[1]))
				fallthrough
			case 1:
				minN, _ = strconv.Atoi(strings.TrimSpace(expb[0]))
			}
			if maxN == 0 {
				if !fn(minN) {
					return
				}
				continue
			}
			for ; minN <= maxN; minN += step {
				if !fn(minN) {
					return
				}
			}
		}
	}
}
