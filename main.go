package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/katabase-ai/katabridge/cmd"
)

func main() {
	err := cmd.NewRootCmd().Execute()
	if err == nil {
		return
	}

	var coded interface {
		Error() string
		Code() int
	}
	if errors.As(err, &coded) {
		if msg := coded.Error(); msg != "" && coded.Code() != 1 {
			fmt.Fprintln(os.Stderr, msg)
		}
		os.Exit(coded.Code())
	}

	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
