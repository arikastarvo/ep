package parser_test

import(
	"ep/parser"
	"testing"
)

func TestParserFromBytesOneSimpleEvent(t *testing.T) {
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

func TestParserFromBytesMultipleSimpleEvent(t *testing.T) {
	definition := 
`
first-event: ^%{DATA:field}$
second-event: ^%{DATA:field}$
third-event: ^%{DATA:field}$
`
	p := parser.ParserFromBytes([]byte(definition))
	
	if _, ok := p.Patterns["first-event"]; ! ok {
		t.Error("no event type 'event' found")
	}
	
	if len(p.Patterns) != 3 {
		t.Error("there should be exactly 1 event type defined")
	}

	if len(p.Patterns["first-event"].Fields) != 1 {
		t.Error("there should be exactly 1 field in `event` event type")
	}
	
	if len(p.Patterns["third-event"].Fields) != 1 {
		t.Error("there should be exactly 1 field in `event` event type")
	}
}

func TestParserFromBytesSimpleEventWithMultiplePatterns(t *testing.T) {
	definition := 
`
event:
 - ^%{INT:int}\t%{DATA:string}$
 - ^%{NUMBER:numeric}\t%{DATA:string}$
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}
	
	if len(p.Patterns) != 1 {
		t.Error("there should be exactly 1 event type defined")
	}

	if len(p.Patterns["event"].Fields) != 3 {
		t.Error("there should be exactly 3 fields in `event` event type")
	}
}

func TestParserFromBytesSimpleEventWithMultiplePatterns2(t *testing.T) {
	definition := 
`
event:
  pattern:
   - ^%{INT:int}\t%{DATA:string}$
   - ^%{NUMBER:numeric}\t%{DATA:string}$
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}
	
	if len(p.Patterns) != 1 {
		t.Error("there should be exactly 1 event type defined")
	}

	if len(p.Patterns["event"].Pattern) != 2 {
		t.Error("there should be exactly 2 patterns in `event` event type")
	}

	if len(p.Patterns["event"].Fields) != 3 {
		t.Error("there should be exactly 3 fields in `event` event type")
	}
}


func TestParserFromBytesSimpleEventWithMultiplePatternsAndOptionalPatterns(t *testing.T) {
	definition := 
`
event:
  pattern:
   - ^%{INT:int}\t%{DATA:string}$
   - ^%{NUMBER:numeric}\t%{DATA:string}$
  optionalpattern:
   - "%{NUMBER:numeric}"
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}
	
	if len(p.Patterns["event"].Optionalpattern) != 1 {
		t.Error("there should be exactly 1 optionalpattern in `event` event type")
	}

	if len(p.Patterns["event"].Fields) != 3 {
		t.Error("there should be exactly 3 field in `event` event type")
	}
}


func TestParserFromBytesComplexEvent(t *testing.T) {
	definition := 
`
event:
  pattern:
   - ^%{INT:int}\t%{DATA:string}$
   - ^%{NUMBER:numeric}\t%{DATA:string}$
  optionalpattern:
   - "%{NUMBER:numeric}"
  grokpattern:
    CUSTOM: ([1-9]|10)
  order: 2
  field: source-field
  cond:
    int: 10
  softcond:
    numeric: 10
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}

	if len(p.Patterns["event"].Pattern) != 2 {
		t.Error("there should be exactly 2 patterns in `event` event type")
	}

	if len(p.Patterns["event"].Optionalpattern) != 1 {
		t.Error("there should be exactly 1 optionalpattern in `event` event type")
	}
	
	if len(p.Patterns["event"].Grokpattern) != 2 {
		t.Error("there should be exactly 2 grokpattern in `event` event type")
	}

	if p.Patterns["event"].Order != 2 {
		t.Error("Order for `event` event type should be 2")
	}
	
	if p.Patterns["event"].Field != "source-field" {
		t.Error("Field for `event` event type should be 'source-field'")
	}
	
	if len(p.Patterns["event"].Cond) != 1 {
		t.Error("There should be exactly 1 Cond in `event` event type")
	}
	
	if len(p.Patterns["event"].Softcond) != 1 {
		t.Error("There should be exactly 1 Softcond in `event` event type")
	}
	
	if len(p.Patterns["event"].Fields) != 3 {
		t.Error("there should be exactly 3 field in `event` event type")
	}
}


func TestReplace(t *testing.T) {
	definition := 
`
event:
  pattern: ^%{INT:field_1}\t%{DATA:field_2}$
  replace:
    - field: field_1
      pattern: 1
      replace: 0
    - field: field_2
      pattern: value
      replace: 'altered value'
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}

	if len(p.Patterns["event"].Replace) != 2 {
		t.Error("there should be exactly 2 replace definition in `event` event type")
	}

	result := p.ParseLine("1\tvalue")
	if _, ok := result["field_1"]; ! ok {
		t.Error("no field_1 found in result")
	}

	if result["field_1"] != "0" {
		t.Error("field_1 value must be 0, is:", result["field_1"])
	}
	
	if _, ok := result["field_2"]; ! ok {
		t.Error("no field_2 found in result")
	}
	
	if result["field_2"] != "altered value" {
		t.Error("field_2 value must be 'altered value', is:", result["field_2"])
	}
}

func TestNoPattern(t *testing.T) {
	definition := 
`
event:
  field: data
`
	p := parser.ParserFromBytes([]byte(definition))

	if _, ok := p.Patterns["event"]; ! ok {
		t.Error("no event type 'event' found")
	}

	if len(p.Patterns["event"].Pattern) != 0 {
		t.Error("there should be exactly 0 Patterns in `event` event type")
	}

	result := p.ParseLine("any value")

	if result["data"] != "any value" {
		t.Error("data value must be 'any value', is:", result["data"])
	}
}