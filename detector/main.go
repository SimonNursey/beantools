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
	mds := bean.NewRPCMDSConnC("tcp", bean.MDS_HOST_SG40+":"+bean.MDS_PORT)
	st := time.Date(2018, 12, 2, 0, 0, 0, 0, time.UTC)
	en := time.Date(2018, 12, 3, 0, 0, 0, 0, time.UTC)
	pair := Pair{BTC, USDT}
	analyseMAPG(st, en, pair, mds)
	//analyseStaticOrders(st, en, pair, mds)
}

func analyseStaticOrders(st time.Time, en time.Time, pair Pair, mds bean.RPCMDSConnC) {
	timeWindow := time.Duration(5) * time.Minute // the historic window in which to scan for orders and transactions at the same level
	// set very high. looking at mapg only.
	amountThreshold := 10.0 // this is the minimum total traded amount at a specific level
	stealthFactor := 2.0    // this is the minimum ratio between the total traded amount and the maximum order amount or maximum trade amount

	fIceBerg, _ := os.Create("detector.csv")
	fmt.Fprintf(fIceBerg, "Type,Time,Level,Traded Amount,Max Trade,Trade Count,Given,Paid,Max Order,Bids,Asks,Subsequent Trades,1min,5min,Profit,Cum Pnl\n")

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

				fmt.Fprintf(fIceBerg, "ICEBERG,%v,%7.2f,%5.2f,%5.2f,%2v,%5.2f,%5.2f",
					t.TimeStamp.Format(dateLayout), t.Price, trnTotalAmount, trnMaxAmount, trnCount, trnBuyAmount, trnSellAmount)
				fmt.Fprintf(fIceBerg, ",%5.2f,%2v,%2v", orderMaxAmount, orderBids, orderAsks)

				// How much subsequently trades at that level
				subsequentlyTrades := 0.0
				var j int
				for j = i + 1; j < len(txns) && txns[j].TimeStamp.Before(t.TimeStamp.Add(5*timeWindow)); j++ {
					if txns[j].Price == t.Price {
						subsequentlyTrades += txns[j].Amount
					}
				}
				fmt.Fprintf(fIceBerg, ",%4.2f", subsequentlyTrades)

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
				fmt.Fprintf(fIceBerg, ",%7.2f,%7.2f,%6.2f,%6.2f\n", level1Min, level5Min, profit5Min, cumPnl)

				//showPriceAction(txns[j:i])

				lastReportedLevel = txns[i].Price
			}
		}
	}
	fIceBerg.Close()
}

// MAPG - track the moving net paid given amounts within a window. Try to find correlation with subsequent price movement.
func analyseMAPG(st time.Time, en time.Time, pair Pair, mds bean.RPCMDSConnC) {
	fMAPG, _ := os.Create("MAPG.csv")
	fmt.Fprintf(fMAPG, "Time,Last trade,1min prior,1min MNPG,5min ENPG,20min ENPG,next 1min,next 5min\n")

	nextSample := st
	sampleRate := time.Minute

	txns, _ := mds.GetTransactions(pair, st, en)

	for i, t := range txns {
		if nextSample.Before(t.TimeStamp) {
			for ; nextSample.Before(t.TimeStamp); nextSample = nextSample.Add(sampleRate) {
			}
			MNPG1min, _ := movingNetPaidGiven(txns[:i], t.TimeStamp, time.Minute)
			//			MNPG5min, _ := movingNetPaidGiven(txns[:i], t.TimeStamp, 5*time.Minute)
			ENPG5min, _ := expNetPaidGiven(txns[:i], t.TimeStamp, 5*time.Minute)
			ENPG20min, _ := expNetPaidGiven(txns[:i], t.TimeStamp, 20*time.Minute)
			priceMove1min := txnPriceLater(txns[i:], time.Minute) - t.Price
			priceMove5min := txnPriceLater(txns[i:], 5*time.Minute) - t.Price
			priceMovePrior1min := t.Price - txnPriceEarlier(txns[:i], time.Minute)
			if !math.IsNaN(priceMovePrior1min) && !math.IsNaN(priceMove5min) {
				fmt.Fprintf(fMAPG, "%s,%7.2f,", t.TimeStamp.Format(dateLayout), t.Price)
				fmt.Fprintf(fMAPG, "%4.2f,%3.2f,%3.2f,%3.2f,", priceMovePrior1min, MNPG1min, ENPG5min, ENPG20min)
				fmt.Fprintf(fMAPG, "%4.2f,%4.2f\n", priceMove1min, priceMove5min)
			}
		}
	}
	fMAPG.Close()
}

func showPriceAction(txns Transactions) {
	for _, tx := range txns {
		fmt.Printf("%v,%f,%f\n", tx.TimeStamp, tx.Price, tx.Amount)
	}
}

func movingNetPaidGiven(txns Transactions, evalTime time.Time, period time.Duration) (float64, float64) {
	if len(txns) == 0 {
		return 0, math.NaN()
	}
	totVolPaid := 0.0
	totVol := 0.0
	st := evalTime.Add(-period)
	for i := len(txns) - 1; i >= 0 && txns[i].TimeStamp.After(st); i-- {
		totVol += txns[i].Amount
		if txns[i].Maker == TraderType(Buyer) {
			totVolPaid -= txns[i].Amount
		} else {
			totVolPaid += txns[i].Amount
		}
	}
	return totVolPaid, totVol
}

func expNetPaidGiven(txns Transactions, evalTime time.Time, halfLife time.Duration) (float64, float64) {
	if len(txns) == 0 {
		return 0, math.NaN()
	}
	totVolPaid := 0.0
	totVol := 0.0
	st := evalTime.Add(-5 * halfLife)
	for i := len(txns) - 1; i >= 0 && txns[i].TimeStamp.After(st); i-- {
		tDiff := evalTime.Sub(txns[i].TimeStamp)

		if tDiff > 0.0 {
			amt := txns[i].Amount * math.Exp(-tDiff.Seconds()/halfLife.Seconds())
			totVol += amt
			if txns[i].Maker == TraderType(Buyer) {
				totVolPaid -= amt
			} else {
				totVolPaid += amt
			}
		}
	}
	return totVolPaid, totVol
}

func txnPriceLater(txns Transactions, period time.Duration) float64 {
	var i int
	if len(txns) == 0 {
		return math.NaN()
	}
	t := txns[0].TimeStamp.Add(period)
	for i = 0; i < len(txns) && t.After(txns[i].TimeStamp); i++ {
	}
	if i == len(txns) {
		return math.NaN()
	}
	return txns[i].Price
}

func txnPriceEarlier(txns Transactions, period time.Duration) float64 {
	var i int
	if len(txns) == 0 {
		return math.NaN()
	}
	t := txns[len(txns)-1].TimeStamp.Add(-period)
	for i = len(txns) - 1; i >= 0 && t.Before(txns[i].TimeStamp); i-- {
	}
	if i < 0 {
		return math.NaN()
	}
	return txns[i].Price
}
