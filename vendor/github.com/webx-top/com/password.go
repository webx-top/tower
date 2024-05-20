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
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Hash 生成哈希值
func Hash(str string) string {
	return Sha256(str)
}

// Salt 盐值加密(生成64个字符)
func Salt() string {
	return Hash(RandStr(64))
}

// MakePassword 创建密码(生成64个字符)
// 可以指定positions用来在hash处理后的密码的不同位置插入salt片段(数量取决于positions的数量)，然后再次hash
func MakePassword(password string, salt string, positions ...uint) string {
	length := len(positions)
	if length < 1 {
		return Hash(salt + password)
	}
	saltLength := len(salt)
	if saltLength < length {
		return Hash(salt + password)
	}
	saltChars := saltLength / length
	hashedPassword := Hash(password)
	maxIndex := len(hashedPassword) - 1
	saltMaxIndex := saltLength - 1
	var result string
	var lastPos int
	for k, pos := range positions {
		end := int(pos)
		start := lastPos
		if start > end {
			start, end = end, start
		}
		if start > maxIndex {
			continue
		}
		lastPos = end
		saltStart := k * saltChars
		saltEnd := saltStart + saltChars
		if end > maxIndex {
			result += hashedPassword[start:] + salt[saltStart:saltEnd]
			continue
		}
		result += hashedPassword[start:end] + salt[saltStart:saltEnd]
		if k == length-1 {
			if end <= maxIndex {
				result += hashedPassword[end:]
			}
			if saltEnd <= saltMaxIndex {
				result += salt[saltEnd:]
			}
		}
	}
	return Hash(result)
}

// CheckPassword 检查密码(密码原文，数据库中保存的哈希过后的密码，数据库中保存的盐值)
func CheckPassword(rawPassword string, hashedPassword string, salt string, positions ...uint) bool {
	return MakePassword(rawPassword, salt, positions...) == hashedPassword
}

// BCryptMakePassword 创建密码(生成60个字符)
func BCryptMakePassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return hash, err
}

// BCryptCheckPassword 检查密码
func BCryptCheckPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

const (
	argon2Type          = "argon2id"
	argon2KeyLen uint32 = 32
	argon2Time   uint32 = 1
	argon2Memory uint32 = 64 * 1024
)

