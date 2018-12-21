package main

import (
	"fmt"
	"math"
	"os"
	"time"

	. "bean"

	bean "bean/rpc"
)

// iceberg detector - look for orders which are executed but keep on popping up
// look for multiple trades at the same level and beyond the original order size

// look every 10 secs
// for each transaction, how many times has the level traded in the last 30secs
// how does that compare to the maximum order size during the period.

const dateLayout = "2/1/2006 15:04:05"

func main() {
	st := time.Date(2018, 11, 25, 0, 0, 0, 0, time.UTC)
	en := time.Date(2018, 12, 18, 0, 0, 0, 0, time.UTC)
	pair := Pair{BTC, USDT}
	timeWindow := time.Duration(5) * time.Minute // the historic window in which to scan for orders and transactions at the same level
	amountThreshold := 10.0                      // this is the minimum total traded amount at a specific level
	stealthFactor := 2.0                         // this is the minimum ratio between the total traded amount and the maximum order amount or maximum trade amount

	f, _ := os.Create("detector.csv")
	fmt.Fprintf(f, "Type,Time,Level,Traded Amount,Max Trade,Trade Count,Given,Paid,Max Order,Bids,Asks,Subsequent Trades,1min,5min,Profit,Cum Pnl\n")

	mds := bean.NewRPCMDSConnC("tcp", bean.MDS_HOST_SG40+":"+bean.MDS_PORT)

	txns, _ := mds.GetTransactions(pair, st, en)

	lastReportedLevel := math.NaN()
	cumPnl := 0.0

	// look for levels that have traded in significant amounts
	for i, t := range txns {
		if lastReportedLevel == txns[i].Price {
			continue
		}

		trnSellAmount := 0.0
		trnBuyAmount := 0.0
		trnMaxAmount := 0.0
		trnCount := 0

		// look for other preceeding transactions at matching price level within the time window
		for j := i; (j > 0) && txns[j].TimeStamp.After(t.TimeStamp.Add(-timeWindow)); j-- {
			if txns[j].Price == t.Price {
				if txns[j].Maker == Buyer {
					trnBuyAmount += txns[j].Amount
				} else {
					trnSellAmount += txns[j].Amount
				}
				trnMaxAmount = math.Max(trnMaxAmount, txns[j].Amount)
				trnCount++
			}
		}

		trnTotalAmount := trnBuyAmount + trnSellAmount

		orderMaxAmount := 0.0
		orderBids := 0
		orderAsks := 0

		// if the total amount traded at that level is above threshold and a multiple of the maximum ticket size
		// then we have a potential iceberg
		if trnTotalAmount > amountThreshold && trnTotalAmount/trnMaxAmount > stealthFactor {
			// now get and scan all historical orders in the window for the largest order at the same level
			obts, _ := mds.GetOrderBookTS(pair, t.TimeStamp.Add(-timeWindow), t.TimeStamp, 20)
			for _, ob := range obts {
				// in each time sample, look through the bids and offers for orders at matching price levels
				for _, o := range ob.OB.Bids {
					if o.Price == t.Price {
						orderMaxAmount = math.Max(orderMaxAmount, o.Amount)
						orderBids++
					}
				}
				for _, o := range ob.OB.Asks {
					if o.Price == t.Price {
						orderMaxAmount = math.Max(orderMaxAmount, o.Amount)
						orderAsks++
					}
				}
			}

			// if the amount traded is a multiple of the largest single order then we are looking at a disguised
			// large order
			if orderMaxAmount > 0 && trnTotalAmount/orderMaxAmount > stealthFactor {
				// ICEBERG AHOY

				// Now report to the captain

				fmt.Fprintf(f, "ICEBERG,%v,%7.2f,%5.2f,%5.2f,%2v,%5.2f,%5.2f",
					t.TimeStamp.Format(dateLayout), t.Price, trnTotalAmount, trnMaxAmount, trnCount, trnBuyAmount, trnSellAmount)
				fmt.Fprintf(f, ",%5.2f,%2v,%2v", orderMaxAmount, orderBids, orderAsks)

				// How much subsequently trades at that level
				subsequentlyTrades := 0.0
				var j int
				for j = i + 1; j < len(txns) && txns[j].TimeStamp.Before(t.TimeStamp.Add(5*timeWindow)); j++ {
					if txns[j].Price == t.Price {
						subsequentlyTrades += txns[j].Amount
					}
				}
				fmt.Fprintf(f, ",%4.2f", subsequentlyTrades)

				// What happens to price 1 min later
				level1Min := math.NaN()
				for j = i; j < len(txns) && txns[j].TimeStamp.Before(t.TimeStamp.Add(time.Minute)); j++ {
				}
				if j < len(txns) {
					level1Min = txns[j].Price
				}

				// And 5 min later
				level5Min := math.NaN()
				profit5Min := math.NaN()
				for ; j < len(txns) && txns[j].TimeStamp.Before(t.TimeStamp.Add(5*time.Minute)); j++ {
				}
				if j < len(txns) {
					level5Min = txns[j].Price
					if trnBuyAmount > trnSellAmount {
						profit5Min = txns[j].Price - t.Price
					} else {
						profit5Min = t.Price - txns[j].Price
					}
				}
				cumPnl += profit5Min
				fmt.Fprintf(f, ",%7.2f,%7.2f,%6.2f,%6.2f\n", level1Min, level5Min, profit5Min, cumPnl)

				//showPriceAction(txns[j:i])

				lastReportedLevel = txns[i].Price
			}
		}
	}
	f.Close()
}

func showPriceAction(txns Transactions) {
	for _, tx := range txns {
		fmt.Printf("%v,%f,%f\n", tx.TimeStamp, tx.Price, tx.Amount)
	}
}
