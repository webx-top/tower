// Copyright 2013 Apcera Inc. All rights reserved.

// Customized heavily from
// https://github.com/BurntSushi/toml/blob/master/lex.go, which is based on
// Rob Pike's talk: http://cuddle.googlecode.com/hg/talk/lex.html

// The format supported is less restrictive than today's formats.
// Supports mixed Arrays [], nested Maps {}, multiple comment types (# and //)
// Also supports key value assigments using '=' or ':' or whiteSpace()
//   e.g. foo = 2, foo : 2, foo 2
// maps can be assigned with no key separator as well
// semicolons as value terminators in key/value assignments are optional
//
// see lex_test.go for more examples.

package confl

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// IdentityChars Which Identity Characters are allowed?
	IdentityChars = "_."
)

type itemType int

const (
	itemError itemType = iota
	itemNIL            // used in the parser to indicate no type
	itemEOF
	itemKey
	itemText
	itemString
	itemBool
	itemInteger
	itemFloat
	itemDatetime
	itemArrayStart
	itemArrayEnd
	itemMapStart
	itemMapEnd
	itemCommentStart
)

const (
	eof               = 0
	mapStart          = '{'
	mapEnd            = '}'
	keySepEqual       = '='
	keySepColon       = ':'
	arrayStart        = '['
	arrayEnd          = ']'
	arrayValTerm      = ','
	mapValTerm        = ','
	commentHashStart  = '#'
	commentSlashStart = '/'
	dqStringStart     = '"'
	dqStringEnd       = '"'
	sqStringStart     = '\''
	sqStringEnd       = '\''
	optValTerm        = ';'
	blockStart        = '('
	blockEnd          = ')'
	backslash         = '\\'
)

type stateFn func(lx *lexer) stateFn

type lexer struct {
	input string
	start int
	pos   int
	line  int
	state stateFn
	items chan item

	// Allow for backing up up to three runes.
	// This is necessary because TOML contains 3-rune tokens (""" and ''').
	prevWidths [3]int
	nprev      int // how many of prevWidths are in use
	// If we emit an eof, we can still back up, but it is not OK to call
	// next again.
	atEOF bool

	circuitBreaker int
	isEnd          func(lx *lexer, r rune) bool

	// A stack of state functions used to maintain context.
	// The idea is to reuse parts of the state machine in various places.
	// For example, values can appear at the top level or within arbitrarily
	// nested arrays. The last state on the stack is used after a value has
	// been lexed. Similarly for comments.
	stack []stateFn
}

type item struct {
	typ  itemType
	val  string
	line int
}

func (lx *lexer) nextItem() item {
	for {
		select {
		case item := <-lx.items:
			return item
		default:
			lx.state = lx.state(lx)
		}
	}
}

func lex(input string) *lexer {
	lx := &lexer{
		input: input,
		state: lexTop,
		line:  1,
		items: make(chan item, 10),
		stack: make([]stateFn, 0, 10),
		isEnd: isEndNormal,
	}
	return lx
}

func (lx *lexer) push(state stateFn) {
	lx.stack = append(lx.stack, state)
}

func (lx *lexer) pop() stateFn {
	if len(lx.stack) == 0 {
		return lx.errorf("BUG in lexer: no states to pop.")
	}
	li := len(lx.stack) - 1
	last := lx.stack[li]
	lx.stack = lx.stack[0:li]
	return last
}

func (lx *lexer) current() string {
	return lx.input[lx.start:lx.pos]
}

func (lx *lexer) emit(typ itemType) {
	lx.items <- item{typ, lx.current(), lx.line}
	lx.start = lx.pos
}

func (lx *lexer) emitTrim(typ itemType) {
	lx.items <- item{typ, strings.TrimSpace(lx.current()), lx.line}
	lx.start = lx.pos
}

func (lx *lexer) next() (r rune) {
	if lx.atEOF {
		panic("next called after EOF")
	}
	if lx.pos >= len(lx.input) {
		lx.atEOF = true
		return eof
	}

	if lx.input[lx.pos] == '\n' {
		lx.line++
	}
	lx.prevWidths[2] = lx.prevWidths[1]
	lx.prevWidths[1] = lx.prevWidths[0]
	if lx.nprev < 3 {
		lx.nprev++
	}
	r, w := utf8.DecodeRuneInString(lx.input[lx.pos:])
	lx.prevWidths[0] = w
	lx.pos += w
	return r
}

