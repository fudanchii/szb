package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fudanchii/szb/internal/display"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"go.bug.st/serial"
)

const CMD_PROMPT = "$>:"

type configStruct struct {
	baudRate      int
	connectTo     string
	overflowStyle string
}

var (
	config = configStruct{}
)

func init() {
	flag.IntVar(&config.baudRate, "b", 115200, "Baudrate for the serial line.")
	flag.StringVar(&config.connectTo, "c", "/dev/ttyACM0", "Device name to connect to.")
	flag.StringVar(&config.overflowStyle, "o", "wrap", "Overflow style when text line is longer than 20 characters.")
}

func run(tty serial.Port, dbuff *display.Buffer) {
	scanner := bufio.NewScanner(tty)

	scanner.Split(bufio.ScanWords)

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}

	dbuff.SetLine2("")

	{
		ifaces, err := net.Interfaces()
		if err != nil {
			panic(err)
		}

		ifaceList := []string{}
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				panic(err)
			}

			if iface.Name == "lo" || len(addrs) == 0 {
				continue
			}

			addrsList := []string{}
			for _, addr := range addrs {
				addrsList = append(addrsList, addr.String())
			}

			ifaceList = append(ifaceList, fmt.Sprintf("%s~%s", iface.Name, strings.Join(addrsList, ",")))
		}

		dbuff.SetLine4(strings.Join(ifaceList, " | "))
	}

	prevCpu, err := cpu.Get()
	if err != nil {
		panic(err)
	}

	currCpu, err := cpu.Get()
	if err != nil {
		panic(err)
	}

	statsCounter := 0

	for {
		now := time.Now().In(loc)

		cpuTotal := float64(currCpu.Total - prevCpu.Total)

		usrCpu := float64(0)
		sysCpu := float64(0)
		idlCpu := float64(0)

		if cpuTotal != 0 {
			usrCpu = float64(currCpu.User-prevCpu.User) / cpuTotal * 100
			sysCpu = float64(currCpu.System-prevCpu.System) / cpuTotal * 100
			idlCpu = float64(currCpu.Idle-prevCpu.Idle) / cpuTotal * 100
		}

		memStat, err := memory.Get()
		if err != nil {
			panic(err)
		}

		memUsed := humanize.IBytes(memStat.Used)
		memFree := humanize.IBytes(memStat.Free)

		uptime, err := uptime.Get()
		if err != nil {
			panic(err)
		}

		dbuff.SetLine1(now.Format("2006/01/02  15:04:05"))
		dbuff.SetLine3(
			fmt.Sprintf("mem used: %s, mem free: %s, cpu.usr: %.1f%%, cpu.sys: %.1f%%, cpu.idl: %.1f%%, up: %v",
				memUsed, memFree, usrCpu, sysCpu, idlCpu, uptime),
		)

		if scanner.Scan() && scanner.Text() == CMD_PROMPT {
			cmd := fmt.Sprintf("display:%s\n", dbuff.NextRender())

			tty.Write([]byte(cmd))

			time.Sleep(500 * time.Millisecond)
		}

		statsCounter++

		if statsCounter >= 2 {
			prevCpu = currCpu
			currCpu, err = cpu.Get()
			if err != nil {
				panic(err)
			}
			statsCounter = 0
		}
	}
}

func main() {
	flag.Parse()

	tty, err := serial.Open(config.connectTo, &serial.Mode{BaudRate: config.baudRate})
	if err != nil {
		log.Fatal(err)
	}

	defer tty.Close()

	switch config.overflowStyle {
	case "wrap":
		run(tty, display.NewBuffer(display.NewOverflowWrapSpanLines()))
	default:
		lines, err := display.TryParseCustomStyle(config.overflowStyle)
		if err != nil {
			log.Fatal(err)
		}

		run(tty, display.NewBuffer(
			display.NewOverflowCustomStylePerLine(
				lines[0],
				lines[1],
				lines[2],
				lines[3],
			),
		))
	}
}
