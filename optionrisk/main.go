package main

import (
	. "bean"
	"bean/rpc"
	"beanex/telegram"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type callOrPut string

const (
	Call callOrPut = "C"
	Put  callOrPut = "P"
)

const USDiscountRate = 0.02

/*type contract interface {
	Name() string
	PV(asof time.Time,spot,future,vol float64)
	Delta(mds bean.rpcmdsConnC,asof time.Time)
}*/

func main() {
	mds := bean.NewRPCMDSConnC("tcp", bean.MDS_HOST_SG40+":"+bean.MDS_PORT)

	const telegramSimon = 773309642
	const telegramRealisedVolGroup = -345918701
	//	telegramRealisedVolUpdate(Pair{BTC, USDT}, []int64{telegramRealisedVolGroup})

	//	fmt.Printf("%s", riskSummary(mds, time.Now().Add(-1*time.Minute)))
	//	go telegramRiskUpdate(mds, []int64{telegramSimon})
	//	go telegramMarketUpdate(mds, []int64{telegramSimon})
	//	time.Sleep(1 * 24 * time.Hour)
	//	liveOptionMarket(mds)
	fmt.Printf("%s", riskLadder(mds, time.Now().Add(-1*time.Minute)))

}

func telegramRiskUpdate(mds bean.RPCMDSConnC, uids []int64) {
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	botID := "714436587:AAGPvl9CdJt04M0DnIvtD3iv7IStVaWZ7p0" // nurseybot
	os.Setenv("TG_NURSEY_BOT", botID)

	// Recalculate every hour, forever
	for {
		// Wait for 5 minutes past the hour (due to delays consolidating)
		for time.Now().Before(t.Add(5 * time.Minute)) {
			time.Sleep(time.Minute)
		}

		s := riskSummary(mds, t)
		telegram.SendMsgRaw("TG_NURSEY_BOT", uids, s)
		fmt.Printf("%s", s)

		t = t.Add(1 * time.Hour)
	}
}

func telegramMarketUpdate(mds bean.RPCMDSConnC, uids []int64) {
	now := time.Now()
	t := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	botID := "714436587:AAGPvl9CdJt04M0DnIvtD3iv7IStVaWZ7p0" // nurseybot
	os.Setenv("TG_NURSEY_BOT", botID)

	// Recalculate every hour, forever
	for {
		// Wait for 5 minutes past the hour (due to delays consolidating)
		for time.Now().Before(t.Add(5 * time.Minute)) {
			time.Sleep(time.Minute)
		}

		s := optMarketSummary(mds, t)
		telegram.SendMsgRaw("TG_NURSEY_BOT", uids, s)
		fmt.Printf("%s", s)

		t = t.Add(1 * time.Hour)
	}
}

func getLivePtf() []OptContract {
	return []OptContract{
		{"BTC-29MAR19-2750-P", 4.0},
		{"BTC-29MAR19-4250-C", -2.0},
		{"BTC-29MAR19-4500-C", -2.0},
		{"BTC-29MAR19-6000-C", -2.0}}
}

func riskSummary(mds bean.RPCMDSConnC, t time.Time) string {
	ptf := getLivePtf()

	var output strings.Builder
	pvSum, deltaSum, vegaSum := 0.0, 0.0, 0.0
	spotMid := SpotPrice(mds, t, ptf[0].Underlying())

	for i, opt := range ptf {
		futMid, optBid, optAsk := FutureOptionPrice(mds, t, opt)
		domRate := USDiscountRate
		volBid := opt.ImpVol(t, spotMid, futMid, domRate, optBid)
		volAsk := opt.ImpVol(t, spotMid, futMid, domRate, optAsk)
		volMid := (volBid + volAsk) / 2.0

		pv := opt.PV(t, spotMid, futMid, domRate, volMid)
		delta := opt.Delta(t, spotMid, futMid, volMid, domRate)
		vega := opt.Vega(t, spotMid, futMid, volMid, domRate)

		pvSum += pv
		deltaSum += delta
		vegaSum += vega

		if i == 0 {
			fmt.Fprintf(&output, "%s   %6.1f\n", t.Format("Mon 02Jan06 15:04"), spotMid)
			fmt.Fprintf(&output, "Contract Qty\nPV(BTC) PV(USD) DELTA(BTC) VEGA(USD)\n")
		}
		fmt.Fprintf(&output, "%s %4.1f\n%6.3f %6.1f %6.3f %5.1f\n", opt.Name(), opt.Quantity(), pv/spotMid, pv, delta, vega)
	}
	fmt.Fprintf(&output, "TOTAL\n%6.3f %6.1f %6.3f %5.1f\n", pvSum/spotMid, pvSum, deltaSum, vegaSum)
	return output.String()
}

func riskLadder(mds bean.RPCMDSConnC, t time.Time) string {
	ptf := getLivePtf()
	var output strings.Builder
	spotBump := [10]float64{-0.50, -0.25, -0.10, -0.05, 0.0, 0.05, 0.10, 0.25, 0.50, 1.0}
	var pv, delta, vega [len(spotBump)]float64

	spotMid := SpotPrice(mds, t, ptf[0].Underlying())

	for _, opt := range ptf {
		futMid, optBid, optAsk := FutureOptionPrice(mds, t, opt)
		domRate := USDiscountRate
		volBid := opt.ImpVol(t, spotMid, futMid, domRate, optBid)
		volAsk := opt.ImpVol(t, spotMid, futMid, domRate, optAsk)
		volMid := (volBid + volAsk) / 2.0

		for j, s := range spotBump {

			pv[j] += opt.PV(t, (1.0+s)*spotMid, (1.0+s)*futMid, domRate, volMid)
			delta[j] += opt.Delta(t, (1.0+s)*spotMid, (1.0+s)*futMid, volMid, domRate)
			vega[j] += opt.Vega(t, (1.0+s)*spotMid, (1.0+s)*futMid, volMid, domRate)
		}
	}

	for g := 0; g < 5; g++ {
		switch g {
		case 0:
			fmt.Fprintf(&output, "      ")
		case 1:
			fmt.Fprintf(&output, "      ")
		case 2:
			fmt.Fprintf(&output, "PV    ")
		case 3:
			fmt.Fprintf(&output, "DELTA ")
		case 4:
			fmt.Fprintf(&output, "VEGA  ")
		}

		for j, s := range spotBump {
			switch g {
			case 0:
				fmt.Fprintf(&output, "%6.2f ", s)
			case 1:
				fmt.Fprintf(&output, "%6.1f ", (1.0+s)*spotMid)
			case 2:
				fmt.Fprintf(&output, "%6.1f ", pv[j])
			case 3:
				fmt.Fprintf(&output, "%6.1f ", delta[j])
			case 4:
				fmt.Fprintf(&output, "%6.1f ", vega[j])
			}
		}
		fmt.Fprintf(&output, "\n")
	}
	return output.String()
}

func optMarketSummary(mds bean.RPCMDSConnC, t time.Time) string {
	benchmarkContracts := []OptContract{
		{"BTC-29MAR19-2000-P", 0.0},
		{"BTC-29MAR19-2500-P", 0.0},
		{"BTC-29MAR19-3000-P", 0.0},
		{"BTC-29MAR19-3500-P", 0.0},
		{"BTC-29MAR19-3500-C", 0.0},
		{"BTC-29MAR19-4000-C", 0.0},
		{"BTC-29MAR19-4500-C", 0.0},
		{"BTC-29MAR19-5000-C", 0.0},
		{"BTC-29MAR19-6000-C", 0.0},
	}
	var output strings.Builder
	spotMid := SpotPrice(mds, t, benchmarkContracts[0].Underlying())

	for i, opt := range benchmarkContracts {
		futMid, optBid, optAsk := FutureOptionPrice(mds, t, opt)
		domRate := USDiscountRate
		volBid := opt.ImpVol(t, spotMid, futMid, domRate, optBid)
		volAsk := opt.ImpVol(t, spotMid, futMid, domRate, optAsk)
		if i == 0 {
			fmt.Fprintf(&output, "%s   %6.1f\n", t.Format("Mon 02Jan06 15:04"), spotMid)
			fmt.Fprintf(&output, "Vol          Prem (BTC)\n")
		}
		fmt.Fprintf(&output, "%s\n%5.1f/%5.1f   %6.4f/%6.4f\n", opt.Name(), volBid*100.0, volAsk*100.0, optBid, optAsk)
	}
	return output.String()
}

func SpotPrice(mds bean.RPCMDSConnC, asof time.Time, p Pair) float64 {
	st := asof.Add(time.Duration(-1) * time.Minute)
	en := asof.Add(time.Duration(1) * time.Minute)

	spotobts, _ := mds.GetOrderBookTS(p, st, en, 20)

	return midPriceAt(spotobts, asof)
}

func FutureOptionPrice(mds bean.RPCMDSConnC, asof time.Time, c OptContract) (futMid, optionBid, optionAsk float64) {
	st := asof.Add(time.Duration(-1) * time.Minute)
	en := asof.Add(time.Duration(1) * time.Minute)

	futContract := c.Name()[0:11]

	futobts, _ := mds.GetFutOrderBookTS(futContract, st, en, 20)
	optobts, _ := mds.GetOptOrderBookTS(c.name, st, en, 20)

	futMid = midPriceAt(futobts, asof)
	optionBid, optionAsk = priceAt(optobts, asof)
	return
}

func midPriceAt(obts OrderBookTS, fix time.Time) float64 {
	i := 0
	for ; i < len(obts) && obts[i].Time.Before(fix); i++ {
	}
	if i >= len(obts) {
		return math.NaN()
	}
	return (obts[i].OB.Bids[0].Price + obts[i].OB.Asks[0].Price) / 2.0
}

func priceAt(obts OrderBookTS, fix time.Time) (bid, ask float64) {
	i := 0
	for ; i < len(obts) && obts[i].Time.Before(fix); i++ {
	}
	if i >= len(obts) {
		return math.NaN(), math.NaN()
	}
	bid = obts[i].OB.Bids[0].Price
	ask = obts[i].OB.Asks[0].Price
	return
}

type OptContract struct {
	name string
	qty  float64
}

func (c OptContract) Name() string {
	return c.name
}

func (c OptContract) Quantity() float64 {
	return c.qty
}

func (c OptContract) expiry() (dt time.Time) {
	dt, _ = time.Parse("02Jan06", strings.ToTitle(strings.Split(c.name, "-")[1]))
	dt = time.Date(dt.Year(), dt.Month(), dt.Day(), 9, 0, 0, 0, time.UTC) // 9am london expiry
	return
}

func (c OptContract) strike() (st float64) {
	st, _ = strconv.ParseFloat(strings.Split(c.name, "-")[2], 64)
	return
}

func (c OptContract) callPut() callOrPut {
	switch strings.Split(c.name, "-")[3] {
	case "C":
		return Call
	case "P":
		return Put
	}
	panic("Need C OR P")
}

func (c OptContract) Underlying() Pair {
	switch strings.Split(c.name, "-")[0] {
	case "BTC":
		return Pair{BTC, USDT}
	}
	panic("Only accept BTC underlying")
}

func (c OptContract) ImpVol(asof time.Time, spotPrice, futPrice, domRate, optionPrice float64) float64 {
	expiry := c.expiry()
	strike := c.strike()
	cp := c.callPut()
	expiryDays := dayDiff(asof, expiry)
	deliveryDays := expiryDays // temp

	return optionImpliedVol(expiryDays, deliveryDays, strike, futPrice, domRate, optionPrice*spotPrice, cp)
}

// in fiat
func (c OptContract) PV(asof time.Time, spotPrice, futPrice, domRate, vol float64) float64 {
	expiry := c.expiry()
	strike := c.strike()
	cp := c.callPut()
	expiryDays := dayDiff(asof, expiry)
	deliveryDays := expiryDays // temp
	return c.Quantity() * forwardOptionPrice(expiryDays, strike, futPrice, vol, cp) * dF(deliveryDays, domRate)
}

// in fiat
func (c OptContract) Vega(asof time.Time, spotPrice, futPrice, vol, domRate float64) float64 {
	return c.PV(asof, spotPrice, futPrice, domRate, vol+0.005) - c.PV(asof, spotPrice, futPrice, domRate, vol-0.005)
}

//in coin
func (c OptContract) Delta(asof time.Time, spotPrice, futPrice, vol, domRate float64) float64 {
	deltaFiat := (c.PV(asof, spotPrice*1.005, futPrice*1.005, domRate, vol) - c.PV(asof, spotPrice*0.995, futPrice*0.995, domRate, vol)) * 100.0
	return deltaFiat / spotPrice
}

// maths stuff now

// day difference rounded.
func dayDiff(t1, t2 time.Time) int {
	t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC) // remove time information and force to utc
	t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
	return int(math.Round(t2.Sub(t1).Truncate(time.Hour).Hours() / 24.0))
}