// ignore skips over the pending input before this point.
func (lx *lexer) ignore() {
	lx.start = lx.pos
}

// backup steps back one rune. Can be called only once per call of next.
func (lx *lexer) backup() {
	if lx.atEOF {
		lx.atEOF = false
		return
	}
	if lx.nprev < 0 {
		panic("backed up too far")
	}
	w := lx.prevWidths[0]
	lx.prevWidths[0] = lx.prevWidths[1]
	lx.prevWidths[1] = lx.prevWidths[2]
	lx.nprev--
	lx.pos -= w
	if lx.pos < len(lx.input) && lx.input[lx.pos] == '\n' {
		lx.line--
	}
}

// peek returns but does not consume the next rune in the input.
func (lx *lexer) peek() rune {
	r := lx.next()
	lx.backup()
	return r
}

// accept consumes the next rune if it's equal to `valid`.
func (lx *lexer) accept(valid rune) bool {
	if lx.next() == valid {
		return true
	}
	lx.backup()
	return false
}

// skip ignores all input that matches the given predicate.
func (lx *lexer) skip(pred func(rune) bool) {
	for {
		r := lx.next()
		if pred(r) {
			continue
		}
		lx.backup()
		lx.ignore()
		return
	}
}

// errorf stops all lexing by emitting an error and returning `nil`.
// Note that any value that is a character is escaped if it's a special
// character (new lines, tabs, etc.).
func (lx *lexer) errorf(format string, values ...interface{}) stateFn {
	for i, value := range values {
		if v, ok := value.(rune); ok {
			values[i] = escapeSpecial(v)
		}
	}
	lx.items <- item{
		itemError,
		fmt.Sprintf(format, values...),
		lx.line,
	}
	return nil
}

// lexTop consumes elements at the top level of data.
func lexTop(lx *lexer) stateFn {
	r := lx.next()
	if r != eof && (isWhitespace(r) || isNL(r)) {
		return lexSkip(lx, lexTop)
	}
	switch r {
	case commentHashStart:
		lx.push(lexTop)
		return lexCommentStart
	case commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexTop)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case eof:
		if lx.pos > lx.start {
			return lx.errorf("Unexpected EOF.")
		}
		lx.emit(itemEOF)
		return nil
	}

	// At this point, the only valid item can be a key, so we back up
	// and let the key lexer do the rest.
	lx.backup()
	lx.push(lexTopEnd)
	return lexKeyStart
}

// lexTopEnd is entered whenever a top-level value has been consumed.
// It must see only whitespace, and will turn back to lexTop upon a new line.
// If it sees EOF, it will quit the lexer successfully.
func lexTopEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == commentHashStart:
		// a comment will read to a new line for us.
		lx.push(lexTop)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexTop)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case isWhitespace(r):
		return lexTopEnd
	case isNL(r) || r == optValTerm:
		lx.ignore()
		return lexTop
	case r == eof:
		lx.emit(itemEOF)
		return nil
	}

	return lx.errorf("Source:-------------\n%v 👈 <------- ❌ Expected a top-level value to end with a new line, comment or EOF, but got '%v' instead.\n", lx.input[0:lx.start]+`[`+lx.input[lx.start:lx.pos]+`]`, r)
}

// lexKeyStart consumes a key name up until the first non-whitespace character.
// lexKeyStart will ignore whitespace. It will also eat enclosing quotes.
func lexKeyStart(lx *lexer) stateFn {
	r := lx.peek()
	switch {
	case isKeySeparator(r):
		return lx.errorf("Unexpected key separator '%v'.", r)
	case isWhitespace(r) || isNL(r):
		// Most likely this is not-reachable, as the lexTop already checks for it.
		lx.next()
		return lexSkip(lx, lexKeyStart)
	case r == dqStringStart:
		lx.next()
		return lexSkip(lx, lexDubQuotedKey)
	case r == sqStringStart:
		lx.next()
		return lexSkip(lx, lexQuotedKey)
	case !isIdentifierRune(r):
		// This is not a valid identity/key rune
		lx.next()
		lx.ignore()
		return lexKeyStart
	case r == eof:
		return lexTop
	}
	lx.ignore()
	lx.next()
	return lexKey
}

// lexDubQuotedKey consumes the text of a key between quotes.
func lexDubQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == dqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexKeyEnd)
	}
	lx.next()
	return lexDubQuotedKey
}

