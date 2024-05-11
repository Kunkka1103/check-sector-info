package timeToHeight

import (
	"github.com/filecoin-project/go-state-types/abi"
	"time"
)

func HeightToTime(height abi.ChainEpoch) time.Time {
	subHeight := height - 1851120
	loc, _ := time.LoadLocation("Local")
	CriterionTime, _ := time.ParseInLocation("2006-01-02 15:04:05", "2022-05-30 00:00:00", loc)
	h, _ := time.ParseDuration("0.5m")
	return CriterionTime.Add(time.Duration(subHeight) * h)
}

func HeightToDay(height abi.ChainEpoch) string {
	subHeight := height - 1851120
	loc, _ := time.LoadLocation("Local")
	CriterionTime, _ := time.ParseInLocation("2006-01-02 15:04:05", "2022-05-30 00:00:00", loc)
	h, _ := time.ParseDuration("0.5m")
	return CriterionTime.Add(time.Duration(subHeight) * h).Format("2006-01-02")
}

func StrToTime(dateStr string) (parsedTime time.Time, err error) {
	dateTimeFormat := "2006-01-02 15:04:05"
	parsedTime, err = time.Parse(dateTimeFormat, dateStr)
	return parsedTime, err
}

func TimeToHeight(t time.Time) abi.ChainEpoch {
	loc, _ := time.LoadLocation("Local")
	CriterionTime, _ := time.ParseInLocation("2006-01-02 15:04:05", "2022-05-30 00:00:00", loc)
	subSecond := t.Sub(CriterionTime)
	return abi.ChainEpoch(subSecond.Seconds()/30 + float64(1851120))
}
