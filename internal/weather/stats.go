package weather

import (
	"errors"
	"fmt"
	"os"
	"time"

	owm "github.com/briandowns/openweathermap"
)

var apiKey = os.Getenv("OWM_API_KEY")

type Stats struct {
	current       *owm.CurrentWeatherData
	nowDisplaying string
}

func NewStats(coordinate *owm.Coordinates) (*Stats, error) {
	if apiKey == "" {
		return nil, errors.New("OWM_API_KEY is empty.")
	}

	weatherer, err := owm.NewCurrent("C", "en", apiKey)
	if err != nil {
		return nil, err
	}

	if err := weatherer.CurrentByCoordinates(coordinate); err != nil {
		return nil, err
	}

	stats := &Stats{current: weatherer, nowDisplaying: "desc"}

	go func(stats *Stats) {
		fiveMinutes := 5 * 60
		counter := 0
		for {
			time.Sleep(10 * time.Second)

			if counter%30 == 0 {
				stats.nowDisplaying = "desc"
			}

			if counter%50 == 0 {
				stats.nowDisplaying = "temp"
			}

			if counter == fiveMinutes {
				err := weatherer.CurrentByCoordinates(coordinate)
				if err != nil {
					fmt.Println(err)
				}

				counter = 0

				continue
			}

			counter += 10
		}
	}(stats)

	return stats, nil
}

func (s *Stats) String() string {
	switch s.nowDisplaying {
	case "desc":
		desc := s.current.Weather[0].Description
		descLen := len(desc)
		return fmt.Sprintf("%*s", 20, fmt.Sprintf("%*s", -(((20-descLen)/2)+descLen), desc))
	case "temp":
		temp := fmt.Sprintf("%.1fºC", s.current.Main.Temp)
		tempLen := len(temp)
		return fmt.Sprintf("%*s", 20, fmt.Sprintf("%*s", -(((20-tempLen)/2)+tempLen), temp))
	}

	return "(fetching...)"
}