// lexQuotedKey consumes the text of a key between quotes.
func lexQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == sqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexKeyEnd)
	}
	lx.next()
	return lexQuotedKey
}

// lexKey consumes the text of a key. Assumes that the first character (which
// is not whitespace) has already been consumed.
func lexKey(lx *lexer) stateFn {
	r := lx.peek()
	switch {
	case r == eof:
		// Unexpected end, allow lexTop eof/error to handle it
		return lexTop
	case isWhitespace(r) || isNL(r) || isKeySeparator(r):
		lx.emit(itemKey)
		return lexKeyEnd
	}
	lx.next()
	return lexKey
}

// lexKeyEnd consumes the end of a key (up to the key separator).
// Assumes that the first whitespace character after a key (or the '=' or ':'
// separator) has NOT been consumed.
func lexKeyEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r) || isNL(r):
		return lexSkip(lx, lexKeyEnd)
	case isKeySeparator(r):
		return lexSkip(lx, lexValue)
	}
	// We start the value here
	lx.backup()
	return lexValue
}

// lexValue starts the consumption of a value anywhere a value is expected.
// lexValue will ignore whitespace.
// After a value is lexed, the last state on the next is popped and returned.
func lexValue(lx *lexer) stateFn {
	// We allow whitespace to precede a value, but NOT new lines.
	// In array syntax, the array states are responsible for ignoring new lines.
	r := lx.next()
	switch {
	case isWhitespace(r):
		return lexSkip(lx, lexValue)
	case isDigit(r):
		lx.backup() // avoid an extra state and use the same as above
		return lexNumberOrDateStart
	case isNL(r):
		return lx.errorf("Expected value but found new line")
	}

	switch r {
	case arrayStart:
		lx.ignore()
		lx.emit(itemArrayStart)
		lx.isEnd = isEndArrayUnQuoted
		return lexArrayValue
	case mapStart:
		lx.ignore()
		lx.emit(itemMapStart)
		return lexMapKeyStart
	case sqStringStart: //  single quote:   '
		if lx.accept(sqStringStart) {
			if lx.accept(sqStringStart) {
				lx.ignore() // Ignore """
				return lexMultilineRawString
			}
			lx.backup()
		}
		lx.ignore() // ignore the " or '
		return lexQuotedString
	case dqStringStart: // "
		if lx.accept(dqStringStart) {
			if lx.accept(dqStringStart) {
				lx.ignore() // Ignore """
				return lexMultilineString
			}
			lx.backup()
		}
		lx.ignore() // ignore the " or '
		return lexDubQuotedString
	case '-':
		return lexNumberStart
	case blockStart:
		r = lx.next() // ignore the \n after {
		if r == '\r' {
			lx.ignore()
			lx.next()
		}
		lx.ignore() // Ignore the (
		return lexBlock
	case '.': // special error case, be kind to users
		return lx.errorf("Floats must start with a digit, not '.'.")
	}
	// we didn't consume it, so backup
	lx.backup()
	return lexString
	//return lx.errorf("Expected value but found '%s' instead.", r)
}

// lexArrayValue consumes one value in an array. It assumes that '[' or ','
// have already been consumed. All whitespace and new lines are ignored.
func lexArrayValue(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r) || isNL(r):
		return lexSkip(lx, lexArrayValue)
	case r == commentHashStart:
		lx.push(lexArrayValue)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexArrayValue)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case r == arrayValTerm: //   ,  we should not have found comma yet
		return lx.errorf("Unexpected array value terminator '%v'.", arrayValTerm)
	case r == arrayEnd:
		return lexArrayEnd
	}

	lx.backup()
	lx.push(lexArrayValueEnd)
	return lexValue
}

// lexArrayValueEnd consumes the cruft between values of an array. Namely,
// it ignores whitespace and expects either a ',' or a ']'.
func lexArrayValueEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r):
		return lexSkip(lx, lexArrayValueEnd)
	case r == commentHashStart:
		lx.push(lexArrayValueEnd)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexArrayValueEnd)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case r == arrayValTerm || isNL(r):
		return lexSkip(lx, lexArrayValue) // Move onto next
	case r == arrayEnd:
		return lexArrayEnd
	}
	return lx.errorf("Expected an array value terminator %q or an array "+
		"terminator %q, but got '%v' instead.", arrayValTerm, arrayEnd, r)
}

