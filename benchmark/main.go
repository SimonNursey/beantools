package main

import (
	. "bean"
	deribit "beanex/exchange/exchangeimpl/go-deribit2"
	risk "beanex/risk"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const deritest = false

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

	errs := make(chan error)
	stop := make(chan bool)
	e, err := deribit.NewExchange(os.Getenv("DERIBIT_API_KEY"), os.Getenv("DERIBIT_SECRET"), deritest, errs, stop)

	if err != nil {
		log.Fatalf("Error creating connection: %s", err)
	}
	if err := e.Connect(); err != nil {
		log.Fatalf("Error connecting to exchange: %s", err)
	}
	go func() {
		for err := range errs {
			fmt.Printf("RPC error: %s\n", err.Error())
		}
	}()

	err = e.Auth()
	if err != nil {
		log.Printf("Couldn't authorise. Error %s", err.Error())
	}

	/*	optInstrs, err := ex.GetInstruments(coin, false, true)
		if err != nil {
			teleChan <- "Couldn't get instrument list:" + err.Error()
			stop <- true
			return
		}*/

	const instrument = "BTC-PERPETUAL"

	bookNotif, err := e.SubscribeOrderBook([]string{instrument})
	if err != nil {
		log.Printf("subscription err:%s\n", err.Error())
		return
	}

	/*	quoteNotif, err := e.SubscribeQuote([]string{instrument})
		if err != nil {
			log.Printf("subscription err:%s\n", err.Error())
			return
		}*/

	/*	tradeNotif, err := e.SubscribeTrades([]string{"BTC-PERPETUAL"})
		if err != nil {
			log.Printf("subscription err:%s\n", err.Error())
			return
		}
	*/
	/*	heartbeat, err = e.RequestHeartBeat(10)
		if err != nil {
			log.Printf("Heartbeat problem " + err.Error())
		}
	*/
	//	count := 0
	lastnotifid := int64(0)
	obt := OrderBookT{OrderBook: risk.EmptyOrderBook2()}
	totalLag := 0.0
	const iterations = 100

	for i := 0; i < iterations; i++ {
		select {
		case book := <-bookNotif:
			//lag := time.Now().Sub(time.Unix(book.Timestamp/1000, book.Timestamp%1000*1e6))
			if lastnotifid != book.PrevChangeID {
				fmt.Printf("id not matching\n")
			}
			lastnotifid = book.ChangeID

			/*			instrument := book.Instrument
						con, err := ContractFromName(instrument)

						obt := mkt.GetOBT(con)
						err, chg := processOBNotif(obt, qu)
			*/
			processOBNotif(&obt, book)
			obt.Time = time.Unix(book.Timestamp/1000, book.Timestamp%1000*1e6)
			lag := time.Now().Sub(obt.Time)
			totalLag += lag.Seconds()
			fmt.Printf("%s (%5.4f) BOOK - %8.2f/%8.2f in %6.0f/%6.0f\n",
				obt.Time.Format("15:04:05.0000"), lag.Seconds(),
				obt.BestBid().Price, obt.BestAsk().Price,
				obt.BestBid().Amount, obt.BestAsk().Amount)

		}
	}
	fmt.Printf("Mean lag %6.4f\n", totalLag/float64(iterations))
	stop <- true
}

func processOBNotifSide(instrument string, side string, tuples [][3]interface{}, insert, edit, cancel func(Order) bool, timeStamp time.Time) bool {
	tobChange := false
	for _, tuple := range tuples {
		action := tuple[0].(string)
		ord := Order{Price: tuple[1].(float64), Amount: tuple[2].(float64)}
		switch action {
		case "new":
			if insert(ord) {
				tobChange = true
			}
		case "change":
			if edit(ord) {
				tobChange = true
			}
		case "delete":
			if cancel(ord) {
				tobChange = true
			}
		default:
			log.Print("Unrecognised action" + action)
		}
		/*		msgLog <- mds.MessagePoint{
				TimeStamp:  timeStamp,
				Instrument: instrument,
				Type:       "BOOK",
				Message:    fmt.Sprintf("BID/%s/%f/%f", action, ord.Price, ord.Amount)}
		*/
	}
	return tobChange
}

func processOBNotif(obt *OrderBookT, obNotif *deribit.OrderBookNotification) (error, bool) {
	if obt.ChangeId != obNotif.PrevChangeID {
		return errors.New("Change ID doesn't match"), false
	}
	obt.ChangeId = obNotif.ChangeID
	obt.Time = time.Unix(obNotif.Timestamp/1000, obNotif.Timestamp%1000*1e6)

	c1 := processOBNotifSide(obNotif.Instrument, "BID", obNotif.Bids, obt.InsertBid, obt.EditBid, obt.CancelBid, obt.Time)
	c2 := processOBNotifSide(obNotif.Instrument, "ASK", obNotif.Asks, obt.InsertAsk, obt.EditAsk, obt.CancelAsk, obt.Time)

	obt.Time = time.Unix(obNotif.Timestamp/1000, obNotif.Timestamp%1000*1e6)
	return nil, c1 || c2
}
