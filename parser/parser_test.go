package parser_test

import(
	"ep/parser"
	"testing"
)

func TestParserFromBytesSimple(t *testing.T) {
	p := parser.ParserFromBytes([]byte(`event: ^%{DATA:field}$`))
	
	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}
	
	if len(p.Patterns) != 1 {
		t.Error("there should be exactly 1 event type defined")
	}

	if len(p.Patterns["event"].Fields) != 1 {
		t.Error("there should be exactly 1 field in `event` event type")
	}
}