// lexArrayEnd finishes the lexing of an array. It assumes that a ']' has
// just been consumed.
func lexArrayEnd(lx *lexer) stateFn {
	lx.ignore()
	lx.emit(itemArrayEnd)
	lx.isEnd = isEndNormal
	return lx.pop()
}

// lexMapKeyStart consumes a key name up until the first non-whitespace
// character.
// lexMapKeyStart will ignore whitespace.
func lexMapKeyStart(lx *lexer) stateFn {
	r := lx.peek()
	switch {
	case r == eof:
		return lx.errorf("Un terminated map")
	case isKeySeparator(r):
		return lx.errorf("Unexpected key separator '%v'.", r)
	case isWhitespace(r) || isNL(r):
		lx.next()
		return lexSkip(lx, lexMapKeyStart)
	case r == mapEnd:
		lx.next()
		return lexSkip(lx, lexMapEnd)
	case r == commentHashStart:
		lx.next()
		lx.push(lexMapKeyStart)
		return lexCommentStart
	case r == commentSlashStart:
		lx.next()
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexMapKeyStart)
			return lexCommentStart
		}
		lx.backup()
	case r == sqStringStart:
		lx.next()
		return lexSkip(lx, lexMapQuotedKey)
	case r == dqStringStart:
		lx.next()
		return lexSkip(lx, lexMapDubQuotedKey)
	}
	lx.ignore()
	lx.next()
	return lexMapKey
}

// lexMapQuotedKey consumes the text of a key between quotes.
func lexMapQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == sqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexMapKeyEnd)
	}
	lx.next()
	return lexMapQuotedKey
}

// lexMapQuotedKey consumes the text of a key between quotes.
func lexMapDubQuotedKey(lx *lexer) stateFn {
	r := lx.peek()
	if r == dqStringEnd {
		lx.emit(itemKey)
		lx.next()
		return lexSkip(lx, lexMapKeyEnd)
	}
	lx.next()
	return lexMapDubQuotedKey
}

// lexMapKey consumes the text of a key. Assumes that the first character (which
// is not whitespace) has already been consumed.
func lexMapKey(lx *lexer) stateFn {
	r := lx.peek()
	if isWhitespace(r) || isNL(r) || isKeySeparator(r) {
		lx.emit(itemKey)
		return lexMapKeyEnd
	}
	lx.next()
	return lexMapKey
}

// lexMapKeyEnd consumes the end of a key (up to the key separator).
// Assumes that the first whitespace character after a key (or the '='
// separator) has NOT been consumed.
func lexMapKeyEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r) || isNL(r):
		return lexSkip(lx, lexMapKeyEnd)
	case isKeySeparator(r):
		return lexSkip(lx, lexMapValue)
	}
	// We start the value here
	lx.backup()
	return lexMapValue
}

// lexMapValue consumes one value in a map. It assumes that '{' or ','
// have already been consumed. All whitespace and new lines are ignored.
// Map values can be separated by ',' or simple NLs.
func lexMapValue(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r) || isNL(r):
		return lexSkip(lx, lexMapValue)
	case r == commentHashStart:
		lx.push(lexMapValue)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexMapValue)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case r == mapValTerm:
		return lx.errorf("Unexpected map value terminator %q.", mapValTerm)
	case r == mapEnd:
		return lexSkip(lx, lexMapEnd)
	}
	lx.backup()
	lx.push(lexMapValueEnd)
	return lexValue
}

// lexMapValueEnd consumes the cruft between values of a map. Namely,
// it ignores whitespace and expects either a ',' or a '}'.
func lexMapValueEnd(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isWhitespace(r):
		return lexSkip(lx, lexMapValueEnd)
	case r == commentHashStart:
		lx.push(lexMapValueEnd)
		return lexCommentStart
	case r == commentSlashStart:
		rn := lx.next()
		if rn == commentSlashStart {
			lx.push(lexMapValueEnd)
			return lexCommentStart
		}
		lx.backup()
		fallthrough
	case r == optValTerm || r == mapValTerm || isNL(r):
		return lexSkip(lx, lexMapKeyStart) // Move onto next
	case r == mapEnd:
		return lexSkip(lx, lexMapEnd)
	}
	return lx.errorf("Expected a map value terminator %q or a map "+
		"terminator %q, but got '%v' instead.", mapValTerm, mapEnd, r)
}

