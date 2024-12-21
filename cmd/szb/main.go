package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mackerelio/go-osstat/cpu"
	"github.com/mackerelio/go-osstat/memory"
	"github.com/mackerelio/go-osstat/uptime"
	"go.bug.st/serial"
)

const CMD_PROMPT = "$>:"

var (
	ErrSettingThisLine      = errors.New("buffer: error, setting this line is not supported")
	ErrInvalidOverflowStyle = errors.New("config: error parsing overflow style, please specify style to use for the entire 4 lines (e.g. t,em,em,em)")
)

type OverflowStyle interface {
	ImplOverflowStyle()

	NextRender(*CharLcdBuffer) []byte
	setLine1(string)
	setLine2(string) error
	setLine3(string) error
	setLine4(string) error
}

func TryParseCustomStyle(flag string) ([4]NoWrapOverflowStyle, error) {
	var (
		line [4]NoWrapOverflowStyle
	)

	flags := strings.Split(flag, ",")
	if len(flags) != 4 {
		return line, ErrInvalidOverflowStyle
	}

	for idx := range flags {
		switch flags[idx] {
		case "t":
			line[idx] = &OfTrimLine{}
		case "em":
			line[idx] = &OfEndlessMarquee{}
		case "cm":
			line[idx] = &OfCycleMarquee{}
		default:
			return line, fmt.Errorf("style parse: error invalid style for line%d", idx)
		}
	}

	return line, nil
}

type NoWrapOverflowStyle interface {
	ImplNoWrapOverflowStyle()

	NextRender([]byte)
	setCurrentLine(string)
}

type BaseOverflowStyle struct{}

func (BaseOverflowStyle) ImplOverflowStyle() {}

func (BaseOverflowStyle) setLine2(line string) error {
	return ErrSettingThisLine
}

func (BaseOverflowStyle) setLine3(line string) error {
	return ErrSettingThisLine
}

func (BaseOverflowStyle) setLine4(line string) error {
	return ErrSettingThisLine
}

type BaseNoWrapOverflowStyle struct{}

func (BaseNoWrapOverflowStyle) ImplNoWrapOverflowStyle() {}

type OfEndlessMarquee struct {
	BaseNoWrapOverflowStyle

	line     string
	nextLine string
	pos      int
}

func (oem *OfEndlessMarquee) NextRender(currentBuffer []byte) {
	trailer := ""
	endPos := oem.pos + 20
	if endPos >= len(oem.line) {
		endPos = len(oem.line)
		trailer = oem.nextLine[:20-(endPos-oem.pos)]
	}

	copy(currentBuffer[:], []byte(oem.line[oem.pos:endPos]+trailer))

	oem.pos += 1
	if oem.pos == len(oem.line) {
		oem.line = oem.nextLine
		oem.pos = 0
	}
}

func (oem *OfEndlessMarquee) setCurrentLine(line string) {
	if len(line) >= 20 {
		line += " . "
	}

	oem.nextLine = fmt.Sprintf("%-20s", line)
	if len(oem.line) == 0 {
		oem.line = oem.nextLine
		oem.pos = 0
	}
}

type OfCycleMarquee struct {
	BaseNoWrapOverflowStyle

	slideLeft bool
	line      string
	nextLine  string
	pos       int
	changed   bool
}

func (ocm *OfCycleMarquee) NextRender(currentBuffer []byte) {
	if ocm.changed && len(ocm.nextLine) == 20 {
		copy(currentBuffer[:], ocm.nextLine[:20])
		ocm.line = ocm.nextLine
		ocm.changed = false
		return
	}

	endPos := ocm.pos + 20
	if endPos >= len(ocm.line) {
		endPos = len(ocm.line)
	}

	copy(currentBuffer[:], []byte(ocm.line[ocm.pos:endPos]))

	if ocm.slideLeft {
		ocm.pos += 1
		ocm.slideLeft = endPos < len(ocm.line)
	} else {
		ocm.pos -= 1
		ocm.slideLeft = ocm.pos == 0
		if ocm.slideLeft {
			ocm.line = ocm.nextLine
		}
	}
}

func (ocm *OfCycleMarquee) setCurrentLine(line string) {
	ocm.nextLine = fmt.Sprintf("%-20s", line)
	ocm.changed = true

	if len(ocm.line) == 0 {
		ocm.line = ocm.nextLine
		ocm.pos = 0
		ocm.slideLeft = true
		ocm.changed = true
	}
}