var (
	argon2Threads          = uint8(runtime.NumCPU())
	defaultArgon2Params    = NewArgon2Params()
	ErrPasswordLength0     = errors.New("password length cannot be 0")
	ErrInvalidPasswordHash = errors.New("invalid password hash")
	ErrPasswordMismatch    = errors.New("password did not match")
	// ErrIncompatibleVariant is returned by ComparePasswordAndHash if the
	// provided hash was created using a unsupported variant of Argon2.
	// Currently only argon2id is supported by this package.
	ErrIncompatibleVariant = errors.New("incompatible variant of argon2")

	// ErrIncompatibleVersion is returned by ComparePasswordAndHash if the
	// provided hash was created using a different version of Argon2.
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

func NewArgon2Params() *Argon2Params {
	return &Argon2Params{
		Type:        argon2Type,
		Memory:      argon2Memory,
		Parallelism: argon2Threads,
		Iterations:  argon2Time,
		SaltLength:  32,
		KeyLength:   argon2KeyLen,
	}
}

// Params describes the input parameters used by the Argon2id algorithm. The
// Memory and Iterations parameters control the computational cost of hashing
// the password. The higher these figures are, the greater the cost of generating
// the hash and the longer the runtime. It also follows that the greater the cost
// will be for any attacker trying to guess the password. If the code is running
// on a machine with multiple cores, then you can decrease the runtime without
// reducing the cost by increasing the Parallelism parameter. This controls the
// number of threads that the work is spread across. Important note: Changing the
// value of the Parallelism parameter changes the hash output.
//
// For guidance and an outline process for choosing appropriate parameters see
// https://tools.ietf.org/html/draft-irtf-cfrg-argon2-04#section-4
type Argon2Params struct {
	Type string

	// The amount of memory used by the algorithm (in kibibytes).
	Memory uint32

	// The number of iterations over the memory.
	Iterations uint32

	// The number of threads (or lanes) used by the algorithm.
	// Recommended value is between 1 and runtime.NumCPU().
	Parallelism uint8

	// Length of the random salt. 16 bytes is recommended for password hashing.
	SaltLength uint32

	// Length of the generated key. 16 bytes or more is recommended.
	KeyLength uint32

	Shortly bool
}

func (a *Argon2Params) SetDefaults() {
	if len(a.Type) == 0 {
		a.Type = defaultArgon2Params.Type
	}
	if a.Memory == 0 {
		a.Memory = defaultArgon2Params.Memory
	}
	if a.Iterations == 0 {
		a.Iterations = defaultArgon2Params.Iterations
	}
	if a.Parallelism == 0 {
		a.Parallelism = defaultArgon2Params.Parallelism
	}
	if a.SaltLength == 0 {
		a.SaltLength = defaultArgon2Params.SaltLength
	}
	if a.KeyLength == 0 {
		a.KeyLength = defaultArgon2Params.KeyLength
	}
}

// Argon2MakePasswordShortly takes a plaintext password and generates an argon2 hash
func Argon2MakePasswordShortly(password string, salt ...string) (string, error) {
	params := NewArgon2Params()
	params.Shortly = true
	return Argon2MakePasswordWithParams(password, params, salt...)
}

// Argon2MakePassword takes a plaintext password and generates an argon2 hash
func Argon2MakePassword(password string, salt ...string) (string, error) {
	return Argon2MakePasswordWithParams(password, nil, salt...)
}

func Argon2MakePasswordWithParams(password string, params *Argon2Params, salt ...string) (string, error) {
	if len(password) == 0 {
		return "", ErrPasswordLength0
	}
	if params == nil {
		params = defaultArgon2Params
	} else {
		params.SetDefaults()
	}
	var _salt string
	if len(salt) > 0 && len(salt[0]) > 0 {
		_salt = salt[0]
	} else {
		_salt = RandStr(int(params.SaltLength))
		_salt = base64.StdEncoding.EncodeToString([]byte(_salt))
	}
	var unencodedPassword []byte
	switch params.Type {
	case "argon2id":
		unencodedPassword = argon2.IDKey([]byte(password), []byte(_salt), params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	case "argon2i", "argon2":
		unencodedPassword = argon2.Key([]byte(password), []byte(_salt), params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	default:
		params.Type = "argon2id"
		unencodedPassword = argon2.IDKey([]byte(password), []byte(_salt), params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	}
	encodedPassword := base64.StdEncoding.EncodeToString(unencodedPassword)

	var hash string
	if params.Shortly {
		hash = fmt.Sprintf("%s$%d$%d$%d$%d$%s$%s",
			params.Type, params.Iterations, params.Memory, params.Parallelism, params.KeyLength, _salt, encodedPassword)
	} else {
		//$argon2id$v=19$m=65536,t=3,p=2$Woo1mErn1s7AHf96ewQ8Uw$D4TzIwGO4XD2buk96qAP+Ed2baMo/KbTRMqXX00wtsU
		hash = fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
			params.Type, argon2.Version, params.Memory, params.Iterations, params.Parallelism, _salt, encodedPassword)
	}
	return hash, nil
}

func (a *Argon2Params) Parse(hash string) (salt, key []byte, err error) {
	var saltStr, keyStr string
	if !strings.HasPrefix(hash, `$`) {
		hashParts := strings.SplitN(hash, "$", 7) // <passwordType>$<time>$<memory>$<threads>$<keyLen>$<salt>$<hash>
		if len(hashParts) != 7 {
			return nil, nil, ErrInvalidPasswordHash
		}
		a.Type = hashParts[0]
		switch a.Type {
		case "argon2id", "argon2i", "argon2":
		default:
			return nil, nil, ErrInvalidPasswordHash
		}
		time, err := strconv.ParseUint(hashParts[1], 10, 32)
		if err != nil {
			return nil, nil, err
		}
		a.Iterations = uint32(time)
		memory, err := strconv.ParseUint(hashParts[2], 10, 32)
		if err != nil {
			return nil, nil, err
		}
		a.Memory = uint32(memory)
		threads, err := strconv.ParseUint(hashParts[3], 10, 8)
		if err != nil {
			return nil, nil, err
		}
		a.Parallelism = uint8(threads)
		keyLen, err := strconv.ParseUint(hashParts[4], 10, 32)
		if err != nil {
			return nil, nil, err
		}
		a.KeyLength = uint32(keyLen)
		saltStr = hashParts[5]
		keyStr = hashParts[6]
	} else {
		vals := strings.SplitN(hash, "$", 6)
		if len(vals) != 6 {
			return nil, nil, ErrInvalidPasswordHash
		}
		a.Type = vals[1]
		switch a.Type {
		case "argon2id", "argon2i", "argon2":
		default:
			return nil, nil, ErrInvalidPasswordHash
		}
		var version int
		_, err = fmt.Sscanf(vals[2], "v=%d", &version)
		if err != nil {
			return nil, nil, err
		}
		if version != argon2.Version {
			return nil, nil, ErrIncompatibleVersion
		}

		_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &a.Memory, &a.Iterations, &a.Parallelism)
		if err != nil {
			return nil, nil, err
		}
		saltStr = vals[4]
		keyStr = vals[5]
	}
	salt, err = base64.StdEncoding.DecodeString(saltStr)
	if err != nil {
		return nil, nil, err
	}
	a.SaltLength = uint32(len(salt))
	salt = []byte(saltStr)
	key, err = base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, nil, err
	}
	if a.KeyLength == 0 {
		a.KeyLength = uint32(len(key))
	}

	//Dump(map[string]interface{}{`params`: a, `salt`: saltStr, `key`: keyStr})
	return salt, key, nil
}

// Argon2CheckPassword compares an argon2 hash against plaintext password
func Argon2CheckPassword(hash, password string) error {
	if len(hash) == 0 || len(password) == 0 {
		return ErrPasswordLength0
	}
	params := NewArgon2Params()
	salt, key, err := params.Parse(hash)
	if err != nil {
		return fmt.Errorf(`failed to parse argon2 params: %w`, err)
	}
	var calculatedKey []byte
	switch params.Type {
	case "argon2id":
		calculatedKey = argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	case "argon2i", "argon2":
		calculatedKey = argon2.Key([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	default:
		return ErrInvalidPasswordHash
	}

	if subtle.ConstantTimeCompare(key, calculatedKey) != 1 {
		return ErrPasswordMismatch
	}
	return nil
}

// PBKDF2Key derives a key from the password, salt and iteration count, returning a
// []byte of length keylen that can be used as cryptographic key. The key is
// derived based on the method described as PBKDF2 with the HMAC variant using
// the supplied hash function.
//
// For example, to use a HMAC-SHA-1 based PBKDF2 key derivation function, you
// can get a derived key for e.g. AES-256 (which needs a 32-byte key) by
// doing:
//
//	dk := pbkdf2.Key([]byte("some password"), salt, 4096, 32, sha1.New)
//
// Remember to get a good random salt. At least 8 bytes is recommended by the
// RFC.
//
// Using a higher iteration count will increase the cost of an exhaustive
// search but will also make derivation proportionally slower.
func PBKDF2Key(password, salt []byte, iter, keyLen int, hFunc ...func() hash.Hash) string {
	var h func() hash.Hash
	if len(hFunc) > 0 {
		h = hFunc[0]
	}
	if h == nil {
		h = sha1.New
	}
	prf := hmac.New(h, password)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	U := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		// N.B.: || means concatenation, ^ means XOR
		// for each block T_i = U_1 ^ U_2 ^ ... ^ U_iter
		// U_1 = PRF(password, salt || uint(i))
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		T := dk[len(dk)-hashLen:]
		copy(U, T)

		// U_n = PRF(password, U_(n-1))
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(U)
			U = U[:0]
			U = prf.Sum(U)
			for x := range U {
				T[x] ^= U[x]
			}
		}
	}
	return Base64Encode(string(dk[:keyLen]))
}