// lexMapEnd finishes the lexing of a map. It assumes that a '}' has
// just been consumed.
func lexMapEnd(lx *lexer) stateFn {
	lx.ignore()
	lx.emit(itemMapEnd)
	return lx.pop()
}

// Checks if the unquoted string was actually a boolean
func (lx *lexer) isBool() bool {
	str := lx.input[lx.start:lx.pos]
	str = strings.ToLower(str)
	return str == "true" || str == "false"
}

// lexQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexQuotedString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == sqStringEnd:
		lx.backup()
		lx.emit(itemString)
		lx.next()
		lx.ignore()
		return lx.pop()
	}
	return lexQuotedString
}

// lexDubQuotedString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored. It will not interpret any
// internal contents.
func lexDubQuotedString(lx *lexer) stateFn {
	r := lx.next()
	switch r {
	case dqStringEnd:
		lx.backup()
		lx.emit(itemString)
		lx.next()
		lx.ignore()
		return lx.pop()
	}
	return lexDubQuotedString
}

// lexString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored.
func lexString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == backslash:
		return lexStringEscape
	// Termination of non-quoted strings
	case lx.isEnd(lx, r):
		lx.backup()
		if lx.isBool() {
			lx.emit(itemBool)
		} else {
			lx.emit(itemString)
		}
		return lx.pop()
	case r == sqStringEnd:
		lx.backup()
		lx.emit(itemString)
		lx.next()
		lx.ignore()
		return lx.pop()
	}
	return lexString
}

// lexDubString consumes the inner contents of a string. It assumes that the
// beginning '"' has already been consumed and ignored.
func lexDubString(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == backslash:
		return lexStringEscape
	// Termination of non-quoted strings
	case lx.isEnd(lx, r):
		lx.backup()
		if lx.isBool() {
			lx.emit(itemBool)
		} else {
			lx.emit(itemString)
		}
		return lx.pop()
	case r == dqStringEnd:
		lx.backup()
		lx.emit(itemString)
		lx.next()
		lx.ignore()
		return lx.pop()
	}
	return lexDubString
}

// lexMultilineString consumes the inner contents of a string. It assumes that
// the beginning '"""' has already been consumed and ignored.
func lexMultilineString(lx *lexer) stateFn {
	switch lx.next() {
	case eof:
		return lx.errorf("unexpected EOF")
	case backslash:
		return lexMultilineStringEscape
	case dqStringEnd:
		if lx.accept(dqStringEnd) {
			if lx.accept(dqStringEnd) {
				lx.backup()
				lx.backup()
				lx.backup()
				lx.emit(itemString)
				lx.next()
				lx.next()
				lx.next()
				lx.ignore()
				return lx.pop()
			}
			lx.backup()
		}
	}
	return lexMultilineString
}

// lexMultilineRawString consumes a raw string. Nothing can be escaped in such
// a string. It assumes that the beginning "'''" has already been consumed and
// ignored.
func lexMultilineRawString(lx *lexer) stateFn {
	switch lx.next() {
	case eof:
		return lx.errorf("unexpected EOF")
	case sqStringEnd:
		if lx.accept(sqStringEnd) {
			if lx.accept(sqStringEnd) {
				lx.backup()
				lx.backup()
				lx.backup()
				lx.emit(itemString)
				lx.next()
				lx.next()
				lx.next()
				lx.ignore()
				return lx.pop()
			}
			lx.backup()
		}
	}
	return lexMultilineRawString
}

// lexMultilineStringEscape consumes an escaped character. It assumes that the
// preceding '\\' has already been consumed.
func lexMultilineStringEscape(lx *lexer) stateFn {
	// Handle the special case first:
	if isNL(lx.next()) {
		return lexMultilineString
	}
	lx.backup()
	lx.push(lexMultilineString)
	return lexStringEscape(lx)
}