type OfTrimLine struct {
	BaseNoWrapOverflowStyle

	line    string
	changed bool
}

func (otl *OfTrimLine) NextRender(currentBuffer []byte) {
	if otl.changed {
		copy(currentBuffer[:], otl.line[0:20])
		otl.changed = false
	}
}

func (otl *OfTrimLine) setCurrentLine(line string) {
	otl.line = fmt.Sprintf("%-20s", line)
	otl.changed = true
}

type OfCustomStylePerLine struct {
	BaseOverflowStyle

	line1 NoWrapOverflowStyle
	line2 NoWrapOverflowStyle
	line3 NoWrapOverflowStyle
	line4 NoWrapOverflowStyle
}

func (ocsp *OfCustomStylePerLine) NextRender(currentBuffer *CharLcdBuffer) []byte {
	ocsp.line1.NextRender(currentBuffer[:20])
	ocsp.line2.NextRender(currentBuffer[40:60])
	ocsp.line3.NextRender(currentBuffer[20:40])
	ocsp.line4.NextRender(currentBuffer[60:80])

	return currentBuffer[:]
}

func (ocsp *OfCustomStylePerLine) setLine1(line string) {
	ocsp.line1.setCurrentLine(line)
}

func (ocsp *OfCustomStylePerLine) setLine2(line string) error {
	ocsp.line2.setCurrentLine(line)
	return nil
}

func (ocsp *OfCustomStylePerLine) setLine3(line string) error {
	ocsp.line3.setCurrentLine(line)
	return nil
}

func (ocsp *OfCustomStylePerLine) setLine4(line string) error {
	ocsp.line4.setCurrentLine(line)
	return nil
}

type OfWrapSpanLines struct {
	BaseOverflowStyle

	line     string
	lchanged bool
}

func (owl *OfWrapSpanLines) NextRender(currentBuffer *CharLcdBuffer) []byte {
	if len(owl.line) < CharLcdDimension {
		owl.setLine1(owl.line)
	}

	if owl.lchanged {
		bytes := []byte(owl.line)[:CharLcdDimension]

		// line 1
		copy(currentBuffer[0:], bytes[0:20])
		// line 3
		copy(currentBuffer[20:], bytes[40:60])
		// line 2
		copy(currentBuffer[40:], bytes[20:40])
		// line 4
		copy(currentBuffer[60:], bytes[60:80])
	}

	return currentBuffer[:]
}

func (owl *OfWrapSpanLines) setLine1(line string) {
	owl.line = fmt.Sprintf("%-[1]*s", CharLcdDimension, line)
	owl.lchanged = true
}

const CharLcdDimension = 4 * 20

type CharLcdBuffer [CharLcdDimension]byte

type DisplayBuffer struct {
	internal        CharLcdBuffer
	overflowContext OverflowStyle
}

func NewDisplayBuffer(style OverflowStyle) *DisplayBuffer {
	return &DisplayBuffer{overflowContext: style}
}

func (db *DisplayBuffer) NextRender() []byte {
	return db.overflowContext.NextRender(&db.internal)
}

func (db *DisplayBuffer) SetLine1(line string) {
	db.overflowContext.setLine1(line)
}

func (db *DisplayBuffer) SetLine2(line string) error {
	return db.overflowContext.setLine2(line)
}

func (db *DisplayBuffer) SetLine3(line string) error {
	return db.overflowContext.setLine3(line)
}

func (db *DisplayBuffer) SetLine4(line string) error {
	return db.overflowContext.setLine4(line)
}

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

func run(tty serial.Port, dbuff *DisplayBuffer) {
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

	cpuStatsCounter := 0

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
			time.Sleep(500 * time.Millisecond)

			cmd := fmt.Sprintf("display:%s\n", dbuff.NextRender())

			tty.Write([]byte(cmd))
		}

		cpuStatsCounter++

		if cpuStatsCounter >= 2 {
			prevCpu = currCpu
			currCpu, err = cpu.Get()
			if err != nil {
				panic(err)
			}
			cpuStatsCounter = 0
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
		run(tty, NewDisplayBuffer(&OfWrapSpanLines{}))
	default:
		lines, err := TryParseCustomStyle(config.overflowStyle)
		if err != nil {
			log.Fatal(err)
		}

		run(tty, NewDisplayBuffer(&OfCustomStylePerLine{
			line1: lines[0],
			line2: lines[1],
			line3: lines[2],
			line4: lines[3],
		}))
	}
}
