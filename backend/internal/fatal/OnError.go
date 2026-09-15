package fatal

import (
	"fmt"
)

func OnError(err error, args ...any) {
	if err == nil {
		return
	}
	message := fmt.Sprint(append(args, err)...)
	LogError(message)
}
