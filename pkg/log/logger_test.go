package log

import (
	_ "embed"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"k8s.io/klog/v2/textlogger"
)

//go:embed testdata/logs.txt
var logsRaw string

var testLogs []testLog

type testLog struct {
	msg string
	kvs []any
}

func init() {
	lines := strings.Split(logsRaw, "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var logData map[string]any
		if err := json.Unmarshal([]byte(line), &logData); err != nil {
			panic(err)
		}

		testLog := testLog{}
		for k, v := range logData {
			if k == "msg" {
				testLog.msg = v.(string)
			} else {
				testLog.kvs = append(testLog.kvs, k)
				testLog.kvs = append(testLog.kvs, v)
			}
		}

		testLogs = append(testLogs, testLog)
	}
}

func Test(t *testing.T) {
	l := New("default").WithName("#123").WithValues("a", 1)
	l.W("something happened", "error", "bla")
}

func Benchmark(b *testing.B) {

	f, err := os.Open("/dev/null")
	if err != nil {
		b.Fatal(err)
	}

	l := New("default").WithName("#123").WithValues("a", 1).WithFile(f)

	b.Run("log", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tl := range testLogs {
				l.I(tl.msg, tl.kvs...)
			}
		}
	})

	kl := textlogger.NewLogger(textlogger.NewConfig(textlogger.Output(f), textlogger.Verbosity(5)))

	b.Run("klog", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, tl := range testLogs {
				kl.Info(tl.msg, tl.kvs...)
			}
		}
	})
}
