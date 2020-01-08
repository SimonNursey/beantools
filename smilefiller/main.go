package main

import (
	. "bean"
	"beanex/db/mds"
	"fmt"
	"time"
)

func main() {
	mdsdb := mds.NewMDS("MDS", mds.DB_HOST_LOCAL, "8086")
	// MDS writer channels writing new mds data
	mdsWriter, mdsStop, _ := mdsdb.Writer()

	st := time.Date(2019, 7, 13, 8, 0, 0, 0, time.UTC)
	en := time.Now()

	threadCount := make(chan bool, 10)
	for asof := st; asof.Before(en); asof = asof.Add(30 * time.Minute) {
		threadCount <- true
		go func(asof time.Time) {
			defer func() { <-threadCount }()
			mkt, _ := mdsdb.GetMarket("DERIBIT", asof)
			lastUpdate := mkt.LastUpdateTime(mkt.LastUpdatedContract())
			if asof.Sub(lastUpdate) < 30*time.Minute {
				return
			}
			for _, exp := range mkt.Expiries() {
				smile, _, _, _, _, _ := mkt.FittedFivePointSmile(asof, Pair{BTC, USD}, exp)
				//msg := risk.ShowFittedFivePointSmile(smile, strikes, bidVols, askVols, deltas, markVols)
				//fmt.Println(msg)
				if smile != nil {
					mdsWriter <- mds.SmilePoint{
						TimeStamp: asof,
						Pair:      Pair{BTC, USD},
						Expiry:    exp,
						VolSmile:  smile,
					}
					fmt.Printf("Asof:%s Exp:%s Atm:%3.1f\n", asof.Format(time.ANSIC), exp.Format(time.ANSIC), smile.Atm*100.0)
				}
			}
		}(asof)
	}
	for i := 0; i < cap(threadCount); i++ {
		threadCount <- true
	}
	mdsStop <- true
}
