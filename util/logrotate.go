package util

import (
	"os"
	"sync"
	"time"
)

//RotateWriter for log rotate
type RotateWriter struct {
	lock     sync.Mutex
	filename string
	fp       *os.File
	maxsize  int64
}

//NewRotateWriter make a new RotateWriter. Return nil if error occus during setup
func NewRotateWriter(filename string, maxsize int64) *RotateWriter {
	w := &RotateWriter{filename: filename, maxsize: maxsize}
	err := w.Rotate()
	if err != nil {
		return nil
	}
	return w
}

//Write satisfies the io.Writer interface
func (w *RotateWriter) Write(output []byte) (int, error) {
	s, err := os.Stat(w.filename)
	if err == nil {
		if s.Size() > w.maxsize {
			w.Rotate()
		}
	}
	w.lock.Lock()
	defer w.lock.Unlock()
	return w.fp.Write(output)
}

//Rotate perform the actual act of rotating and reopening file
func (w *RotateWriter) Rotate() (err error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// close existing file if open
	if w.fp != nil {
		err = w.fp.Close()
		w.fp = nil
		if err != nil {
			return
		}
	}

	// rename dest file if it already exists
	_, err = os.Stat(w.filename)
	if err == nil {
		err = os.Rename(w.filename, w.filename+"."+time.Now().Format(time.RFC3339))
		if err != nil {
			return
		}
	}

	//create a file
	w.fp, err = os.Create(w.filename)
	return
}
