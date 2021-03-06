package zvelo

import (
	"io"
	"os"

	"github.com/fatih/color"

	msg "zvelo.io/msg/msgpb"
	"zvelo.io/msg/status"
)

func Errorf(format string, a ...interface{}) {
	_, _ = color.New(color.FgRed).Fprintf(os.Stderr, format, a...)
}

func IsComplete(result *msg.QueryResult) bool {
	if result == nil || result.QueryStatus == nil {
		return false
	}

	if status.ErrorProto(result.QueryStatus.Error) != nil {
		// there was an error, complete is implied
		return true
	}

	return result.QueryStatus.Complete
}

func PrintfFunc(attr color.Attribute, w io.Writer) func(format string, a ...interface{}) {
	c := color.New(attr).FprintfFunc()
	return func(format string, a ...interface{}) {
		c(w, format, a...)
	}
}
