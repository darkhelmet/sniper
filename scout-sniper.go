package main

import (
    "bufio"
    "exec"
    "flag"
    "http"
    "os"
    "syscall"
    "time"
    // "runtime"
)

const Version = "0.1"
const Seconds = 1e9
// const UserAgent = "scout-sniper/" + Version + " golang/" + runtime.GOOS + "-" + runtime.GOARCH

var extraInterval = flag.Int64("extra-interval", 30, "The time interval between extra checks")
var killIfExtraCheckFail = flag.Bool("kill-if-extra-check-fails", true, "Kill the process if one of the extra checks fails?")
var killCode = flag.Int("kill-signal", 9, "Kill signal to use when killing a process")

var httpTimeoutUrl = flag.String("http-timeout-url", "", "The url to check for HTTP timeouts")
var httpTimeoutTime = flag.Int64("http-timeout-time", 5, "The timeout for the HTTP timeout check")

var httpStatusUrl = flag.String("http-status-url", "", "The url to check for HTTP status")
var httpStatusCode = flag.Int("http-status-code", 200, "The status code for the HTTP status check")

type QC chan bool

func closeAll(chans []QC) {
    for _, c := range(chans) {
        c <- true
        close(c)
    }
}

func check(interval int64, quit QC, f func()) {
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

func getWithTimeout(url string, timeout int64) (ok bool) {
    ctr := make(chan int, 1)
    cto := make(chan int, 1)

    go func() {
       resp, _, err := http.Get(url)
       if err == nil && resp.Body != nil {
           reader := bufio.NewReader(resp.Body)
           reader.ReadString('\r')
           resp.Body.Close()
           ctr <- 1
       }
    }()

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


func setupHttpTimeoutCheck(pid int) QC {
    ch := make(QC)
    go check(*extraInterval, ch, func() {
        if !getWithTimeout(*httpTimeoutUrl, *httpTimeoutTime) {
            println("HTTP timeout check failed after", *httpTimeoutTime, "seconds. Killing process", pid, "with signal", *killCode)
            syscall.Kill(pid, *killCode)
        }
    })
    return ch
}

func setupHttpStatusCheck(pid int) QC {
    return make(QC)
}

func main() {
    flag.Parse()
    cwd, _ := os.Getwd()
    binary, _ := exec.LookPath(flag.Args()[0])
    for {
        cmd, _ := exec.Run(binary, flag.Args(), nil, cwd, exec.PassThrough, exec.PassThrough, exec.PassThrough)
        pid := cmd.Process.Pid
        extras := make([]QC, 0)

        if *httpTimeoutUrl != "" {
            extras = append(extras, setupHttpTimeoutCheck(pid))
        }

        if *httpStatusUrl != "" {
            extras = append(extras, setupHttpStatusCheck(pid))
        }

        cmd.Wait(os.WSTOPPED)
        println("Process died, restarting.")
        closeAll(extras)
    }
}