// lexBlock consumes the inner contents as a string. It assumes that the
// beginning '(' has already been consumed and ignored. It will continue
// processing until it finds a ')' on a new line by itself.
func lexBlock(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case r == blockEnd:

		lx.backup() // unconsume )  we are going to verify below
		lx.backup() // unconsume previous rune to ensure it is newline

		// Looking for a ')' character on a line by itself, if the previous
		// character isn't a new line, then break so we keep processing the block.
		if !isNL(lx.next()) {
			lx.next() // if inline ( this will consume it
			break
		}
		lx.next()
		// Make sure the next character is a new line or an eof. We want a ')' on a
		// bare line by itself.
		switch r = lx.next(); r {
		case '\r', '\n', eof:
			if r == eof {
				lx.prevWidths[0] = 1
			}
			lx.backup() // unconsume the \n, or eof
			r = lx.peek()
			if r != ')' { // For some reason on EOF we aren't where we think
				lx.backup() // unconsume the )
			}
			lx.backup() // unconsume the \n
			lx.backup()
			if lx.next() == '\r' {
				lx.backup()
			}
			lx.emit(itemString)
			lx.next() // consume the previous line \n
			lx.next() // consume the )
			lx.ignore()
			return lx.pop()
		}
		lx.backup()
	}
	return lexBlock
}

// lexStringEscape consumes an escaped character. It assumes that the preceding
// '\\' has already been consumed.
func lexStringEscape(lx *lexer) stateFn {
	r := lx.next()
	switch r {
	case 'x':
		return lexStringBinary
	case 'b':
		fallthrough
	case 't':
		fallthrough
	case 'n':
		fallthrough
	case 'f':
		fallthrough
	case 'r':
		fallthrough
	case '"':
		fallthrough
	case backslash:
		return lx.pop()
	case 'u':
		return lexShortUnicodeEscape
	case 'U':
		return lexLongUnicodeEscape
	}
	return lx.errorf("invalid escape character %q; only the following "+
		"escape characters are allowed: "+
		`\xXXX, \b, \t, \n, \f, \r, \", \\, \uXXXX, and \UXXXXXXXX`, r)
}

func lexShortUnicodeEscape(lx *lexer) stateFn {
	var r rune
	for i := 0; i < 4; i++ {
		r = lx.next()
		if !isHexadecimal(r) {
			return lx.errorf(`expected four hexadecimal digits after '\u', `+
				"but got %q instead", lx.current())
		}
	}
	return lx.pop()
}

func lexLongUnicodeEscape(lx *lexer) stateFn {
	var r rune
	for i := 0; i < 8; i++ {
		r = lx.next()
		if !isHexadecimal(r) {
			return lx.errorf(`expected eight hexadecimal digits after '\U', `+
				"but got %q instead", lx.current())
		}
	}
	return lx.pop()
}

// lexStringBinary consumes two hexadecimal digits following '\x'. It assumes
// that the '\x' has already been consumed.
func lexStringBinary(lx *lexer) stateFn {
	r := lx.next()
	if !isHexadecimal(r) {
		return lx.errorf("Expected two hexadecimal digits after '\\x', but "+
			"got '%v' instead.", r)
	}

	r = lx.next()
	if !isHexadecimal(r) {
		return lx.errorf("Expected two hexadecimal digits after '\\x', but "+
			"got '%v' instead.", r)
	}
	return lexString
}

// lexNumberOrDateStart consumes either a (positive) integer, float or datetime.
// It assumes that NO negative sign has been consumed.
func lexNumberOrDateStart(lx *lexer) stateFn {
	r := lx.next()
	if isDigit(r) {
		return lexNumberOrDate
	}
	switch r {
	case '_':
		return lexNumber
	case 'e', 'E':
		return lexFloat
	case '.':
		return lx.errorf("floats must start with a digit, not '.'")
	}
	return lx.errorf("expected a digit but got %q", r)
}

// lexNumberOrDate consumes either a (positive) integer, float or datetime.
func lexNumberOrDate(lx *lexer) stateFn {
	r := lx.next()

	if isDigit(r) {
		return lexNumberOrDate
	}
	switch r {
	case '-':
		return lexDatetime
	case '_':
		return lexNumber
	case '.', 'e', 'E':
		return lexFloat
	}
	lx.backup()
	lx.emit(itemInteger)
	return lx.pop()
}

// lexDatetime consumes a Datetime, to a first approximation.
// The parser validates that it matches one of the accepted formats.
func lexDatetime(lx *lexer) stateFn {
	r := lx.next()
	if isDigit(r) {
		return lexDatetime
	}
	switch r {
	case '-', 'T', ':', '.', 'Z':
		return lexDatetime
	}

	lx.backup()
	lx.emit(itemDatetime)
	return lx.pop()
}

