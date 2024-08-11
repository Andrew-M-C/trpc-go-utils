package log

var internal = struct {
	keySequence []string
}{
	keySequence: []string{
		"TIME",
		"LEVEL",
		"TEXT",
		"FILE",
		"LINE",
		"FUNC",
		"TRACE_ID",
		"TRACE_ID_STACK",
	},
}
