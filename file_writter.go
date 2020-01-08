package hlog

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	logFileNameTimeFormat     = "2006010215"
	defaultQueueSize          = 100000
	defaultFlushDiskTimeout   = 50 * time.Millisecond
	defaultFileChangeInterval = time.Second
	defaultFileMillInterval   = 10 * time.Second

	MEGABYTE = 1024 * 1024
)

var (
	singleWriter io.Writer = os.Stdout
	singleQueue            = make(chan []byte, defaultQueueSize)
	currentTime            = time.Now // currentTime exists so it can be mocked out by tests.
)

type FileWriter struct {
	*FileConfig
	mu        sync.Mutex
	wg        WaitGroupWrapper
	iNode     uint64
	file      *os.File
	startMill sync.Once
	millCh    chan bool
	quitChan  chan struct{} //外界用于通知此Writer关闭
	closeChan chan struct{} //自身的关闭，用于本身的Close()方法
}

type logInfo struct {
	ts time.Time
	os.FileInfo
}

func newFileWritter(fc *FileConfig, wg WaitGroupWrapper, quitChan chan struct{}) (fw *FileWriter) {
	if fc == nil {
		fc = &FileConfig{}
	}
	fw = &FileWriter{FileConfig: fc, wg: wg, quitChan: quitChan, closeChan: make(chan struct{})}
	fw.init()
	return fw
}

func (fw *FileWriter) init() {
	if len(fw.FileName) > 0 { //配置了文件输出
		if err := fw.openFile(fw.currentFileName()); err != nil {
			fmt.Printf("OpenFile err: %v, use stdout", err)
		} else {
			fw.wg.Wrap(fw.fileWatcher)
		}
	}
	fw.wg.Wrap(fw.logWatcher)
}

func (fw *FileWriter) fileWatcher() {
	changeTimer := time.NewTicker(defaultFileChangeInterval)
	millTimer := time.NewTicker(defaultFileMillInterval)
	for {
		select {
		case <-fw.closeChan:
			if fw.file != nil {
				fw.file.Close()
			}
			return
		case <-changeTimer.C:
			if fw.file == nil { //代表是stdout的输出
				return
			}
			//文件不存在，或者大小变小，或者时间过了一个周期，都重新打开
			var needReopen bool
			currentFileName := fw.currentFileName()
			if currentFileName != fw.file.Name() {
				fmt.Printf("rotate file, old file name:%s new file name:%s \n", fw.file.Name(), currentFileName)
				needReopen = true
			} else if stat, err := os.Stat(fw.file.Name()); (err != nil && !os.IsExist(err)) ||
				(nil != stat && stat.Sys().(*syscall.Stat_t).Ino != fw.iNode) {
				fmt.Printf("fileScaner diff, err:%v inode:%v \n", err, fw.iNode)
				needReopen = true
			}
			if needReopen {
				if err := fw.openFile(currentFileName); err != nil {
					fmt.Printf("fileWatcher OpenFile err: %v", err)
				}
			}
		case <-millTimer.C:
			fw.mill()
		}
	}
}

//根据当前应使用的filename开启新的文件
func (fw *FileWriter) openFile(name string) error {
	if len(name) == 0 {
		return errors.New("invalid current fileName")
	} else if f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666); err != nil {
		return err
	} else {
		if fw.file != nil {
			fw.file.Close()
		}
		fw.file = f
		//修改单例writer，进入临界区
		fw.mu.Lock()
		singleWriter = f
		fw.mu.Unlock()
		//出临界区
		stat, err := os.Stat(fw.file.Name())
		if err == nil && nil != stat {
			fw.iNode = stat.Sys().(*syscall.Stat_t).Ino
		}
		return nil
	}
}

