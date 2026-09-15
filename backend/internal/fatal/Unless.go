package fatal

func Unlessf(b bool, format string, args ...any) {
	if !b {
		LogErrorf(format, args...)
	}
}
