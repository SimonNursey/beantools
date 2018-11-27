package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"time"

	. "bean"

	bean "bean/rpc"
)

// Fixings ... object contains fixings data, able to calculate realised vol
type Fixings struct {
	Fix []float64
	T   []time.Time
}

const dateLayout = "02/Jan/06 15:04:05"
const includeWeekends = true

func Fixing(mds bean.RPCMDSConnC, pair Pair, T time.Time) float64 {
	// modify to take transaction data first

	if !includeWeekends && (T.Weekday() == time.Sunday || T.Weekday() == time.Saturday) {
		return math.NaN()
	}

	// Fixing window 1 minute before and after
	start := T.Add(time.Duration(-1) * time.Minute)
	end := T.Add(time.Duration(1) * time.Minute)

	// Get VWAP transaction data first
	sum := 0.0
	amt := 0.0
	txns, _ := mds.GetTransactions(pair, start, end)
	for _, t := range txns {
		sum += t.Price * t.Amount
		amt += t.Amount
	}

	if amt > 0.1 {
		return sum / amt
	} else {
		// if nothing of size traded then take price quotes
		obts, _ := mds.GetOrderBookTS(pair, start, end, 20)
		if len(obts) == 0 {
			// no prices in the period
			return math.NaN()
		}
		sum := 0.0
		samples := 0
		for _, obt := range obts {
			if len(obt.OB.Bids) > 0 && len(obt.OB.Asks) > 0 {
				// must be 2way price
				mid := (obt.OB.Bids[0].Price + obt.OB.Asks[0].Price) / 2.0
				sum += mid
				samples++
			}
		}
		if samples == 0 {
			return math.NaN()
		}
		return sum / float64(samples)
	}
}

// LoadFixingsFromBean ... create a new fixings object loaded with fixing data from the bean mds
func LoadFixingsFromBean(mds bean.RPCMDSConnC, pair Pair, start time.Time, end time.Time, freq time.Duration) Fixings {
	// Load fixings for a range of dates direct from bean and return a fixings object
	fh := Fixings{
		make([]float64, 0),
		make([]time.Time, 0)}

	for d := start; d.Before(end); d = d.Add(freq) {
		fix := Fixing(mds, pair, d)
		if !math.IsNaN(fix) {
			fh.Fix, fh.T = append(fh.Fix, fix), append(fh.T, d)
		}
	}
	return fh
}

// Save ... fixings in a csv format Date, fixing
func (fh *Fixings) Save(filename string) {

	//	f, _ := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0)
	os.Remove(filename)
	f, _ := os.Create(filename)
	csvfile := csv.NewWriter(f)

	defer f.Close()

	for i := range fh.T {
		line := []string{fh.T[i].Format(dateLayout), fmt.Sprintf("%6.1f", fh.Fix[i])}
		csvfile.Write(line)
		//		f.WriteString("\"" + fh.T[i].Format(DateLayout) + "\"," + fmt.Sprintf("%6.1f", fh.Fix[i]) + "\n")
	}
	csvfile.Flush()
}

// LoadFixingsFromFile ... Load a previous saved fixings file. Should be in csv format Date, fixing
func LoadFixingsFromFile(filename string) Fixings {
	f, _ := os.Open(filename)
	defer f.Close()

	csvrecords, _ := csv.NewReader(f).ReadAll()

	fh := Fixings{make([]float64, 0), make([]time.Time, 0)}

	var v float64
	for _, csvrecord := range csvrecords {
		fmt.Sscanf(csvrecord[1], "%f", &v)
		fh.Fix = append(fh.Fix, v)

		t, _ := time.Parse(dateLayout, csvrecord[0])
		fh.T = append(fh.T, t)
	}
	return fh
}

// HourlyRealised ... stdev of log returns, no mean adjustment
// Annualisation depends on whether we include weekends or assume they are zero vol (as per fx)
func (fh *Fixings) HourlyRealised(t time.Time, periods int) float64 {
	en := 0
	for en = 0; en < len(fh.T) && fh.T[en].Before(t); en++ {
	}
	if en > len(fh.T) {
		return math.NaN()
	}

	st := en - periods
	if st < 0 {
		return math.NaN()
	}

	var volAnnualisation int
	if includeWeekends {
		volAnnualisation = 365
	} else {
		volAnnualisation = 260
	}

	return stdev(log(fh.Fix[st:en])) * math.Sqrt(float64(volAnnualisation*24))
}

// Fixing ... Return a stored fixing for a particular time. Must be exact time
func (fh *Fixings) Fixing(t time.Time) float64 {
	en := 0
	for en = 0; en < len(fh.T) && fh.T[en].Before(t); en++ {
	}
	if !t.Equal(fh.T[en]) {
		return math.NaN()
	}
	return fh.Fix[en]
}

// LastDate ... return the last loaded fixing date/time
func (fh *Fixings) LastDate() time.Time {
	return fh.T[len(fh.T)-1]
}

func log(series []float64) []float64 {
	l := make([]float64, len(series)-1)
	for i := 0; i < len(series)-1; i++ {
		l[i] = math.Log(series[i+1] / series[i])
	}
	return l
}

func stdev(series []float64) float64 {
	sumsq := 0.0
	for _, s := range series {
		sumsq += s * s
	}
	return math.Sqrt(sumsq / float64((len(series) - 1)))
}

func main() {
	loadFromFile := false
	now := time.Now()
	en := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	pair := Pair{BTC, USDT}

	var fixhist Fixings
	if !loadFromFile {
		mds := bean.NewRPCMDSConnC("tcp", bean.MDS_HOST_SG40+":"+bean.MDS_PORT)

		st := time.Date(2018, 9, 1, 0, 00, 00, 00, time.UTC)
		fixhist = LoadFixingsFromBean(mds, pair, st, en, time.Duration(1)*time.Hour)
		fixhist.Save("fixings.csv")
	} else {
		fixhist = LoadFixingsFromFile("fixings.csv")
	}

	en = fixhist.LastDate()
	fmt.Printf("Hourly realised\n")
	fmt.Printf("Date/time(UTC)\t\tSpot\t1mth\t2wk\t1wk\t24hr\n")
	for t := en.AddDate(0, 0, -60); t.Before(en) || t.Equal(en); t = t.AddDate(0, 0, 1) {
		fmt.Printf("%v", t.Format("Mon 02Jan06 15:04"))
		fmt.Printf("\t%6.1f", fixhist.Fixing(t))
		for _, periodDays := range [...]int{30, 14, 7, 1} {
			fmt.Printf("\t%5.1f%%", fixhist.HourlyRealised(t, 24*periodDays)*100.0)
		}
		fmt.Printf("\n")
	}
}