func optionImpliedVol(expiryDays, deliveryDays int, strike, forward, domRate, prm float64, callPut callOrPut) (bs float64) {
	// newton raphson on vega and bs
	//	guessVol := math.Sqrt(2.0*math.Pi/(float64(expiryDays)/365)) * prm / forward
	guessVol := 0.80
	for i := 0; i < 1000; i++ {
		guessPrm := dF(deliveryDays, domRate) * forwardOptionPrice(expiryDays, strike, forward, guessVol, callPut)
		vega := optionVega(expiryDays, deliveryDays, strike, forward, guessVol, domRate)
		guessVol = guessVol - (guessPrm-prm)/(vega*100.0)
		if guessPrm/prm < 1.00001 && guessPrm/prm > 0.99999 {
			return guessVol
		}
	}
	return math.NaN()
}

func dF(days int, rate float64) float64 {
	return math.Exp(-float64(days) / 365 * rate)
}

// in fiat
func forwardOptionPrice(expiryDays int, strike, forward, vol float64, callPut callOrPut) (prm float64) {
	d1 := (math.Log(forward/strike) + (vol*vol/2.0)*(float64(expiryDays)/365)) / (vol * math.Sqrt(float64(expiryDays)/365))
	d2 := d1 - vol*math.Sqrt(float64(expiryDays)/365.0)

	if callPut == Call {
		prm = forward*cumNormDist(d1) - strike*cumNormDist(d2)
	} else {
		prm = -forward*cumNormDist(-d1) + strike*cumNormDist(-d2)
	}
	return
}

