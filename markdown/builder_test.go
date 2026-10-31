package markdown

import (
	"strings"
	"testing"
)

func TestHeaders(t *testing.T) {
	tests := []struct {
		name string
		got  func(b *Builder) *Builder
		want string
	}{
		{"h1", func(b *Builder) *Builder { return b.H1("T") }, "# T"},
		{"h2", func(b *Builder) *Builder { return b.H2("T") }, "## T"},
		{"h3", func(b *Builder) *Builder { return b.H3("T") }, "### T"},
		{"h4", func(b *Builder) *Builder { return b.H4("T") }, "#### T"},
		{"h5", func(b *Builder) *Builder { return b.H5("T") }, "##### T"},
		{"h6", func(b *Builder) *Builder { return b.H6(" a\nb ") }, "###### a b"},
		{"alt h1", func(b *Builder) *Builder { return b.AltH1(" Title\r\n") }, "Title\n====="},
		{"alt h2", func(b *Builder) *Builder { return b.AltH2("Title") }, "Title\n-----"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.got(new(Builder)).String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBlocksAndInline(t *testing.T) {
	b := new(Builder)
	if b.Block() == b {
		t.Error("Block must return a new builder")
	}
	got := b.P("para").HR().Line("a").BR().Code("x := 1", "go").String()
	for _, want := range []string{"para\n\n", "-----", "a   \n", "```go\nx := 1\n```"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	inline := map[string]string{
		b.InlineCode("c"):        "`c`",
		b.Emphasis("e"):          "*e*",
		b.Italic("i"):            "*i*",
		b.StrongEmphasis("s"):    "**s**",
		b.Bold("b"):              "**b**",
		b.CombinedEmphasis("c"):  "**_c_**",
		b.Strikethrough("x"):     "~~x~~",
		b.Link("http://u", "t"):  "[t](http://u)",
		b.Image("http://u", "t"): "![t](http://u)",
	}
	for got, want := range inline {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestQuoteAndLists(t *testing.T) {
	if got, want := new(Builder).Blockquote("a\nb").String(), ">  a\n>  b"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := new(Builder).BulletedList("a", "b\nc").String(), "* a\n* b\n  c"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := new(Builder).NumberedList("a", "b\nc").String(), "1. a\n2. b\n   c"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTable(t *testing.T) {
	if got := new(Builder).Table([][]string{{"only"}}, nil).String(); got != "" {
		t.Errorf("short table should render nothing, got %q", got)
	}
	table := [][]string{
		{"Name", "Type", "Description"},
		{"id", "integer", ""},
		{"label", "string"},
		{"é", "a\nb", "long description"},
	}
	got := new(Builder).Table(table, []TableAlignment{AlignRight, AlignCenter}).String()
	lines := strings.Split(got, "\n")
	if len(lines) != 5 {
		t.Fatalf("want 5 lines, got %d: %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[1], "| ") || !strings.Contains(lines[1], ":") {
		t.Errorf("unexpected separator %q", lines[1])
	}
	if !strings.Contains(lines[0], "Name") || !strings.Contains(lines[0], "Description") {
		t.Errorf("unexpected header %q", lines[0])
	}
	if !strings.Contains(lines[4], "a b") {
		t.Errorf("newline in cell not flattened: %q", lines[4])
	}
}

func TestPadSpaces(t *testing.T) {
	tests := []struct {
		align TableAlignment
		want  string
	}{
		{AlignLeft, "ab   "},
		{AlignRight, "   ab"},
		{AlignCenter, " ab  "},
	}
	for _, tt := range tests {
		if got := padSpaces("ab", 5, tt.align); got != tt.want {
			t.Errorf("align %d: got %q, want %q", tt.align, got, tt.want)
		}
	}
	if got := padSpaces("toolong", 3, AlignLeft); got != "toolong" {
		t.Errorf("got %q", got)
	}
}

func TestCharLen(t *testing.T) {
	if got := charLen("éa"); got != 2 {
		t.Errorf("combining accent: got %d, want 2", got)
	}
}
