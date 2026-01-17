package parser

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func TestParseSimpleString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "valid simple string",
			input: "+OK\r\n",
			want:  &resp.SimpleString{Val: []byte("OK")},
		},
		{
			name:  "empty simple string",
			input: "+\r\n",
			want:  &resp.SimpleString{Val: []byte("")},
		},
		{
			name:  "simple string with spaces",
			input: "+Hello World\r\n",
			want:  &resp.SimpleString{Val: []byte("Hello World")},
		},
		{
			name:    "missing CRLF",
			input:   "+OK\n",
			wantErr: true,
		},
		{
			name:    "missing LF",
			input:   "+OK\r",
			wantErr: true,
		},
		{
			name:    "incomplete message",
			input:   "+OK",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '+' prefix
			got, err := handleSimpleString(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleSimpleString() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleSimpleString() = %v, want %v",
					got, tt.want)
			}
		})
	}
}

func TestParseBulkString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "valid bulk string",
			input: "$5\r\nhello\r\n",
			want:  &resp.BulkString{Size: 5, Str: []byte("hello")},
		},
		{
			name:  "empty bulk string",
			input: "$0\r\n\r\n",
			want:  &resp.BulkString{Size: 0, Str: []byte("")},
		},
		{
			name:  "null bulk string",
			input: "$-1\r\n",
			want:  &resp.BulkString{Size: -1, Str: nil},
		},
		{
			name:  "bulk string with special characters",
			input: "$11\r\nhello\nworld\r\n",
			want:  &resp.BulkString{Size: 11, Str: []byte("hello\nworld")},
		},
		{
			name:    "invalid size",
			input:   "$-2\r\n",
			wantErr: true,
		},
		{
			name:    "missing trailing CRLF",
			input:   "$5\r\nhello",
			wantErr: true,
		},
		{
			name:    "invalid trailing CRLF",
			input:   "$5\r\nhello\n\n",
			wantErr: true,
		},
		{
			name:    "size mismatch",
			input:   "$10\r\nhello\r\n",
			wantErr: true,
		},
		{
			name:    "invalid size format",
			input:   "$abc\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '$' prefix
			got, err := handleBulkString(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleBulkString() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleBulkString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSimpleError(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "valid simple error",
			input: "-Error message\r\n",
			want:  &resp.SimpleError{Val: []byte("Error message")},
		},
		{
			name:  "error with ERR prefix",
			input: "-ERR unknown command\r\n",
			want:  &resp.SimpleError{Val: []byte("ERR unknown command")},
		},
		{
			name:  "empty error message",
			input: "-\r\n",
			want:  &resp.SimpleError{Val: []byte("")},
		},
		{
			name:    "missing CRLF",
			input:   "-Error\n",
			wantErr: true,
		},
		{
			name:    "incomplete message",
			input:   "-Error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '-' prefix
			got, err := handleSimpleError(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleSimpleError() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleSimpleError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInteger(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "positive integer",
			input: ":42\r\n",
			want:  &resp.Integer{Val: 42},
		},
		{
			name:  "negative integer",
			input: ":-100\r\n",
			want:  &resp.Integer{Val: -100},
		},
		{
			name:  "zero",
			input: ":0\r\n",
			want:  &resp.Integer{Val: 0},
		},
		{
			name:  "large integer",
			input: ":9223372036854775807\r\n",
			want:  &resp.Integer{Val: 9223372036854775807},
		},
		{
			name:    "invalid integer format",
			input:   ":abc\r\n",
			wantErr: true,
		},
		{
			name:    "missing CRLF",
			input:   ":42\n",
			wantErr: true,
		},
		{
			name:    "incomplete message",
			input:   ":42",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the ':' prefix
			got, err := handleInteger(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleInteger() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleInteger() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNull(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "valid null",
			input: "_\r\n",
			want:  &resp.Null{},
		},
		{
			name:    "invalid CRLF",
			input:   "_\n\n",
			wantErr: true,
		},
		{
			name:    "missing LF",
			input:   "_\r",
			wantErr: true,
		},
		{
			name:    "extra characters",
			input:   "_abc\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '_' prefix
			got, err := handleNull(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleNull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleNull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBoolean(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "true value",
			input: "#t\r\n",
			want:  &resp.Boolean{Val: true},
		},
		{
			name:  "false value",
			input: "#f\r\n",
			want:  &resp.Boolean{Val: false},
		},
		{
			name:    "invalid boolean character",
			input:   "#x\r\n",
			wantErr: true,
		},
		{
			name:    "missing CRLF",
			input:   "#t\n",
			wantErr: true,
		},
		{
			name:    "extra characters",
			input:   "#true\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '#' prefix
			got, err := handleBoolean(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleBoolean() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleBoolean() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseArray(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "empty array",
			input: "*0\r\n",
			want:  &resp.Array{Val: []resp.Message{}},
		},
		{
			name:  "null array",
			input: "*-1\r\n",
			want:  &resp.Array{Val: nil},
		},
		{
			name:  "array with simple strings",
			input: "*2\r\n+hello\r\n+world\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.SimpleString{Val: []byte("hello")},
				&resp.SimpleString{Val: []byte("world")},
			}},
		},
		{
			name:  "array with integers",
			input: "*3\r\n:1\r\n:2\r\n:3\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.Integer{Val: 1},
				&resp.Integer{Val: 2},
				&resp.Integer{Val: 3},
			}},
		},
		{
			name:  "array with mixed types",
			input: "*5\r\n:1\r\n+hello\r\n$5\r\nworld\r\n_\r\n#t\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.Integer{Val: 1},
				&resp.SimpleString{Val: []byte("hello")},
				&resp.BulkString{Size: 5, Str: []byte("world")},
				&resp.Null{},
				&resp.Boolean{Val: true},
			}},
		},
		{
			name:  "nested arrays",
			input: "*2\r\n*2\r\n:1\r\n:2\r\n*2\r\n:3\r\n:4\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.Array{Val: []resp.Message{
					&resp.Integer{Val: 1},
					&resp.Integer{Val: 2},
				}},
				&resp.Array{Val: []resp.Message{
					&resp.Integer{Val: 3},
					&resp.Integer{Val: 4},
				}},
			}},
		},
		{
			name:    "invalid count format",
			input:   "*abc\r\n",
			wantErr: true,
		},
		{
			name:    "negative count (not -1)",
			input:   "*-2\r\n",
			wantErr: true,
		},
		{
			name:    "missing CRLF",
			input:   "*2\n",
			wantErr: true,
		},
		{
			name:    "incomplete array elements",
			input:   "*2\r\n:1\r\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			r.ReadByte() // skip the '*' prefix
			got, err := handleArray(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleArray() error = %v, wantErr %v",
					err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleArray() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    resp.Message
		wantErr bool
	}{
		{
			name:  "parse simple string",
			input: "+OK\r\n",
			want:  &resp.SimpleString{Val: []byte("OK")},
		},
		{
			name:  "parse bulk string",
			input: "$5\r\nhello\r\n",
			want:  &resp.BulkString{Size: 5, Str: []byte("hello")},
		},
		{
			name:  "parse simple error",
			input: "-Error\r\n",
			want:  &resp.SimpleError{Val: []byte("Error")},
		},
		{
			name:  "parse integer",
			input: ":42\r\n",
			want:  &resp.Integer{Val: 42},
		},
		{
			name:  "parse null",
			input: "_\r\n",
			want:  &resp.Null{},
		},
		{
			name:  "parse boolean",
			input: "#t\r\n",
			want:  &resp.Boolean{Val: true},
		},
		{
			name:  "parse array",
			input: "*2\r\n+hello\r\n:42\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.SimpleString{Val: []byte("hello")},
				&resp.Integer{Val: 42},
			}},
		},
		{
			name:  "parse PING command",
			input: "*1\r\n$4\r\nPING\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.BulkString{Size: 4, Str: []byte("PING")},
			}},
		},
		{
			name:  "parse ECHO command",
			input: "*2\r\n$4\r\nECHO\r\n$5\r\nhello\r\n",
			want: &resp.Array{Val: []resp.Message{
				&resp.BulkString{Size: 4, Str: []byte("ECHO")},
				&resp.BulkString{Size: 5, Str: []byte("hello")},
			}},
		},
		{
			name:    "invalid type marker",
			input:   "@invalid\r\n",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(bytes.NewBufferString(tt.input))
			got, err := Parse(r)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse() = %v, want %v", got, tt.want)
			}
		})
	}
}
