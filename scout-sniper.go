package main

import (
    "exec"
    "flag"
    "os"
    "time"
)

const Seconds = 1e9

var interval = flag.Int64("interval", 2, "The interval to do the basic check")

type QC chan bool

func pump(msg bool, chans []QC) {
    for _, c := range(chans) {
        c <- msg
        close(c)
    }
}

func check(interval int64, quit chan bool, f func()) {
    for {
        time.Sleep(interval)
        select {
        case <- quit:
            return
        default:
            f()
        }
    }
}

func main() {
    flag.Parse()
    cwd, _ := os.Getwd()
    binary, _ := exec.LookPath(flag.Args()[0])
    for {
        cmd, _ := exec.Run(binary, flag.Args(), nil, cwd, exec.PassThrough, exec.PassThrough, exec.PassThrough)
        cmd.Wait(os.WSTOPPED)
    }
}