// Seems to work!
func cumNormDist(x float64) float64 {
	return 0.5 * math.Erfc(-x/math.Sqrt(2))
}

func optionVega(expiryDays, deliveryDays int, strike, forward, vol, domRate float64) float64 {
	//	d1 := (math.Log(forward/strike) + (vol*vol/2.0)*(float64(expiryDays)/365)) / (vol * math.Sqrt(float64(expiryDays)/365))
	//	return forward * cumNormDist(d1) * math.Sqrt(float64(expiryDays)/365.0) * dF(deliveryDays, domRate)
	return dF(deliveryDays, domRate) * (forwardOptionPrice(expiryDays, strike, forward, vol+0.005, Call) - forwardOptionPrice(expiryDays, strike, forward, vol-0.005, Call))
}

func optionDelta(expiryDays, deliveryDays int, callPut callOrPut, strike, forward, vol, domRate float64) float64 {
	//	d1 := (math.Log(forward/strike) + (vol*vol/2.0)*(float64(expiryDays)/365)) / (vol * math.Sqrt(float64(expiryDays)/365))
	//	return forward * cumNormDist(d1) * math.Sqrt(float64(expiryDays)/365.0) * dF(deliveryDays, domRate)
	return dF(deliveryDays, domRate) * (forwardOptionPrice(expiryDays, strike, forward, vol, callPut) - forwardOptionPrice(expiryDays, strike, forward, vol, callPut))
}
