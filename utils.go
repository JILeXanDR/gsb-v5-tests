package main

import (
	"bytes"
	"io"
	"os"
)

func writeFile(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o777)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		return err
	}

	return nil
}
