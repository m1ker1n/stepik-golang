package main

import (
	"fmt"
	"io"
	"os"
)

func fileString(file os.DirEntry) (string, error) {
	info, err := file.Info()
	if err != nil {
		return "", err
	}
	name := info.Name()
	size := info.Size()
	if size == 0 {
		return fmt.Sprintf("%s (empty)", name), nil
	}
	return fmt.Sprintf("%s (%db)", name, size), nil
}

func filter[T any](slice []T, predicate func(el T) bool) []T {
	result := make([]T, 0, cap(slice))
	for _, el := range slice {
		if predicate(el) {
			result = append(result, el)
		}
	}
	return result
}

func dirTreeRecursive(out io.Writer, path string, printFiles bool, prefix string) error {
	files, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	//need to filter because we need handle last element differently
	//if we not filter may come problem that last element is file & prinfiles is disabled
	//so this file actually is not last as we don't need to print it
	if !printFiles {
		files = filter(files, func(el os.DirEntry) bool {
			return el.IsDir()
		})
	}
	if len(files) == 0 {
		return nil
	}

	for _, file := range files[:len(files)-1] {
		if err := printFile(out, path, printFiles, prefix, file, false); err != nil {
			return err
		}
	}

	//last element
	file := files[len(files)-1]
	if err := printFile(out, path, printFiles, prefix, file, true); err != nil {
		return err
	}
	return nil
}

func printFile(out io.Writer, path string, printFiles bool, prefix string, file os.DirEntry, isLast bool) error {
	branchSymbol := "├"
	addToPrefix := "│\t"
	if isLast {
		branchSymbol = "└"
		addToPrefix = "\t"
	}
	if file.IsDir() {
		if _, err := fmt.Fprintf(out, "%s%s\n", prefix+branchSymbol+"───", file.Name()); err != nil {
			return err
		}
		if err := dirTreeRecursive(out, path+"/"+file.Name(), printFiles, prefix+addToPrefix); err != nil {
			return err
		}
	} else {
		str, err := fileString(file)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "%s%s\n", prefix+branchSymbol+"───", str); err != nil {
			return err
		}
	}
	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return dirTreeRecursive(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
