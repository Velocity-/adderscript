package main

type tokenType int
type char uint8

const eof char = 255

const (
	tokenEOF            tokenType = iota
	tokenComment         // Simple single or multi-line comment
	tokenIdentifier      // Any identifier that is not a keyword
	tokenOn              // The keyword 'on', indicating a trigger
	tokenFunc            // The 'func' keyword indicating a new method
	tokenIf
	tokenElse
	tokenLParen                   // Left parenthesis '('
	tokenRParen                   // Right parenthesis ')'
	tokenInteger                 // An integer literal
	tokenSemicolon             // A semicolon ';'
	tokenLBrack
	tokenRBrack
	tokenAssign
	tokenString                   // A string literal
	tokenBool
	tokenComma
	tokenMinus
	tokenEqual                     // ==
	tokenNotEqual               // !=
	tokenLessThan               // <
	tokenGreaterThan         // >
	tokenLessOrEqual         // <=
	tokenGreaterOrEqual   // >=
	tokenNot             // !
	tokenPlus
	tokenDivide
	tokenMultiply
	tokenModulo
)

type scanAction func(*scanner) scanAction

type token struct {
	tokenType tokenType
	value     string
	from      int
	to        int
}

type scanner struct {
	data    string
	pos     int
	prevpos int
	mark    int
	tokens  chan token
	state   scanAction
}

func (s *scanner) markPosition(offset int) {
	s.mark = s.pos
	s.mark += offset
}

func (s *scanner) current() char {
	if s.pos >= len(s.data) {
		return eof
	}

	return char(s.data[s.pos])
}

func (s *scanner) rewind(num int) {
	s.pos -= num
}

func (s *scanner) peek(num int) char {
	return char(s.data[s.pos+num])
}

func (s *scanner) next() char {
	if s.pos >= len(s.data) {
		return eof
	}

	r := s.data[s.pos]
	s.pos++
	return char(r)
}

func (s *scanner) value() string {
	return s.data[s.mark:s.pos]
}

func (s *scanner) makeToken(t tokenType) {
	newToken := token{
		tokenType: t,
		value:     s.value(),
		from:      s.mark,
		to:        s.pos,
	}

	if newToken.tokenType != tokenComment {
		s.tokens <- newToken
	}
}

func scanAny(s *scanner) scanAction {
	c := s.current()
	if c == eof {
		s.makeToken(tokenEOF)
		return nil
	}

	for {
		if c == 0 || c == eof {
			s.makeToken(tokenEOF)
			return nil
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			s.next()
			c = s.current()
			continue
		}

		break
	}

	s.markPosition(0)

	if c == '/' {
		s.next()
		c = s.peek(0)

		if c == '/' {
			s.next() // consume '/'
			return scanSingleComment
		} else if c == '*' {
			s.next()
			return scanMultilineComment
		} else {
			s.makeToken(tokenDivide)
			return scanAny
		}
	} else if c == '"' {
		return scanString
	} else if c == '(' {
		s.next()
		s.makeToken(tokenLParen)
		return scanAny
	} else if c == ')' {
		s.next()
		s.makeToken(tokenRParen)
		return scanAny
	} else if c == '{' {
		s.next()
		s.makeToken(tokenLBrack)
		return scanAny
	} else if c == '}' {
		s.next()
		s.makeToken(tokenRBrack)
		return scanAny
	} else if c == ';' {
		s.next()
		s.makeToken(tokenSemicolon)
		return scanAny
	} else if c == '=' {
		s.next()
		c = s.peek(0)

		if c == '=' {
			s.next()
			s.makeToken(tokenEqual)
		} else {
			s.makeToken(tokenAssign)
		}

		return scanAny
	} else if c == '!' {
		c = s.next()

		if c == '=' {
			s.next()
			s.makeToken(tokenNotEqual)
		} else {
			s.makeToken(tokenNot)
		}

		return scanAny
	} else if c == '<' {
		c = s.next()

		if c == '=' {
			s.next()
			s.makeToken(tokenLessOrEqual)
		} else {
			s.makeToken(tokenLessThan)
		}

		return scanAny
	} else if c == '>' {
		c = s.next()

		if c == '=' {
			s.next()
			s.makeToken(tokenGreaterOrEqual)
		} else {
			s.makeToken(tokenGreaterThan)
		}

		return scanAny
	} else if c == ',' {
		s.next()
		s.makeToken(tokenComma)
		return scanAny
	} else if c == '-' {
		s.next()

		if isIntegerChar(s.current()) {
			return scanIntegerLiteral
		} else {
			s.makeToken(tokenMinus)
			return scanAny
		}
	} else if c == '+' {
		s.next()
		s.makeToken(tokenPlus)
		return scanAny
	} else if c == '*' {
		s.next()
		s.makeToken(tokenMultiply)
		return scanAny
	} else if c == '%' {
		s.next()
		s.makeToken(tokenModulo)
		return scanAny
	}

	if isStartOfIdentifier(c) {
		return scanIdentifier
	} else if isIntegerChar(c) {
		return scanIntegerLiteral
	}

	s.rewind(1)
	fragmentEnd := s.pos + 10
	if fragmentEnd > len(s.data) {
		fragmentEnd = len(s.data)
	}

	panic("unknown char: " + string(c) + "; data: " + s.data[s.pos:fragmentEnd])
	return nil
}

func scanSingleComment(s *scanner) scanAction {
	for {
		c := s.next()

		if c == '\n' || c == eof {
			s.rewind(1)
			break
		}
	}

	s.makeToken(tokenComment)
	return scanAny
}

func scanMultilineComment(s *scanner) scanAction {
	for {
		c := s.next()

		if c == '*' && s.peek(0) == '/' {
			s.next(); s.next() // consume both * and /
			break
		} else if c == eof {
			s.rewind(1)
			break
		}
	}

	s.makeToken(tokenComment)
	return scanAny
}

func scanString(s *scanner) scanAction {
	c := s.next()

	for {
		c = s.next()
		if c == '"' { // TODO EOL/EOF
			break
		}
	}

	s.makeToken(tokenString)
	return scanAny
}

func isStartOfIdentifier(c char) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentifierChar(c char) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func isIntegerChar(c char) bool {
	return c >= '0' && c <= '9'
}

func scanIdentifier(s *scanner) scanAction {
	for {
		c := s.next()
		if !isIdentifierChar(c) {
			s.rewind(1)
			break
		}
	}

	value := s.value()
	if value == "on" {
		s.makeToken(tokenOn)
	} else if value == "func" {
		s.makeToken(tokenFunc)
	} else if value == "if" {
		s.makeToken(tokenIf)
	} else if value == "else" {
		s.makeToken(tokenElse)
	} else if value == "true" || value == "false" {
		s.makeToken(tokenBool)
	} else {
		s.makeToken(tokenIdentifier)
	}

	return scanAny
}

func scanIntegerLiteral(s *scanner) scanAction {
	if s.current() == '-' {
		s.next()
	}

	for {
		c := s.next()
		if !isIntegerChar(c) {
			s.rewind(1)
			break
		}
	}

	s.makeToken(tokenInteger)
	return scanAny
}

func (s *scanner) run() {
	for s.state = scanAny; s.state != nil; {
		s.state = s.state(s)
	}

	close(s.tokens)
}

func ScanText(text string) []token {
	s := scanner{
		data:   text,
		tokens: make(chan token),
	}

	go s.run()

	tokens := make([]token, 512)[:0]
	for v := range s.tokens {
		tokens = append(tokens, v)
	}

	return tokens
}
