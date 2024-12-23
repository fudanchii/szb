package sysstats

import (
	"fmt"
	"time"
)

const (
	ONE_SECOND_IN_MS  = 1000
	ONE_MINUTE_PERIOD = 60
)

type DateTime struct {
	callCounter   int
	renderRate    int
	showDoWPeriod int
	timezone      *time.Location
}

func NewDateTime(timezone string, renderRate, showDoWPeriod int) (*DateTime, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, err
	}

	return &DateTime{
		renderRate:    renderRate,
		showDoWPeriod: showDoWPeriod,
		timezone:      loc,
	}, nil
}

func (dt *DateTime) String() string {
	var result string

	now := time.Now().In(dt.timezone)

	waitUnit := ONE_SECOND_IN_MS / dt.renderRate
	startDoWAt := (ONE_MINUTE_PERIOD - dt.showDoWPeriod) * waitUnit
	if dt.callCounter >= startDoWAt {
		if dt.callCounter > ONE_MINUTE_PERIOD*waitUnit {
			dt.callCounter = 0
		}

		result = fmt.Sprintf("%-12s%s", now.Format("Monday"), now.Format("15:04:05"))
	} else {
		result = now.Format("2006-01-02  15:04:05")
	}

	dt.callCounter++

	return result
}