// lexDateAfterYear consumes a full Zulu Datetime in ISO8601 format.
// It assumes that "YYYY-" has already been consumed.
func lexDateAfterYear(lx *lexer) stateFn {
	formats := []rune{
		// digits are '0'.
		// everything else is direct equality.
		'0', '0', '-', '0', '0',
		'T',
		'0', '0', ':', '0', '0', ':', '0', '0',
		'Z',
	}
	for _, f := range formats {
		r := lx.next()
		if f == '0' {
			if !isDigit(r) {
				return lx.errorf("Expected digit in ISO8601 datetime, "+
					"but found '%v' instead.", r)
			}
		} else if f != r {
			return lx.errorf("Expected '%v' in ISO8601 datetime, "+
				"but found '%v' instead.", f, r)
		}
	}
	lx.emit(itemDatetime)
	return lx.pop()
}

// lexNumberStart consumes either an integer or a float. It assumes that a
// negative sign has already been read, but that *no* digits have been consumed.
// lexNumberStart will move to the appropriate integer or float states.
func lexNumberStart(lx *lexer) stateFn {
	// we MUST see a digit. Even floats have to start with a digit.
	r := lx.next()
	if !isDigit(r) {
		if r == '.' {
			return lx.errorf("Floats must start with a digit, not '.'.")
		}
		return lx.errorf("Expected a digit but got '%v'.", r)
	}
	return lexNumber
}

// lexNumber consumes an integer or a float after seeing the first digit.
func lexNumber(lx *lexer) stateFn {
	r := lx.next()
	switch {
	case isDigit(r):
		return lexNumber
	case r == '.':
		return lexFloatStart
	}

	lx.backup()
	lx.emit(itemInteger)
	return lx.pop()
}

// lexFloatStart starts the consumption of digits of a float after a '.'.
// Namely, at least one digit is required.
func lexFloatStart(lx *lexer) stateFn {
	r := lx.next()
	if !isDigit(r) {
		return lx.errorf("Floats must have a digit after the '.', but got "+
			"'%v' instead.", r)
	}
	return lexFloat
}

// lexFloat consumes the digits of a float after a '.'.
// Assumes that one digit has been consumed after a '.' already.
func lexFloat(lx *lexer) stateFn {
	r := lx.next()
	if isDigit(r) {
		return lexFloat
	}

	lx.backup()
	lx.emit(itemFloat)
	return lx.pop()
}

// lexCommentStart begins the lexing of a comment. It will emit
// itemCommentStart and consume no characters, passing control to lexComment.
func lexCommentStart(lx *lexer) stateFn {
	lx.ignore()
	lx.emit(itemCommentStart)
	return lexComment
}

// lexComment lexes an entire comment. It assumes that '#' has been consumed.
// It will consume *up to* the first new line character, and pass control
// back to the last state on the stack.
func lexComment(lx *lexer) stateFn {
	r := lx.peek()
	if isNL(r) || r == eof {
		lx.emit(itemText)
		return lx.pop()
	}
	lx.next()
	return lexComment
}

// lexSkip ignores all slurped input and moves on to the next state.
func lexSkip(lx *lexer, nextState stateFn) stateFn {
	return func(lx *lexer) stateFn {
		lx.ignore()
		return nextState
	}
}

func isEndNormal(lx *lexer, r rune) bool {
	return (isNL(r) || r == eof || r == optValTerm || isWhitespace(r))
}

func isEndArrayUnQuoted(lx *lexer, r rune) bool {
	return (isNL(r) || r == eof || r == optValTerm || r == arrayEnd || r == arrayValTerm || isWhitespace(r))
}

func isIdentifierRune(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) {
		return true
	}
	for _, allowedRune := range IdentityChars {
		if allowedRune == r {
			return true
		}
	}
	return false
}

// Tests for both key separators
func isKeySeparator(r rune) bool {
	return r == keySepEqual || r == keySepColon
}

// isWhitespace returns true if `r` is a whitespace character according
// to the spec.
func isWhitespace(r rune) bool {
	return r == '\t' || r == ' '
}

func isNL(r rune) bool {
	return r == '\n' || r == '\r'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isHexadecimal(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}

func (item item) String() string {
	return fmt.Sprintf("(%T, '%s', %d)", item.typ, item.val, item.line)
}

func escapeSpecial(c rune) string {
	switch c {
	case '\n':
		return "\\n"
	}
	return string(c)
}
