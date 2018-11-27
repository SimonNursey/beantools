package main

import (
	. "bean"
	"bean/rpc"
	"fmt"
	"os"
	"time"
)

type orderAgg struct {
	Orders []float64
	Time   time.Time
}

type orderAggTS struct {
	OrderAgg  []orderAgg
	Low, High float64
}

func createGridFromOB(obts OrderBookTS) orderAggTS {
	AggTS := orderAggTS{nil, 9999999.0, 0.0}
	AggTS.OrderAgg = make([]orderAgg, len(obts))

	AggTS.High, AggTS.Low = 0.0, 9999999.0
	for _, obt := range obts {
		for _, lvl := range obt.OB.Bids {
			if lvl.Price < AggTS.Low {
				AggTS.Low = lvl.Price
			}
			if lvl.Price > AggTS.High {
				AggTS.High = lvl.Price
			}
		}
		for _, lvl := range obt.OB.Asks {
			if lvl.Price < AggTS.Low {
				AggTS.Low = lvl.Price
			}
			if lvl.Price > AggTS.High {
				AggTS.High = lvl.Price
			}
		}
	}

	for i, obt := range obts {
		AggTS.OrderAgg[i].Time = obt.Time
		AggTS.OrderAgg[i].Orders = make([]float64, AggTS.priceToIndex(AggTS.High)+1)
		for _, lvl := range obt.OB.Bids {
			AggTS.OrderAgg[i].Orders[AggTS.priceToIndex(lvl.Price)] += lvl.Amount
		}
		for _, lvl := range obt.OB.Asks {
			AggTS.OrderAgg[i].Orders[AggTS.priceToIndex(lvl.Price)] -= lvl.Amount
		}
	}

	return AggTS
}

func (self *orderAggTS) writeGrid(FileName string) {
	f, _ := os.OpenFile(FileName, os.O_CREATE|os.O_WRONLY, 0644)
	f.WriteString(",")
	for i := self.priceToIndex(self.Low); i <= self.priceToIndex(self.High); i++ {
		f.WriteString(fmt.Sprintf("%6.1f,", self.indexToPrice(i)))
	}
	f.WriteString("\n")
	for _, oagg := range self.OrderAgg {
		f.WriteString(oagg.Time.Format("15:04:05") + ",")
		for _, amount := range oagg.Orders {
			if amount == 0 {
				f.WriteString(",")
			} else {
				f.WriteString(fmt.Sprintf("%6.1f,", amount))
			}
		}
		f.WriteString("\n")
	}
	f.Close()
}

func (self orderAggTS) priceToIndex(l float64) int {
	return int(l) - int(self.Low)
}

func (self orderAggTS) indexToPrice(i int) float64 {
	return float64(i + int(self.Low))
}

func main() {

	mds := bean.NewRPCMDSConnC("tcp", bean.MDS_HOST_SG40+":"+bean.MDS_PORT)
	pair := Pair{BTC, USDT}

	start := time.Date(2018, 11, 16, 16, 00, 00, 00, time.Local)
	end := start.Add(time.Duration(3) * time.Hour)
	fmt.Println("Orderbook history from", start.Format("15:04:05"), "to", end.Format("15:04:05"))

	// open book history
	obts, _ := mds.GetOrderBookTS(pair, start, end, 20)
	//	txns, _ := mds.GetTransactions(pair, start, end)

	orderAggT := createGridFromOB(obts)
	orderAggT.writeGrid("orderAgg.csv")
}
