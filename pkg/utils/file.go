package utils

import (
	"bufio"
	"errors"
	"os"
)

func ReadFileLineByLine(filename string) ([]string, error) {
	var result []string

	fp, err := os.Open(filename)
	if err != nil {
		return result, err
	}

	buf := bufio.NewScanner(fp)
	for {
		if !buf.Scan() {
			break //文件读完了,退出for
		}
		line := buf.Text() //获取每一行
		result = append(result, line)
	}
	return result, err
}

func WriteString(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	wbuf := bufio.NewWriterSize(file, len(content))
	wbuf.WriteString(content)
	wbuf.Flush()

	return nil
}

// FileExists checks if the file exists in the provided path
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ReadFile with filename
func ReadFile(filename string) (chan string, error) {
	if !FileExists(filename) {
		return nil, errors.New("file doesn't exist")
	}
	out := make(chan string)
	go func() {
		defer close(out)
		f, err := os.Open(filename)
		if err != nil {
			return
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			out <- scanner.Text()
		}
	}()

	return out, nil
}
