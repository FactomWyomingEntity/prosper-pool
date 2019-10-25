package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	r, err := regexp.Compile("prosper-miner_.*")
	if err != nil {
		panic(err)
	}
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if r.Match([]byte(path)) {
			d := filepath.Dir(path)
			b := filepath.Base(path)

			arr := strings.SplitN(b, "_", 2)
			if len(arr) != 2 {
				fmt.Printf("%s failed to be renamed\n", path)
				return nil
			}
			newPath := fmt.Sprintf("%s-%s_%s", arr[0], runtime.Version(), arr[1])
			err := os.Rename(path, newPath)
			if err != nil {
				fmt.Printf("%s: %s", path, err.Error())
			} else {
				fmt.Printf("%s -> %s\n", path, filepath.Join(d, newPath))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("Done.")
	}
}
