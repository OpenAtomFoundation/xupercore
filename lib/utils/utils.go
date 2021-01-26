package utils

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

// FileExists reports whether the named file or directory exists.
func FileIsExist(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// Generate unique id, Not strictly unique
// But the probability of repetition is very low
// Run unit test verification
func GenPseudoUniqId() uint64 {
	nano := time.Now().UnixNano()
	rand.Seed(nano)

	randNum1 := rand.Int63()
	randNum2 := rand.Int63()
	shift1 := rand.Intn(16) + 2
	shift2 := rand.Intn(8) + 1

	uId := ((randNum1 >> uint(shift1)) + (randNum2 >> uint(shift2)) + (nano >> 1)) &
		0x1FFFFFFFFFFFFF
	return uint64(uId)
}

// Generate log id, Not strictly unique
// But the probability of repetition is very low
// Run unit test verification
func GenLogId() string {
	return fmt.Sprintf("%d_%d", time.Now().Unix(), GenPseudoUniqId())
}

func GenNonce() string {
	return fmt.Sprintf("%d%8d", time.Now().Unix(), GenPseudoUniqId())
}

// Get call method by runtime.Caller
func GetFuncCall(callDepth int) (string, string) {
	pc, file, line, ok := runtime.Caller(callDepth)
	if !ok {
		return "???:0", "???"
	}

	f := runtime.FuncForPC(pc)
	_, function := path.Split(f.Name())
	_, filename := path.Split(file)

	fline := filename + ":" + strconv.Itoa(line)
	return fline, function
}

// 获取当前源文件目录
func GetCurFileDir() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

// 获取当前执行目录
func GetCurExecDir() string {
	curDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return curDir
}

func GetHostName() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "127.0.0.1"
	}

	return hostname
}

// Print byte slice data as hex string
func F(b []byte) string {
	return fmt.Sprintf("%x", b)
}

// decode string id to bytes
func DecodeId(str string) []byte {
	raw, err := hex.DecodeString(str)
	if err != nil {
		return nil
	}

	return raw
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
