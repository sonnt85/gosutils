package shellwords

import (
	"github.com/sonnt85/goutils/shellwords"
	"go/build"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
)

type testCase struct {
	expected []string
	message  string
}

var testcases = []struct { //for parser
	line     string
	expected []string
}{
	{``, []string{}},
	{`""`, []string{``}},
	{`''`, []string{``}},
	{`var --bar=baz`, []string{`var`, `--bar=baz`}},
	{`var --bar="baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar=baz"`, []string{`var`, `--bar=baz`}},
	{`var "--bar='baz'"`, []string{`var`, `--bar='baz'`}},
	{"var --bar=`baz`", []string{`var`, "--bar=`baz`"}},
	{`var "--bar=\"baz'"`, []string{`var`, `--bar="baz'`}},
	{`var "--bar=\'baz\'"`, []string{`var`, `--bar='baz'`}},
	{`var --bar='\'`, []string{`var`, `--bar=\`}},
	{`var "--bar baz"`, []string{`var`, `--bar baz`}},
	{`var --"bar baz"`, []string{`var`, `--bar baz`}},
	{`var  --"bar baz"`, []string{`var`, `--bar baz`}},
	{`a "b"`, []string{`a`, `b`}},
	{`a " b "`, []string{`a`, ` b `}},
	{`a "   "`, []string{`a`, `   `}},
	{`a 'b'`, []string{`a`, `b`}},
	{`a ' b '`, []string{`a`, ` b `}},
	{`a '   '`, []string{`a`, `   `}},
	{"foo bar\\  ", []string{`foo`, `bar `}},
	{`foo "" bar ''`, []string{`foo`, ``, `bar`, ``}},
}

func TestSplitString(t *testing.T) {
	testCases := map[string]testCase{
		`a "b b" a`:      {[]string{"a", "b b", "a"}, "quoted strings"},
		`a "'b' c" d`:    {[]string{"a", "'b' c", "d"}, "escaped double quotes"},
		`a '"b" c' d`:    {[]string{"a", `"b" c`, "d"}, "escaped single quotes"},
		`a b\ c d`:       {[]string{"a", "b c", "d"}, "escaped spaces"},
		`a   b\ c d`:     {[]string{"a", "b c", "d"}, "extra spaces in seplator"},
		`   a b\ c d`:    {[]string{"a", "b c", "d"}, "extra leading spaces"},
		`a b\ c d   `:    {[]string{"a", "b c", "d"}, "extra tailing spaces"},
		"a 'aa\nbb\ncc'": {[]string{"a", "aa\nbb\ncc"}, "multi-line"},
	}
	errorCases := []string{
		`a "b c d e`,
		`a 'b c d e`,
		`"a "'b' c" d`,
	}

	for input, res := range testCases {
		actual, err := Split(input)
		assert.NoError(t, err)
		assert.Equal(t, res.expected, actual, res.message)
	}
	for _, input := range errorCases {
		_, err := Split(input)
		assert.Error(t, err)
	}
}

func TestEscape(t *testing.T) {
	testCases := []string{
		``,
		`abc`,
		`a b c`,
		`a  b `,
		`a\nb`,
		"a\nb",
		"a\n\nb",
		`a $HOME`,
		`sh -c 'pwd'`,
		`a"b'`,
	}

	for _, expected := range testCases {
		escaped := Escape(expected)
		actual, err := exec.Command("sh", "-c", "printf %s "+escaped).Output()
		assert.NoError(t, err)
		assert.Equal(t, expected, string(actual), "input: [%s], escaped: [%s], actual: [%s]", expected, escaped, actual)
	}
}

func TestJoin(t *testing.T) {
	testCases := map[string][]string{
		"":                {},
		"a b c":           {"a", "b", "c"},
		`a\ b c`:          {"a b", "c"},
		`sh -c echo\ foo`: {"sh", "-c", "echo foo"},
	}
	for expected, input := range testCases {
		actual := Join(input)
		assert.Equal(t, expected, actual, "input: %#v, expected: (%s), actual: (%s)", input, expected, actual)
	}
}

//parser

func TestSimple(t *testing.T) {
	for _, testcase := range testcases {
		args, err := Parse(testcase.line)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(args, testcase.expected) {
			t.Fatalf("Expected %#v for %q, but %#v:", testcase.expected, testcase.line, args)
		}
	}
}

func TestError(t *testing.T) {
	_, err := Parse("foo '")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = Parse(`foo "`)
	if err == nil {
		t.Fatal("Should be an error")
	}

	_, err = Parse("foo `")
	if err == nil {
		t.Fatal("Should be an error")
	}
}

func TestShellRun(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}

func TestShellRunNoEnv(t *testing.T) {
	old := os.Getenv("SHELL")
	defer os.Setenv("SHELL", old)
	os.Unsetenv("SHELL")

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	pwd, err := shellRun("pwd", "")
	if err != nil {
		t.Fatal(err)
	}

	pwd2, err := shellRun("pwd", path.Join(dir, "/_example"))
	if err != nil {
		t.Fatal(err)
	}

	if pwd == pwd2 {
		t.Fatal("`pwd` should be changed")
	}
}

func TestBacktick(t *testing.T) {
	goversion, err := shellRun("go version", "")
	if err != nil {
		t.Fatal(err)
	}

	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse("echo `go version`")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", goversion}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`echo $(echo foo)`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "foo"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse(`echo bar=$(echo 200)cm`)
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "bar=200cm"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	parser.ParseBacktick = false
	args, err = parser.Parse(`echo $(echo "foo")`)
	expected = []string{"echo", `$(echo "foo")`}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
	args, err = parser.Parse("echo $(`echo1)")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "$(`echo1)"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickMulti(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse(`echo $(go env GOPATH && go env GOROOT)`)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", build.Default.GOPATH + "\n" + build.Default.GOROOT}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestBacktickError(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	_, err := parser.Parse("echo `go Version`")
	if err == nil {
		t.Fatal("Should be an error")
	}
	expected := "exit status 2:go Version: unknown command\nRun 'go help' for usage.\n"
	if expected != err.Error() {
		t.Fatalf("Expected %q, but %q", expected, err.Error())
	}
	_, err = parser.Parse(`echo $(echo1)`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo FOO=$(echo1)`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo $(echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo $ (echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo (echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse(`echo )echo1`)
	if err == nil {
		t.Fatal("Should be an error")
	}
}

func TestEnv(t *testing.T) {
	os.Setenv("FOO", "bar")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "bar"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestCustomEnv(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	parser.Getenv = func(k string) string { return map[string]string{"FOO": "baz"}[k] }
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "baz"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestNoEnv(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $BAR")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvArguments(t *testing.T) {
	os.Setenv("FOO", "bar baz")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $FOO")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "bar", "baz"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestEnvArgumentsFail(t *testing.T) {
	os.Setenv("FOO", "bar '")

	parser := NewParser()
	parser.ParseEnv = true
	_, err := parser.Parse("echo $FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse("$FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	_, err = parser.Parse("echo $FOO")
	if err == nil {
		t.Fatal("Should be an error")
	}
	os.Setenv("FOO", "bar `")
	_, err = parser.Parse("$FOO ")
	if err == nil {
		t.Fatal("Should be an error")
	}
}

func TestDupEnv(t *testing.T) {
	os.Setenv("FOO", "bar")
	os.Setenv("FOO_BAR", "baz")

	parser := NewParser()
	parser.ParseEnv = true
	args, err := parser.Parse("echo $$FOO$")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"echo", "$bar$"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	args, err = parser.Parse("echo $${FOO_BAR}$")
	if err != nil {
		t.Fatal(err)
	}
	expected = []string{"echo", "$baz$"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}

func TestHaveMore(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true

	line := "echo foo; seq 1 10"
	args, err := parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := []string{"echo", "foo"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position == 0 {
		t.Fatalf("Commands should be remaining")
	}

	line = string([]rune(line)[parser.Position+1:])
	args, err = parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected = []string{"seq", "1", "10"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position > 0 {
		t.Fatalf("Commands should not be remaining")
	}
}

func TestHaveRedirect(t *testing.T) {
	parser := NewParser()
	parser.ParseEnv = true

	line := "ls -la 2>foo"
	args, err := parser.Parse(line)
	if err != nil {
		t.Fatalf(err.Error())
	}
	expected := []string{"ls", "-la"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}

	if parser.Position == 0 {
		t.Fatalf("Commands should be remaining")
	}
}

func TestBackquoteInFlag(t *testing.T) {
	parser := NewParser()
	parser.ParseBacktick = true
	args, err := parser.Parse("cmd -flag=`echo val1` -flag=val2")
	if err != nil {
		panic(err)
	}
	expected := []string{"cmd", "-flag=val1", "-flag=val2"}
	if !reflect.DeepEqual(args, expected) {
		t.Fatalf("Expected %#v, but %#v:", expected, args)
	}
}
