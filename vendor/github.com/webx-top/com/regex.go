// Copyright 2013 com authors
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

import "regexp"

const (
	regexEmailPattern       = `(?i)[A-Z0-9._%+-]+@(?:[A-Z0-9-]+\.)+[A-Z]{2,6}`
	regexStrictEmailPattern = `(?i)[A-Z0-9!#$%&'*+/=?^_{|}~-]+` +
		`(?:\.[A-Z0-9!#$%&'*+/=?^_{|}~-]+)*` +
		`@(?:[A-Z0-9](?:[A-Z0-9-]*[A-Z0-9])?\.)+` +
		`[A-Z0-9](?:[A-Z0-9-]*[A-Z0-9])?`
	regexURLPattern                          = `(ftp|http|https):\/\/(\w+:{0,1}\w*@)?(\S+)(:[0-9]+)?(\/|\/([\w#!:.?+=&%@!\-\/]))?`
	regexUsernamePattern                     = `^[\w\p{Han}]+$`
	regexChinesePattern                      = `^[\p{Han}]+$`
	regexChineseFirstPattern                 = `^[\p{Han}]`
	regexContainsChinesePattern              = `[\p{Han}]+`
	regexEOLPattern                          = "[\r\n]+"
	regexAlphaNumericUnderscorePattern       = `^[a-zA-Z0-9_]+$`
	regexAlphaNumericUnderscoreHyphenPattern = `^[a-zA-Z0-9_-]+$`
)

var (
	regexEmail                        = regexp.MustCompile(regexEmailPattern)
	regexStrictEmail                  = regexp.MustCompile(regexStrictEmailPattern)
	regexURL                          = regexp.MustCompile(regexURLPattern)
	regexUsername                     = regexp.MustCompile(regexUsernamePattern)
	regexChinese                      = regexp.MustCompile(regexChinesePattern)
	regexContainsChinese              = regexp.MustCompile(regexContainsChinesePattern)
	regexChinesePrefix                = regexp.MustCompile(regexChineseFirstPattern)
	regexEOL                          = regexp.MustCompile(regexEOLPattern)
	regexAlphaNumericUnderscore       = regexp.MustCompile(regexAlphaNumericUnderscorePattern)
	regexAlphaNumericUnderscoreHyphen = regexp.MustCompile(regexAlphaNumericUnderscoreHyphenPattern)
	regexFloat                        = regexp.MustCompile(`^[-]?[\d]+\.[\d]+$`)
	regexInteger                      = regexp.MustCompile(`^[-]?[\d]+$`)
	regexUnsignedInteger              = regexp.MustCompile(`^[\d]+$`)
)

// IsAlphaNumericUnderscore 是否仅仅包含字母、数字、和下划线
func IsAlphaNumericUnderscore(s string) bool {
	return regexAlphaNumericUnderscore.MatchString(s)
}

// IsAlphaNumericUnderscoreHyphen 是否仅仅包含字母、数字、下划线和连字符(-)
func IsAlphaNumericUnderscoreHyphen(s string) bool {
	return regexAlphaNumericUnderscoreHyphen.MatchString(s)
}

// IsEmail validate string is an email address, if not return false
// basically validation can match 99% cases
func IsEmail(email string) bool {
	return regexEmail.MatchString(email)
}

// IsEmailRFC validate string is an email address, if not return false
// this validation omits RFC 2822
func IsEmailRFC(email string) bool {
	return regexStrictEmail.MatchString(email)
}

// IsURL validate string is a url link, if not return false
// simple validation can match 99% cases
func IsURL(url string) bool {
	return regexURL.MatchString(url)
}

// IsUsername validate string is a available username
func IsUsername(username string) bool {
	return regexUsername.MatchString(username)
}

// IsChinese validate string is Chinese
func IsChinese(str string) bool {
	return regexChinese.MatchString(str)
}

// HasChinese contains Chinese
func HasChinese(str string) bool {
	return regexContainsChinese.MatchString(str)
}

func HasChineseFirst(str string) bool {
	return regexChinesePrefix.MatchString(str)
}

// IsSingleLineText validate string is a single-line text
func IsSingleLineText(text string) bool {
	return !regexEOL.MatchString(text)
}

// IsMultiLineText validate string is a multi-line text
func IsMultiLineText(text string) bool {
	return regexEOL.MatchString(text)
}

// IsFloat validate string is a float number
func IsFloat(val string) bool {
	return regexFloat.MatchString(val)
}

// IsInteger validate string is a integer
func IsInteger(val string) bool {
	return regexInteger.MatchString(val)
}

// IsUnsignedInteger validate string is a unsigned-integer
func IsUnsignedInteger(val string) bool {
	return regexUnsignedInteger.MatchString(val)
}

// RemoveEOL remove \r and \n
func RemoveEOL(text string) string {
	return regexEOL.ReplaceAllString(text, ` `)
}

// FindChineseWords find chinese words
func FindChineseWords(text string, n ...int) []string {
	var _n int
	if len(n) > 0 {
		_n = n[0]
	} else {
		_n = -1
	}
	matches := regexContainsChinese.FindAllStringSubmatch(text, _n)
	_n = 0
	for _, words := range matches {
		_n += len(words)
	}
	result := make([]string, 0, _n)
	for _, words := range matches {
		result = append(result, words...)
	}
	return result
}

// ReplaceByMatchedIndex  通过 FindAllStringSubmatchIndex 的值来替换
//  matches = FindAllStringSubmatchIndex
//  var replaced string
// 	replacer := ReplaceByMatchedIndex(content, matches, &replaced)
//  for k, v := range matches {
// 		var fullmatch string
// 		replacer(k, v, `newContent`)
//  }
func ReplaceByMatchedIndex(content string, matches [][]int, replaced *string) func(k int, v []int, newInnerStr ...string) {
	endK := len(matches) - 1
	var lastEndIdx int
	return func(k int, v []int, newInnerStr ...string) {
		if len(newInnerStr) > 0 {
			if k == 0 {
				*replaced = content[0:v[0]] + newInnerStr[0]
				if k == endK {
					*replaced += content[v[1]:]
				}
			} else if k == endK {
				*replaced += content[lastEndIdx:v[0]] + newInnerStr[0] + content[v[1]:]
			} else {
				*replaced += content[lastEndIdx:v[0]] + newInnerStr[0]
			}
		} else {
			if k == 0 {
				if k == endK {
					*replaced = content
				} else {
					*replaced = content[0:v[1]]
				}
			} else if k == endK {
				*replaced += content[lastEndIdx:]
			} else {
				*replaced += content[lastEndIdx:v[1]]
			}
		}
		lastEndIdx = v[1]
	}
}

// GetMatchedByIndex 通过 FindAllStringSubmatchIndex 的值获取匹配结果
// matches = FindAllStringSubmatchIndex
//  for _, match := range matches {
// 		var fullmatch string
// 		GetMatchedByIndex(contet, match, &fullmatch)
//  }
func GetMatchedByIndex(content string, v []int, recv ...*string) {
	recvNum := len(recv)
	matchIdx := 0
	matchNum := len(v)
	matchEdx := matchNum - 1
	for idx := 0; idx < recvNum; idx++ {
		if matchIdx > matchEdx {
			return
		}
		if recv[idx] != nil && v[matchIdx] > -1 {
			endIdx := matchIdx + 1
			if endIdx >= matchNum {
				return
			}
			*(recv[idx]) = content[v[matchIdx]:v[endIdx]]
		}
		matchIdx += 2
	}
}
