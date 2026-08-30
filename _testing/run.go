package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	mode := flag.String("mode", "autocomplete", "test mode: autocomplete or typesinfo")
	flag.Parse()
	root, _ := os.Getwd()
	bin := filepath.Join(root, "gocode")
	if runtime.GOOS == "windows" {
		bin = filepath.Join(root, "gocode.exe")
	}
	if _, err := os.Stat(bin); err != nil {
		fail("gocode binary not found: %v", err)
	}
	var ok bool
	if *mode == "typesinfo" {
		ok = runTypesInfo(root, bin)
	} else {
		ok = runAutocomplete(root, bin)
	}
	if !ok {
		os.Exit(1)
	}
}

func runAutocomplete(root, bin string) bool {
	tests, _ := filepath.Glob(filepath.Join(root, "test.*"))
	sort.Strings(tests)
	version := goVersion()
	platform := ""
	if runtime.GOOS == "windows" {
		platform = ".windows"
	}
	addr := fmt.Sprintf("127.0.0.1:%d", 41000+os.Getpid()%1000)
	server := exec.Command(bin, "-s", "-sock", "tcp", "-addr", addr)
	server.Stdout = io.Discard
	server.Stderr = io.Discard
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start server: %v\n", err)
		return false
	}
	defer func() {
		exec.Command(bin, "-sock", "tcp", "-addr", addr, "close").Run()
		server.Process.Kill()
		server.Wait()
	}()
	time.Sleep(300 * time.Millisecond)
	pass := 0
	for _, dir := range tests {
		cursor := cursorOf(dir)
		expected := expectedPath(dir, version, platform)
		file := filepath.Join(dir, "test.go.in")
		got, _ := command(bin, "-sock", "tcp", "-addr", addr, "-in", file, "autocomplete", file, strconv.Itoa(cursor))
		want, _ := os.ReadFile(expected)
		if string(got) != string(want) {
			fmt.Printf("%s: FAIL!\nGot:\n%sExpected:\n%s", filepath.Base(dir), got, want)
		} else {
			fmt.Printf("%s: PASS!\n", filepath.Base(dir))
			pass++
		}
	}
	fmt.Printf("\nSummary (total: %d)\n  PASS: %d\n  FAIL: %d\n", len(tests), pass, len(tests)-pass)
	return pass == len(tests)
}

func runTypesInfo(root, bin string) bool {
	addr := fmt.Sprintf("127.0.0.1:%d", 40000+os.Getpid()%1000)
	server := exec.Command(bin, "-s", "-sock", "tcp", "-addr", addr)
	server.Stdout = io.Discard
	server.Stderr = io.Discard
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start server: %v\n", err)
		return false
	}
	defer func() {
		exec.Command(bin, "-sock", "tcp", "-addr", addr, "close").Run()
		server.Process.Kill()
		server.Wait()
	}()
	time.Sleep(300 * time.Millisecond)
	tests, _ := filepath.Glob(filepath.Join(root, "typesinfo.[0-9]*"))
	sort.Strings(tests)
	pass := 0
	for _, dir := range tests {
		if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			continue
		}
		cursor := cursorOf(dir)
		file := filepath.Join(dir, "test.go.in")
		gotBytes, _ := command(bin, "-sock", "tcp", "-addr", addr, "-in", file, "liteide_typesinfo", file, strconv.Itoa(cursor))
		got := normalize(string(gotBytes), file)
		want, _ := os.ReadFile(filepath.Join(dir, "out.expected"))
		if got != string(want) {
			fmt.Printf("%s: FAIL!\nGot:\n%sExpected:\n%s", filepath.Base(dir), got, want)
		} else {
			fmt.Printf("%s: PASS!\n", filepath.Base(dir))
			pass++
		}
	}
	fmt.Printf("\nSummary (total: %d)\n  PASS: %d\n  FAIL: %d\n", len(tests), pass, len(tests)-pass)
	return pass == len(tests)
}

func command(bin string, args ...string) ([]byte, error) { return exec.Command(bin, args...).Output() }
func cursorOf(dir string) int {
	files, _ := filepath.Glob(filepath.Join(dir, "cursor.*"))
	p := strings.Split(filepath.Base(files[0]), ".")
	n, _ := strconv.Atoi(p[len(p)-1])
	if source, err := os.ReadFile(filepath.Join(dir, "test.go.in")); err == nil && bytesBeforeCursor(source, n) {
		for i := 0; i < n && i+1 < len(source); i++ {
			if source[i] == '\r' && source[i+1] == '\n' {
				n++
			}
		}
	}
	return n
}

func bytesBeforeCursor(source []byte, cursor int) bool {
	return cursor >= 0 && cursor <= len(source) && bytes.Index(source[:cursor], []byte("\r\n")) >= 0
}
func goVersion() string {
	b, _ := exec.Command("go", "env", "GOVERSION").Output()
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "go")
}
func expectedPath(dir, version, platform string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		mm := strings.Join(parts[:2], ".")
		p := filepath.Join(dir, "out.expected.go"+mm+platform)
		if _, err := os.Stat(p); err == nil {
			return p
		}
		p = filepath.Join(dir, "out.expected.go"+mm)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(dir, "out.expected")
}
func normalize(s, file string) string {
	s = strings.ReplaceAll(s, "\\", "/")
	if absolute, err := filepath.Abs(file); err == nil {
		s = strings.ReplaceAll(s, filepath.ToSlash(absolute), "FILE")
	}
	for _, root := range []string{os.Getenv("GOROOT"), goroot()} {
		if root != "" {
			s = strings.ReplaceAll(s, filepath.ToSlash(root), "GOROOT")
		}
	}
	s = regexp.MustCompile(`GOROOT/src/([^:]+):[0-9]+:[0-9]+`).ReplaceAllString(s, "GOROOT/src/$1:LINE:COLUMN")
	return s
}
func goroot() string {
	b, _ := exec.Command("go", "env", "GOROOT").Output()
	return strings.TrimSpace(string(b))
}
func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
