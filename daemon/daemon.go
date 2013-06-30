package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"
)

var (
	daemon  bool
	quit    bool
	pidfile string
)

func _not_work_Fork(closeno bool) (pid int, err error) {
	// don't run this
	// not work with go threads
	// see: http://code.google.com/p/go/issues/detail?id=227

	darwin := runtime.GOOS == "darwin"

	// already a daemon
	if syscall.Getppid() == 1 {
		return 0, nil
	}

	// fork off the parent process
	ret, ret2, errno := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if errno != 0 {
		return -1, fmt.Errorf("Fork failure: %s", errno)
	}

	// failure
	if ret2 < 0 {
		return -1, fmt.Errorf("Fork failure")
	}

	// handle exception for darwin
	if darwin && ret2 == 1 {
		ret = 0
	}

	// if we got a good PID, then we call exit the parent process.
	if ret > 0 {
		return 0, nil
	}

	// create a new SID for the child process
	s_ret, s_errno := syscall.Setsid()
	if s_errno != nil {
		return -1, fmt.Errorf("Error: syscall.Setsid: %s", s_errno)
	}
	if s_ret < 0 {
		return -1, fmt.Errorf("Error: syscall.Setsid: %s", s_errno)
	}

	if closeno {
		f, e := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if e == nil {
			fd := int(f.Fd())
			syscall.Dup2(fd, int(os.Stdin.Fd()))
			syscall.Dup2(fd, int(os.Stdout.Fd()))
			syscall.Dup2(fd, int(os.Stderr.Fd()))
		}
	}

	return os.Getpid(), nil
}

func MakeDaemon() (pid int, err error) {
	wd, err := os.Getwd()
	if err != nil {
		wd, _ = filepath.Split(os.Args[0])
	}
	env := os.Environ()
	files := []*os.File{os.Stdin, os.Stdout, os.Stderr}
	p, err := os.StartProcess(os.Args[0], os.Args, &os.ProcAttr{
		Dir:   wd,
		Env:   env,
		Files: files,
	})
	if err != nil {
		return 0, err
	}
	return p.Pid, nil
}

func pidFileCreate() {
	fh, err := os.OpenFile(pidfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err == nil {
		defer fh.Close()
		pid := int64(os.Getpid())
		fh.WriteString(strconv.FormatInt(pid, 10))
		log.Println("pid file write", pid)
	}
}

func getPidFromFile() int {
	fh, err := os.Open(pidfile)
	buf := make([]byte, 5)
	if os.IsNotExist(err) {
		return 0
	} else {
		defer fh.Close()
		n, _ := fh.Read(buf)
		buf = buf[:n]
		pid, _ := strconv.Atoi(string(buf))
		return pid
	}
}

func oldQuit() {
	pid := getPidFromFile()
	if pid > 0 {
		syscall.Kill(pid, syscall.SIGQUIT)
		// return
		timeout := 0
		for {
			err := syscall.Kill(pid, 0)
			if err != nil {
				break
			}
			if timeout > 10 {
				syscall.Kill(pid, syscall.SIGKILL)
				os.Remove(pidfile)
				fmt.Printf("failed kill pid: %d force SIGKILL\n", pid)
				break
			}
			<-time.Tick(1 * time.Second)
			fmt.Print(".")
			timeout++
		}
	}
}

func waitSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGQUIT)
	log.Println("Wait SIGQUIT Signal")
	<-c
	log.Println("daemon will quit after 3 seconds")
	<-time.After(1 * time.Second)
	log.Println("daemon quit")
	os.Remove(pidfile)
	os.Exit(0)
}

func init() {
	flag.BoolVar(&daemon, "d", true, "run as daemon if true")
	flag.StringVar(&pidfile, "p", "daemon.pid", "pidfile")
	flag.BoolVar(&quit, "q", false, "quit daemon")
	flag.Parse()
}

func main() {
	os.Chdir(filepath.Dir(os.Args[0]))

	oldQuit()

	if quit {
		fmt.Println("quit")
		return
	}

	if os.Getppid() != 1 && daemon {
		if pid, err := MakeDaemon(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		} else {
			fmt.Println("daemonize success pid ", pid)
			os.Exit(0)
		}
	}

	logfh, _ := os.OpenFile("daemon.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	log.SetOutput(logfh)
	defer logfh.Close()

	time.Sleep(time.Millisecond)

	pidFileCreate()
	waitSignal()
}
