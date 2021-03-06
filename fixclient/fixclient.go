package main

import (
	. "bean"
	fix "beanex/exchange/exchangeimpl/deribitFIX"
	"beanex/risk"
	"fmt"
	"time"

	"github.com/joho/godotenv"
)

const deritest = true

func main() {

	var envFile string
	if deritest {
		envFile = "deribittest.env"
	} else {
		envFile = "deribit.env"
	}

	err := godotenv.Overload(envFile)
	if err != nil {
		panic("Error loading deribit.env file")
	}

	exch, err := fix.NewDeribitExchange(deritest)
	if err != nil {
		fmt.Printf("Error creating deribit client:%s", err.Error())
		return
	}
	err = exch.Start()
	if err != nil {
		fmt.Printf("Error starting deribit: %s\n", err.Error())
		return
	}

	time.Sleep(5 * time.Second)

	/*	mkt := risk.NewMarket(Pair{BTC, USD})
		mktMon := func(ch chan *fix.MarketDataNotification) {
			for md := range ch {
				con, _ := ContractFromName(md.Instrument)
				obt := mkt.GetOBT(con)
				if orderBookUpdate(obt, md) {
					bid := obt.BestBid()
					ask := obt.BestAsk()
					transmitLag := md.RecTime.Sub(obt.Time)
					processLag := time.Now().Sub(md.RecTime)
					fmt.Printf("Lag:%v+%v ms %s:%8.2f/%8.2f in %6.0f/%6.0f\n",
						transmitLag.Nanoseconds()/1e3, processLag.Nanoseconds()/1e3,
						con.Name(), bid.Price, ask.Price, bid.Amount, ask.Amount)
				}
			}
		}

		futch, err := exch.MarketDataRequest([]string{"BTC-PERPETUAL"}) //, "BTC-27MAR20"})
		if err != nil {
			fmt.Printf("Cannnot request market data %s\n", err.Error())
			return
		}
		go mktMon(futch)
	*/

	stop := time.NewTimer(20 * time.Second)

	go func() {
		ptf, err := risk.DeribitPortfolio(exch)
		if err != nil {
			fmt.Printf("Error getting position:%s", err.Error())
		}
		for _, p := range ptf.Positions() {
			fmt.Println(p)
		}
		ptf.ShowBrief()

		stop.Reset(0)
	}()

	/*	id, ch, err := exch.NewOrder("BTC-PERPETUAL", 100.0, 10000.0, SELL)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		go func() {
			for update := range ch {
				fmt.Printf("Order update:%s / %s status:%v\n", update.DeriID, update.MyID, update.Status)
			}
			fmt.Println("Order channel closed")
		}()

		time.Sleep(5 * time.Second)

		err = exch.EditOrder(id, "BTC-PERPETUAL", 100.0, 10020.0, SELL)
		if err != nil {
			fmt.Println(err.Error())
		}

		time.Sleep(5 * time.Second)

		err = exch.CancelOrder(id, SELL)
		if err != nil {
			fmt.Println(err.Error())
		}
	*/
	for {
		select {
		case <-stop.C:
			fmt.Println("Time's up")
			exch.Stop()
			return
		case err := <-exch.Err:
			fmt.Printf("ERROR: %s\n", err.Error())
			exch.Stop()
			return
		}
	}

}

func orderBookUpdate(obt *OrderBookT, obNotif *fix.MarketDataNotification) (chg bool) {

	for _, act := range obNotif.Act {
		switch act.Update {
		case fix.New:
			if act.BidAsk == fix.BID {
				chg = obt.InsertBid(Order{Price: act.Price, Amount: act.Qty}) || chg
			} else {
				chg = obt.InsertAsk(Order{Price: act.Price, Amount: act.Qty}) || chg
			}
		case fix.Change:
			if act.BidAsk == fix.BID {
				chg = obt.EditBid(Order{Price: act.Price, Amount: act.Qty}) || chg
			} else {
				chg = obt.EditAsk(Order{Price: act.Price, Amount: act.Qty}) || chg
			}
		case fix.Delete:
			if act.BidAsk == fix.BID {
				chg = obt.CancelBid(Order{Price: act.Price, Amount: act.Qty}) || chg
			} else {
				chg = obt.CancelAsk(Order{Price: act.Price, Amount: act.Qty}) || chg
			}

		}
		obt.Time = act.TimeStamp

	}
	return
}
