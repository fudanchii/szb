package main

import (
	"bufio"
	"flag"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fudanchii/szb/internal/display"
	"github.com/fudanchii/szb/internal/kickstart"
	"github.com/fudanchii/szb/internal/sysstats"
	"github.com/fudanchii/szb/internal/weather"
	"go.bug.st/serial"

	owm "github.com/briandowns/openweathermap"
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
	coordLongitude         float64
	coordLatitude          float64
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

	coordInput := ""
	flag.StringVar(&coordInput, "x", "35.66017559963725,139.70039568656168", "Lat,Long coordinate for weather information, by default it's pointing to Shibuya.")
	coords := strings.SplitN(coordInput, ",", 2)
	if len(coords) != 2 {
		panic("please input correct coordinate")
	}

	var err error

	config.coordLatitude, err = strconv.ParseFloat(coords[0], 64)
	if err != nil {
		panic("invalid latitude value")
	}

	config.coordLongitude, err = strconv.ParseFloat(coords[1], 64)
	if err != nil {
		panic("invalid longitude value")
	}

}

type AppHandler struct {
	tty     serial.Port
	buffer  *display.Buffer
	scanner *bufio.Scanner

	datetime   *sysstats.DateTime
	netStats   *sysstats.NetworkStats
	aggregates *sysstats.Aggregates
	weatherer  *weather.Stats
}

func main() {
	err := kickstart.
		Init(setup).
		Loop(mainOperation).
		Then(shutdown).
		Exec()

	if err != nil {
		panic(err)
	}
}

func setup(kctx *kickstart.Context[AppHandler]) error {
	var (
		buffer *display.Buffer
	)

	flag.Parse()

	tty, err := serial.Open(config.connectTo, &serial.Mode{BaudRate: config.baudRate})
	if err != nil {
		return err
	}

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

	dateTime, err := sysstats.NewDateTime(
		config.timezone,
		DISPLAY_RATE_MS,
		config.dayOfWeekDisplayPeriod,
	)
	if err != nil {
		return err
	}

	aggregates, err := sysstats.NewAggregates()
	if err != nil {
		return err
	}

	netStats := sysstats.NewNetworkStats()

	weatherer, err := weather.NewStats(&owm.Coordinates{
		Latitude:  config.coordLatitude,
		Longitude: config.coordLongitude,
	})

	if err != nil {
		return err
	}

	kctx.AppHandler = AppHandler{
		tty:     tty,
		buffer:  buffer,
		scanner: scanner,

		datetime:   dateTime,
		netStats:   netStats,
		aggregates: aggregates,
		weatherer:  weatherer,
	}

	return nil
}

func shutdown(kctx *kickstart.Context[AppHandler]) error {
	defer kctx.AppHandler.tty.Close()

	fmt.Println("Shutting down...")
	kctx.AppHandler.tty.Write([]byte("clr\n"))

	return nil
}

func mainOperation(kctx *kickstart.Context[AppHandler]) error {
	kctx.AppHandler.buffer.SetLine1(kctx.AppHandler.datetime)
	kctx.AppHandler.buffer.SetLine2(kctx.AppHandler.weatherer)
	kctx.AppHandler.buffer.SetLine3(kctx.AppHandler.aggregates)
	kctx.AppHandler.buffer.SetLine4(kctx.AppHandler.netStats)

	if kctx.AppHandler.scanner.Scan() && kctx.AppHandler.scanner.Text() == CMD_PROMPT {
		kctx.AppHandler.tty.Write(
			slices.Concat([]byte("display:"), kctx.AppHandler.buffer.NextRender(), []byte("\n")),
		)

		time.Sleep(DISPLAY_RATE_MS * time.Millisecond)
	}

	return nil
}