//此处异步清理多余的日志
func (fw *FileWriter) millRunOnce() error {
	if fw.MaxSize == 0 && fw.MaxAge == 0 {
		return nil
	}

	files, err := fw.oldLogFiles()
	if err != nil {
		return err
	}

	var remove []logInfo
	//根据最多保留天数清理
	if fw.MaxAge > 0 {
		diff := time.Duration(int64(24*time.Hour) * fw.MaxAge)
		cutoff := currentTime().Add(-1 * diff)
		fmt.Println("cutoff ts:", cutoff.Unix())
		var remaining []logInfo
		for _, f := range files {
			fmt.Println("file:", f.Name(), "ts:", f.ts.Unix())
			if f.ts.Before(cutoff) {
				fmt.Println("out of range, remove")
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}
	//根据最多保留日志总大小清理
	if fw.MaxSize > 0 {
		preserved := make(map[string]bool)
		var remaining []logInfo
		var totalSize int64
		for _, f := range files {
			if !preserved[f.Name()] { //去个重
				preserved[f.Name()] = true
				if totalSize+f.Size() < fw.MaxSize*MEGABYTE {
					totalSize += f.Size()
					remaining = append(remaining, f)
				} else {
					remove = append(remove, f)
				}
			}
		}
		files = remaining
	}
	//删除所有的待清除日志
	for _, f := range remove {
		errRemove := os.Remove(filepath.Join(filepath.Dir(fw.FileName), f.Name()))
		if err == nil && errRemove != nil {
			err = errRemove
		}
	}

	return err
}

func (fw *FileWriter) mill() {
	fw.startMill.Do(func() {
		fw.millCh = make(chan bool, 1)
		go fw.millRun()
	})
	select {
	case fw.millCh <- true:
	default:
	}
}

func (fw *FileWriter) millRun() {
	for range fw.millCh {
		err := fw.millRunOnce()
		if err != nil {
			fmt.Printf("mill logs error:%v", err)
		}
	}
}

func (fw *FileWriter) logWatcher() {
	for {
		select {
		case msg := <-singleQueue:
			fw.flush(msg)
		case <-fw.quitChan:
			for {
				select {
				case msg := <-singleQueue:
					fw.flush(msg)
				default:
					fw.Close()
					return
				}
			}
		case <-fw.closeChan:
			for {
				select {
				case msg := <-singleQueue:
					fw.flush(msg)
				default:
					return
				}
			}
		}
	}
}

func (fw *FileWriter) flush(msg []byte) {
	done := make(chan bool, 1)
	fw.wg.Wrap(func() {
		fw.mu.Lock()
		defer fw.mu.Unlock()
		singleWriter.Write(msg)
		done <- true
	})
	select {
	case <-done:
		return
	case <-time.After(defaultFlushDiskTimeout):
		return
	}
}

func (fw *FileWriter) Write(p []byte) (n int, err error) {
	select {
	case singleQueue <- p:
		return len(p), nil
	default:
		return 0, nil
	}
}

func (fw *FileWriter) Close() error {
	close(fw.closeChan)
	return nil
}

func (fw *FileWriter) currentFileName() string {
	if fw.Interval <= 0 { //如果不需要切分，直接返回正常的文件名
		return fw.FileName
	}
	dir := filepath.Dir(fw.FileName)
	filename := filepath.Base(fw.FileName)
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]
	t := currentTime()
	if !fw.LocalTime {
		t = t.UTC()
	}
	t = t.Add(-time.Hour * time.Duration(int64(t.Hour())%fw.Interval)) //根据interval取整
	ts := t.Format(logFileNameTimeFormat)
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, ts, ext))
}

func (fw *FileWriter) timeFromFileName(filename, prefix, ext string) (time.Time, error) {
	if !strings.HasPrefix(filename, prefix) {
		return time.Time{}, errors.New("mismatched prefix")
	}
	if !strings.HasSuffix(filename, ext) {
		return time.Time{}, errors.New("mismatched extension")
	}
	t := filename[len(prefix) : len(filename)-len(ext)]
	return time.Parse(logFileNameTimeFormat, t)
}

func (fw *FileWriter) prefixAndExt() (prefix, ext string) {
	filename := filepath.Base(fw.FileName)
	ext = filepath.Ext(filename)
	prefix = filename[:len(filename)-len(ext)] + "-"
	return prefix, ext
}

func (fw *FileWriter) oldLogFiles() ([]logInfo, error) {
	files, err := ioutil.ReadDir(filepath.Dir(fw.FileName))
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}
	logFiles := make([]logInfo, 0)

	prefix, ext := fw.prefixAndExt()

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if t, err := fw.timeFromFileName(f.Name(), prefix, ext); err == nil {
			logFiles = append(logFiles, logInfo{t, f})
		}

	}

	sort.Sort(byFormatTime(logFiles))

	return logFiles, nil
}

// byFormatTime sorts by newest time formatted in the name.
type byFormatTime []logInfo

func (b byFormatTime) Less(i, j int) bool {
	return b[i].ts.After(b[j].ts)
}

func (b byFormatTime) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byFormatTime) Len() int {
	return len(b)
}
