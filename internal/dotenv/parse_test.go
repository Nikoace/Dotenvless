package dotenv

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseSupportedSyntaxWithoutExpansion(t *testing.T) {
	input := "\ufeff# comment\r\nexport token = plain # comment\r\nEMPTY=\r\nSINGLE='C:\\temp #literal'\nDOUBLE=\"hello\\nworld\\t\\\"quoted\\\"\"\nMULTI='first\nsecond'\nLITERAL=${HOME}/$(command)\nHASH=value#literal\n"
	values, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"TOKEN": "plain", "EMPTY": "", "SINGLE": "C:\\temp #literal", "DOUBLE": "hello\nworld\t\"quoted\"", "MULTI": "first\nsecond", "LITERAL": "${HOME}/$(command)", "HASH": "value#literal"}
	if len(values) != len(expected) {
		t.Fatal("wrong number of variables")
	}
	for key, value := range expected {
		if string(values[key]) != value {
			t.Errorf("wrong parsed value for %s", key)
		}
	}
}
func TestParseRejectsMalformedWithoutValueLeak(t *testing.T) {
	for _, input := range []string{
		"BROKEN_FAKE_SECRET", "=FAKE_SECRET", "A=x\na=FAKE_SECRET", "A=\"FAKE_SECRET", "A='FAKE_SECRET' junk",
		"A=\"FAKE_SECRET\\q\"", "A=FAKE_SECRET\x00", "A=\xff", "A=" + strings.Repeat("x", 32769),
	} {
		if _, err := Parse(strings.NewReader(input)); err == nil {
			t.Fatal("malformed input accepted")
		} else if strings.Contains(err.Error(), "FAKE_SECRET") {
			t.Fatal("source value leaked")
		}
	}
	if _, err := Parse(bytes.NewReader(bytes.Repeat([]byte{'#'}, (1<<20)+1))); err == nil {
		t.Fatal("oversized source accepted")
	}
}

func TestMultilineQuotedValuesPreserveFirstLineWhitespace(t *testing.T) {
	for _, tc := range []struct{ name, input string }{
		{"single", "TOKEN='FAKE  \t\nNEXT'\n"},
		{"double", "TOKEN=\"FAKE  \t\nNEXT\"\n"},
		{"export-single", " export TOKEN='FAKE  \t\nNEXT'\n"},
		{"export-double-crlf", "export\tTOKEN=\"FAKE  \t\r\nNEXT\"\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values, err := Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(values["TOKEN"]) != "FAKE  \t\nNEXT" {
				t.Fatal("quoted value lost whitespace")
			}
		})
	}
	values, err := Parse(strings.NewReader(" \t\n export PLAIN = value \t\n"))
	if err != nil || string(values["PLAIN"]) != "value" {
		t.Fatal("unquoted whitespace handling changed")
	}
}
