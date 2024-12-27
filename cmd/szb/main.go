package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/fudanchii/szb/internal/display"
	"github.com/fudanchii/szb/internal/humanreadable"
	"github.com/fudanchii/szb/internal/kickstart"
	"github.com/fudanchii/szb/internal/sysstats"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"go.bug.st/serial"
)

const (
	CMD_PROMPT      = "$>:"
	DISPLAY_RATE_MS = 100
	STATS_RATE_MS   = 1000
	ONE_MINUTE      = 60
)

type configStruct struct {
	baudRate               int
	connectTo              string
	overflowStyle          string
	dayOfWeekDisplayPeriod int
	timezone               string
}

var (
	config = configStruct{}
)

func init() {
	flag.IntVar(&config.baudRate, "b", 115200, "Baudrate for the serial line.")
	flag.IntVar(&config.dayOfWeekDisplayPeriod, "d", 20, "How long day of week should be displayed in alternate with full date.")
	flag.StringVar(&config.connectTo, "c", "/dev/ttyACM0", "Device name to connect to.")
	flag.StringVar(&config.overflowStyle, "o", "wrap", "Overflow style when text line is longer than 20 characters.")
	flag.StringVar(&config.timezone, "t", "UTC", "Timezone local to use when displaying date time.")
}

type AppHandler struct {
	tty     serial.Port
	buffer  *display.Buffer
	scanner *bufio.Scanner

	datetime     *sysstats.DateTime
	prevCpu      *cpu.Stats
	currCpu      *cpu.Stats
	statsCounter int
}

func main() {
	err := kickstart.Init(setupFn).
		Loop(runFn).
		Then(shutdownFn).
		Exec()

	if err != nil {
		panic(err)
	}
}

func setupFn(kctx *kickstart.Context[AppHandler]) error {
	flag.Parse()

	tty, err := serial.Open(config.connectTo, &serial.Mode{BaudRate: config.baudRate})
	if err != nil {
		return err
	}

	var (
		buffer *display.Buffer
	)

	switch config.overflowStyle {
	case "wrap":
		buffer = display.NewBuffer(display.NewOverflowWrapSpanLines())
	default:
		lines, err := display.TryParseCustomStyle(config.overflowStyle)
		if err != nil {
			return err
		}

		buffer = display.NewBuffer(
			display.NewOverflowCustomStylePerLine(
				lines[0],
				lines[1],
				lines[2],
				lines[3],
			),
		)
	}

	scanner := bufio.NewScanner(tty)

	scanner.Split(bufio.ScanWords)

	timeDate, err := sysstats.NewDateTime(
		config.timezone,
		DISPLAY_RATE_MS,
		config.dayOfWeekDisplayPeriod,
	)
	if err != nil {
		panic(err)
	}

	buffer.SetLine2("")

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
				if strings.HasPrefix(addr.String(), "fe80") {
					continue
				}
				addrsList = append(addrsList, addr.String())
			}
			ifaceList = append(ifaceList, fmt.Sprintf("%s~%s", iface.Name, strings.Join(addrsList, ", ")))
		}
		buffer.SetLine4(strings.Join(ifaceList, " | "))
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

	kctx.AppHandler = AppHandler{
		tty:     tty,
		buffer:  buffer,
		scanner: scanner,

		datetime:     timeDate,
		prevCpu:      prevCpu,
		currCpu:      currCpu,
		statsCounter: statsCounter,
	}

	return nil
}

func shutdownFn(kctx *kickstart.Context[AppHandler]) error {
	defer kctx.AppHandler.tty.Close()

	fmt.Println("Shutting down...")
	kctx.AppHandler.tty.Write([]byte("clr\n"))

	return nil
}

func runFn(kctx *kickstart.Context[AppHandler]) error {
	kctx.AppHandler.buffer.SetLine1(kctx.AppHandler.datetime.String())

	cpuTotal := float64(kctx.AppHandler.currCpu.Total - kctx.AppHandler.prevCpu.Total)

	usrCpu := float64(0)
	sysCpu := float64(0)
	idlCpu := float64(0)

	if cpuTotal != 0 {
		usrCpu = float64(kctx.AppHandler.currCpu.User-kctx.AppHandler.prevCpu.User) / cpuTotal * 100
		sysCpu = float64(kctx.AppHandler.currCpu.System-kctx.AppHandler.prevCpu.System) / cpuTotal * 100
		idlCpu = float64(kctx.AppHandler.currCpu.Idle-kctx.AppHandler.prevCpu.Idle) / cpuTotal * 100
	}

	memStat, err := memory.Get()
	if err != nil {
		return err
	}

	uptime, err := uptime.Get()
	if err != nil {
		return err
	}

	kctx.AppHandler.buffer.SetLine3(
		fmt.Sprintf("mem.total:%s, mem.avail:%s, mem.cached:%s, mem.act:%s, mem.inact:%s, mem.free:%s, cpu.usr:%.1f%%, cpu.sys:%.1f%%, cpu.idle:%.1f%%, up:%v",
			humanreadable.BiBytes(memStat.Total),
			humanreadable.BiBytes(memStat.Available),
			humanreadable.BiBytes(memStat.Cached),
			humanreadable.BiBytes(memStat.Active),
			humanreadable.BiBytes(memStat.Inactive),
			humanreadable.BiBytes(memStat.Free),
			usrCpu, sysCpu, idlCpu,
			humanreadable.Second(uptime)),
	)

	if kctx.AppHandler.scanner.Scan() && kctx.AppHandler.scanner.Text() == CMD_PROMPT {
		cmd := fmt.Sprintf("display:%s\n", kctx.AppHandler.buffer.NextRender())

		kctx.AppHandler.tty.Write([]byte(cmd))

		time.Sleep(DISPLAY_RATE_MS * time.Millisecond)
	}

	kctx.AppHandler.statsCounter++

	if kctx.AppHandler.statsCounter >= (STATS_RATE_MS / DISPLAY_RATE_MS) {
		kctx.AppHandler.prevCpu = kctx.AppHandler.currCpu
		kctx.AppHandler.currCpu, err = cpu.Get()
		if err != nil {
			return err
		}
		kctx.AppHandler.statsCounter = 0
	}

	return nil
}
