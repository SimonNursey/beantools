package main

import (
	. "bean"
	"beanex/db/mds"
	deribit "beanex/exchange/exchangeimpl/go-deribit"
	"beanex/risk"
	"fmt"
	"log"
	"os"
	"time"
	//	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

	mds := mds.NewMDS(NameDeribit, mds.DB_HOST_BEANEX_SG_40, "8086")

	errs, stop := make(chan error), make(chan bool)
	deribit, err := deribit.NewExchange(os.Getenv("DERIBIT_API_KEY"), os.Getenv("DERIBIT_SECRET"), false, errs, stop)
	if err != nil {
		log.Fatal("Error creating connection")
	}
	if err = deribit.Connect(); err != nil {
		log.Fatalf("Error connecting to exchange: %s", err)
	}

	ptf := qcpPortfolio()
	if err == nil {
		msgText := risk.PortfolioRiskSummary(ptf, mds, time.Now().Add(-time.Minute), Pair{BTC, USDT})
		fmt.Println(msgText)
	}

}

func qcpPortfolio() Portfolio {
	strikes := []float64{3375, 3500, 3625, 3750, 3825, 4000, 4125, 4250, 4375, 4500, 4625}
	amt := []float64{278.8, 281.1, 148.5, 137.8, 22, 0, 139, 122.6, 166.5, 23.4, 10.9}
	cp := []CallOrPut{Put, Put, Put, Put, Put, Put, Call, Call, Call, Call, Call}

	ptf := NewPortfolio(map[Coin]float64{BTC: 0.0}) // temp coin balance
	for i := range strikes {
		c := OptContractFromDets(BTC, time.Date(2019, 3, 29, 10, 0, 0, 0, time.UTC), strikes[i], cp[i])
		ptf.AddContract(c, amt[i])
	}

	strikes = []float64{2000, 2250, 2500, 2750, 3000, 3250, 3500, 3750, 4000, 4250, 4500, 4750, 5000, 5250, 6000, 7000, 7500, 8000, 9000, 10000, 12500, 15000, 20000, 25000, 30000, 35000, 40000}
	amt = []float64{500, 290.8, 1633.9, 266.1, 431.2, 126.6, 10, 30, 36.3, 0, 50, 27.7, 781.1, 40.3, 704.6, 797.9, 676.1, 267.8, 930.1, 117.9, 77.5, 1282.1, 725.5, 58.2, 108, 5.1, 9}
	cp = []CallOrPut{Put, Put, Put, Put, Put, Put, Put, Put, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call, Call}

	for i := range strikes {
		c := OptContractFromDets(BTC, time.Date(2019, 6, 28, 10, 0, 0, 0, time.UTC), strikes[i], cp[i])
		ptf.AddContract(c, amt[i])
	}

	return ptf
}
