package diagnostics

import (
	"fmt"
	"strings"

	"github.com/gearsdatapacks/libra/lexer/token"
)

type Manager struct {
	Diagnostics []Diagnostic
	file        string
	lines       []string
}

func New(file string, src string) Manager {
	return Manager{
		Diagnostics: []Diagnostic{},
		file:        file,
		lines:       strings.Split(src, "\n"),
	}
}

func (m *Manager) reportError(msg string, span token.Span) {
	m.Diagnostics = append(m.Diagnostics, new(Error, msg, span, m.file, m.lines))
}

func (m *Manager) reportInfo(msg string, span token.Span) {
	m.Diagnostics = append(m.Diagnostics, new(Info, msg, span, m.file, m.lines))
}

// Lexer Diagnostics

func (m *Manager) ReportInvalidCharacter(span token.Span, char byte) {
	msg := fmt.Sprintf("Invalid character: %q", char)
	m.reportError(msg, span)
}

func (m *Manager) ReportUnterminatedString(span token.Span) {
	msg := "Unterminated string"
	m.reportError(msg, span)
}

func (m *Manager) ReportInvalidEscapeSequence(span token.Span, char byte) {
	msg := fmt.Sprintf("Invalid escape sequence: '\\%c'", char)
	m.reportError(msg, span)
}

func (m *Manager) ReportNumbersCannotEndWithSeparator(span token.Span) {
	msg := "Numbers cannot end with numeric separators"
	m.reportError(msg, span)
}


// Parser Diagnostics

func (m *Manager) ReportExpectedExpression(span token.Span, kind token.Kind) {
	msg := fmt.Sprintf("Expected expression, found %s", kind.String())
	m.reportError(msg, span)
}

func (m *Manager) ReportExpectedNewline(span token.Span, kind token.Kind) {
	msg := fmt.Sprintf("Expected newline after statement, found %s", kind.String())
	m.reportError(msg, span)
}

func (m *Manager) ReportExpectedToken(span token.Span, expected token.Kind, actual token.Kind) {
	msg := fmt.Sprintf("Expected %s, found %s", expected.String(), actual.String())
	m.reportError(msg, span)
}

func (m *Manager) ReportElseStatementWithoutIf(span token.Span) {
	msg := "Else statement not allowed without preceding if"
	m.reportError(msg, span)
}

func (m *Manager) ReportExpectedKeyword(span token.Span, keyword string, foundToken token.Token) {
	tokenValue := foundToken.Kind.String()
	if foundToken.Kind == token.IDENTIFIER {
		tokenValue = foundToken.Value
	}

	msg := fmt.Sprintf("Expected %q keyword, found %s", keyword, tokenValue)
	m.reportError(msg, span)
}

func (m *Manager) ReportKeywordOverwritten(span token.Span, keyword string, declared token.Span) {
	errMsg := fmt.Sprintf(
		"Expected %q keyword, but it has been overwritten by a variable",
		keyword)
		info := "Try removing or renaming this variable"

	m.reportError(errMsg, span)
	m.reportInfo(info, declared)
}
