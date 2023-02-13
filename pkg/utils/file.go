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
