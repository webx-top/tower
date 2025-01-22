package com

import (
	"math"
	"strings"
)

type chNumberLevel struct {
	Val int64
	// 中文数字节位分权 大权位：万，亿…… 小权位：十，百，千
	IsPower bool
}

// 权位对应表
var chPowerNumMap = map[string]*chNumberLevel{
	"亿": {
		int64(math.Pow10(8)),
		true,
	},
	"万": {
		int64(math.Pow10(4)),
		true,
	},
	"千": {
		int64(math.Pow10(3)),
		false,
	},
	"百": {
		int64(math.Pow10(2)),
		false,
	},
	"十": {
		int64(math.Pow10(1)),
		false,
	},

	// Upper
	"仟": {
		int64(math.Pow10(3)),
		false,
	},
	"佰": {
		int64(math.Pow10(2)),
		false,
	},
	"拾": {
		int64(math.Pow10(1)),
		false,
	},
}

// 数值对应表
var chToArNumberMap = map[string]int64{
	"零": 0,
	"一": 1,
	"二": 2,
	"三": 3,
	"四": 4,
	"五": 5,
	"六": 6,
	"七": 7,
	"八": 8,
	"九": 9,

	// Upper
	"壹": 1,
	"贰": 2,
	"叁": 3,
	"肆": 4,
	"伍": 5,
	"陆": 6,
	"柒": 7,
	"捌": 8,
	"玖": 9,
}

// chPowerNumber 获取权位的信息
func chPowerNumber(inputStr string) (int64, bool) {
	if powerNum, exist := chPowerNumMap[inputStr]; exist {
		return powerNum.Val, powerNum.IsPower
	}
	return 0, false
}

// ConvertNumberChToAr 中文数字转阿拉伯数字
func ConvertNumberChToAr(inputStrNum string) (ansNum int64) {
	chNum := []rune(inputStrNum)
	var (
		curNumber int64
		sumNumber int64
	)
	for index, end := 0, len(chNum); index < end; index++ {
		// 将中文转为阿拉伯数字
		var getNum = chToArNumberMap[string(chNum[index])]
		// 如果转换失败 getNum = -1
		if getNum > 0 {
			// 处理九九八专用
			if curNumber != 0 {
				curNumber *= int64(math.Pow10(1))
				curNumber += getNum
			} else {
				curNumber = getNum
			}

			// 如果队列结束，则终止循环，并将临时数据添加到ansNum
			if index == len(chNum)-1 {
				sumNumber += curNumber
				ansNum += sumNumber
				break
			}
		} else {
			// getNum 等于 -1 或 0 进入这里
			// 没有零对应的权，所以这一步所得出的大概率是上面提到的权位信息
			powerNum, isPower := chPowerNumber(string(chNum[index]))

			// 如果是大权位，则准备下一个权区间
			if isPower {
				sumNumber = (sumNumber + curNumber) * powerNum
				ansNum, sumNumber = ansNum+sumNumber, 0
			} else {
				// 如果是小权位，则将当前数字添加到临时数和中
				if curNumber != 0 {
					sumNumber += curNumber * powerNum
				} else {
					sumNumber += powerNum
				}
			}
			curNumber = 0
			// 如果队列结束，则终止循环，并将临时数据添加到ansNum
			if index == len(chNum)-1 {
				ansNum += sumNumber
				break
			}
		}
	}
	return
}

var numMap = map[int64]string{
	0: "零", 1: "一", 2: "二", 3: "三", 4: "四",
	5: "五", 6: "六", 7: "七", 8: "八", 9: "九",
}

var unitMap = []string{"", "十", "百", "千"}
var bigUnitMap = []string{"", "万", "亿"}

// ConvertNumberArToCh 阿拉伯数字转中文数字
func ConvertNumberArToCh(num int64) string {
	if num == 0 {
		return numMap[0]
	}

	var result []string
	bigUnitIndex := 0

	for num > 0 {
		part := num % 10000
		if part == 0 {
			if len(result) > 0 && result[0] != numMap[0] {
				result = append([]string{numMap[0]}, result...)
			}
		} else {
			partStr := convertArNumberPart(part)
			result = append([]string{partStr + bigUnitMap[bigUnitIndex]}, result...)
		}
		num /= 10000
		bigUnitIndex++
	}

	return strings.Join(result, "")
}

func convertArNumberPart(part int64) string {
	var partResult []string
	unitIndex := 0

	for part > 0 {
		digit := part % 10
		if digit == 0 {
			if len(partResult) > 0 && partResult[0] != numMap[0] {
				partResult = append([]string{numMap[0]}, partResult...)
			}
		} else {
			partResult = append([]string{numMap[digit] + unitMap[unitIndex]}, partResult...)
		}
		part /= 10
		unitIndex++
	}

	return strings.Join(partResult, "")
}

var chNumberToUpperReplacer = strings.NewReplacer(
	"一", "壹",
	"二", "贰",
	"三", "叁",
	"四", "肆",
	"五", "伍",
	"六", "陆",
	"七", "柒",
	"八", "捌",
	"九", "玖",
	"十", "拾",
	"百", "佰",
	"千", "仟",
)

// UpperChNumber 大写中文数字
func UpperChNumber(s string) string {
	return chNumberToUpperReplacer.Replace(s)
}

// ConvertNumberArToChUpper 阿拉伯数字转大写中文数字
func ConvertNumberArToChUpper(num int64) string {
	v := ConvertNumberArToCh(num)
	return UpperChNumber(v)
}
