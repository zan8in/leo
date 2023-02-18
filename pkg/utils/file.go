package utils

import (
	"bufio"
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
