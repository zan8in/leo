package fileutil

import (
	"bufio"
	"os"
	"sync"
)

type Syncfile struct {
	mutex     *sync.Mutex
	iohandler *os.File
}

func NewSyncfile(filename string) (*Syncfile, error) {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &Syncfile{mutex: &sync.Mutex{}, iohandler: f}, nil
}

func (sf *Syncfile) Write(content string) {
	sf.mutex.Lock()
	defer sf.mutex.Unlock()

	wbuf := bufio.NewWriterSize(sf.iohandler, len(content))
	wbuf.WriteString(content)
	wbuf.Flush()
}

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
