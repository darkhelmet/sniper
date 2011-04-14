package main

import (
    "bufio"
    "exec"
    "flag"
    "http"
    "os"
    "syscall"
    "time"
    "strconv"
    // "runtime"
)

const Version = "0.1"
const Seconds = 1e9
const KiloBytes = 1024
const Megabytes = KiloBytes * 1024
// const UserAgent = "scout-sniper/" + Version + " golang/" + runtime.GOOS + "-" + runtime.GOARCH

var extraInterval = flag.Int64("extra-interval", 30, "The time interval between extra checks")
var killCode = flag.Int("kill-signal", 9, "Kill signal to use when killing a process")

var httpTimeoutUrl = flag.String("http-timeout-url", "", "The url to check for HTTP timeouts")
var httpTimeoutTime = flag.Int64("http-timeout-time", 5, "The timeout for the HTTP timeout check")

var httpStatusUrl = flag.String("http-status-url", "", "The url to check for HTTP status")
var httpStatusCode = flag.Int("http-status-code", 200, "The status code for the HTTP status check")

var maxMemory = flag.Float64("max-mem", 0, "The amount of memory in megabytes that the process is allowed")

type BC chan bool
type ABC []BC
type ProcInfo map[string] string

func getProcessInformation(pid int) ProcInfo {
    return map[string] string {
        "foo": "bar",
    }
}

func (a ABC) closeAll() {
    for _, c := range(a) {
        c <- true
        close(c)
    }
}

func timeout(timeout int64, f func(chan int)) (ok bool) {
    ctr := make(chan int, 1)
    cto := make(chan int, 1)

    go f(ctr)

    go func() {
        time.Sleep(timeout * Seconds)
        cto <- 1
    }()

    select {
    case <- cto:
        ok = false
    case <- ctr:
        ok = true
    }

    return ok
}


func check(interval int64, quit BC, f func()) {
    for {
        time.Sleep(interval * Seconds)
        select {
        case <- quit:
            return
        default:
            f()
        }
    }
}

func setupHttpTimeoutCheck(pid int) BC {
    ch := make(BC)
    go check(*extraInterval, ch, func() {
        failed := !timeout(*httpTimeoutTime, func(ctr chan int) {
            resp, _, err := http.Get(*httpTimeoutUrl)
            if err == nil && resp.Body != nil {
                defer resp.Body.Close()
                reader := bufio.NewReader(resp.Body)
                reader.ReadString('\r')
                ctr <- 1
            }
        })
        if failed {
            println("HTTP timeout check failed after", *httpTimeoutTime, "seconds. Killing process", pid, "with signal", *killCode)
            syscall.Kill(pid, *killCode)
        }
    })
    return ch
}

func setupHttpStatusCheck(pid int) BC {
    ch := make(BC)
    go check(*extraInterval, ch, func() {
        failed := !timeout(*httpTimeoutTime, func(ctr chan int) {
            resp, _, err := http.Get(*httpStatusUrl)
            if err == nil && *httpStatusCode == resp.StatusCode {
                ctr <- 1
            }
        })
        if failed {
            println("HTTP status check failed after", *httpTimeoutTime, "seconds. Killing process", pid, "with signal", *killCode)
            syscall.Kill(pid, *killCode)
        }
    })
    return ch
}

func setupMaxMemoryCheck(pid int) BC {
    ch := make(BC)
    poller := GetPoller()
    alreadyExceded := false
    go check(*extraInterval, ch, func() {
        kb := poller.GetMemory(pid)
        if kb > *maxMemory {
            if alreadyExceded {
                println("Memory check failed for 2 intervals. Killing process", pid, "with signal", *killCode)
                syscall.Kill(pid, *killCode)
            }
            alreadyExceded = true
        } else {
            alreadyExceded = false
        }
    })
    return ch
}

func main() {
    flag.Parse()
    cwd, _ := os.Getwd()
    binary, _ := exec.LookPath(flag.Args()[0])
    for {
        cmd, _ := exec.Run(binary, flag.Args(), nil, cwd, exec.PassThrough, exec.PassThrough, exec.PassThrough)
        pid := cmd.Process.Pid
        extras := make(ABC, 0)

        if *httpTimeoutUrl != "" {
            extras = append(extras, setupHttpTimeoutCheck(pid))
        }

        if *httpStatusUrl != "" {
            extras = append(extras, setupHttpStatusCheck(pid))
        }

        if *maxMemory > 0 {
            extras = append(extras, setupMaxMemoryCheck(pid))
        }

        cmd.Wait(os.WSTOPPED)
        println("Process died, restarting.")
        extras.closeAll()
    }
}
