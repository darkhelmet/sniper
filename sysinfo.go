package main

import (
    "io/ioutil"
    "exec"
    "os"
    "strconv"
    "strings"
)

type Poller interface {
    GetMemory(pid int) float64
}

type ProcPoller struct {}
type PsPoller struct {}

func runWithOutput(binary string, args []string) string {
    cwd, _ := os.Getwd()
    cmd, _ := exec.Run(binary, args, nil, cwd, exec.PassThrough, exec.Pipe, exec.PassThrough)
    cmd.Wait(os.WSTOPPED)
    stdout, _ := ioutil.ReadAll(cmd.Stdout)
    return strings.Trim(string(stdout), " \n")
}

func ps(keyword string, pid int) string {
    binary := "/bin/ps"
    args := []string{binary, "-o", keyword + "=", "-p", strconv.Itoa(pid)}
    return runWithOutput(binary, args)
}

func psInt(keyword string, pid int) float64 {
    kb, _ := strconv.Atof64(ps(keyword, pid))
    return kb / 1024
}

func (p ProcPoller) GetMemory(pid int) float64 {
    // filename := "/proc/" + strconv.Itoa(pid) + "/stat"
    return psInt("rss", pid)
}

func (p PsPoller) GetMemory(pid int) float64 {
    return psInt("rss", pid)
}

func GetPoller() Poller {
    // _, err := os.Stat("/proc")
    // if err == nil {
    //     return new(ProcPoller)
    // }
    return new(PsPoller)
}