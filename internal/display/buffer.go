package display

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

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

func tryParseIntParam(value string) (int, error) {
	splits := strings.Split(value, ":")
	if len(splits) == 2 {
		return strconv.Atoi(splits[1])
	}

	return 1, nil
}

func TryParseCustomStyle(flag string) ([4]NoWrapOverflowStyle, error) {
	var line [4]NoWrapOverflowStyle

	flags := strings.Split(flag, ",")
	if len(flags) != 4 {
		return line, ErrInvalidOverflowStyle
	}

	for idx := range flags {
		switch {
		case flags[idx] == "t":
			line[idx] = &OfTrimLine{}
		case strings.HasPrefix(flags[idx], "em"):
			renderRate, err := tryParseIntParam(flags[idx])
			if err != nil {
				return line, err
			}

			line[idx] = &OfEndlessMarquee{rate: renderRate}
		case strings.HasPrefix(flags[idx], "cm"):
			renderRate, err := tryParseIntParam(flags[idx])
			if err != nil {
				return line, err
			}

			line[idx] = &OfCycleMarquee{rate: renderRate}
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

	line     []byte
	nextLine []byte
	pos      int

	counter, rate int
	rendered      bool
}

func (oem *OfEndlessMarquee) NextRender(currentBuffer []byte) {
	if oem.rendered && oem.counter < oem.rate {
		oem.counter++
		return
	}

	oem.counter = 0
	trailer := []byte{}
	endPos := oem.pos + 20
	if endPos >= len(oem.line) {
		endPos = len(oem.line)
		trailer = oem.nextLine[:20-(endPos-oem.pos)]
	}

	copy(currentBuffer[:], slices.Concat([]byte(oem.line[oem.pos:endPos]), []byte(trailer)))

	oem.pos += 1
	if oem.pos == len(oem.line) {
		oem.line = oem.nextLine
		oem.pos = 0
	}

	oem.rendered = true
}

func (oem *OfEndlessMarquee) setCurrentLine(line string) {
	if len(line) >= 20 {
		line += " . "
	}

	oem.nextLine = ReplaceRuneWithLCDCharMap(fmt.Sprintf("%-20s", line))
	if len(oem.line) == 0 {
		oem.line = oem.nextLine
		oem.pos = 0
	}
}

type OfCycleMarquee struct {
	BaseNoWrapOverflowStyle

	slideLeft         bool
	line              []byte
	nextLine          []byte
	pos               int
	changed, rendered bool

	counter, rate int
}

func (ocm *OfCycleMarquee) NextRender(currentBuffer []byte) {
	if ocm.rendered && ocm.counter < ocm.rate {
		ocm.counter++
		return
	}

	ocm.counter = 0

	if len(ocm.nextLine) == 20 {
		if ocm.changed {
			copy(currentBuffer[:], ocm.nextLine[:20])

			ocm.line = ocm.nextLine
			ocm.changed = false
		}

		return
	}

	endPos := ocm.pos + 20
	if endPos >= len(ocm.line) {
		endPos = len(ocm.line)
	}

	copy(currentBuffer[:], []byte(ocm.line[ocm.pos:endPos]))

	if ocm.slideLeft {
		ocm.slideLeft = endPos < len(ocm.line)
		if ocm.slideLeft {
			ocm.pos += 1
		}
	} else {
		ocm.pos -= 1
		ocm.slideLeft = ocm.pos == 0
		if ocm.slideLeft {
			ocm.line = ocm.nextLine
		}
	}

	ocm.rendered = true
}

func (ocm *OfCycleMarquee) setCurrentLine(line string) {
	ocm.nextLine = ReplaceRuneWithLCDCharMap(fmt.Sprintf("%-20s", line))
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

	line    []byte
	changed bool
}

func (otl *OfTrimLine) NextRender(currentBuffer []byte) {
	if otl.changed {
		copy(currentBuffer[:], otl.line[0:20])
		otl.changed = false
	}
}

func (otl *OfTrimLine) setCurrentLine(line string) {
	otl.line = ReplaceRuneWithLCDCharMap(fmt.Sprintf("%-20s", line))
	otl.changed = true
}

type OfCustomStylePerLine struct {
	BaseOverflowStyle

	line1 NoWrapOverflowStyle
	line2 NoWrapOverflowStyle
	line3 NoWrapOverflowStyle
	line4 NoWrapOverflowStyle
}

func NewOverflowCustomStylePerLine(l1, l2, l3, l4 NoWrapOverflowStyle) *OfCustomStylePerLine {
	return &OfCustomStylePerLine{
		line1: l1,
		line2: l2,
		line3: l3,
		line4: l4,
	}
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

	line     []byte
	lchanged bool
}

func NewOverflowWrapSpanLines() *OfWrapSpanLines {
	return &OfWrapSpanLines{}
}

func (owl *OfWrapSpanLines) NextRender(currentBuffer *CharLcdBuffer) []byte {
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
	owl.line = ReplaceRuneWithLCDCharMap(fmt.Sprintf("%-[1]*s", CharLcdDimension, line))
	owl.lchanged = true
}

const CharLcdDimension = 4 * 20

type CharLcdBuffer [CharLcdDimension]byte

type Buffer struct {
	internal        CharLcdBuffer
	overflowContext OverflowStyle
}

func NewBuffer(style OverflowStyle) *Buffer {
	return &Buffer{overflowContext: style}
}

func (db *Buffer) NextRender() []byte {
	return db.overflowContext.NextRender(&db.internal)
}

func (db *Buffer) SetLine1(line string) {
	db.overflowContext.setLine1(line)
}

func (db *Buffer) SetLine2(line string) error {
	return db.overflowContext.setLine2(line)
}

func (db *Buffer) SetLine3(line string) error {
	return db.overflowContext.setLine3(line)
}

func (db *Buffer) SetLine4(line string) error {
	return db.overflowContext.setLine4(line)
}
