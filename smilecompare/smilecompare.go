package main

import (
	"beanex/risk"
	"fmt"
	"time"
)

func main() {
	//	atm, rr, fly := 0.60, 0.08, 0.04
	spot := 8000.0
	forward := 8010.0
	expiryDays := int(time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC).Sub(time.Now()).Hours() / 24.0)

	mktStrikes := []float64{5000, 5500, 6000, 6500, 7000, 7500, 8000, 8500, 9000, 9500, 10000, 10500}
	mktBids := []float64{71.5, 66.7, 61.5, 56.8, 55.3, 53.1, 54.2, 55.1, 57.8, 60.2, 63.6, 66.3}
	mktAsks := []float64{76.9, 70, 63.7, 58.9, 56.6, 56, 56.1, 56.2, 60, 63.1, 66.1, 69.5}

	smiles := []risk.Smile{
		risk.NewFivePointSmile(),
		risk.NewClampedFivePointSmile(),
		risk.NewSevenPointSmile(),
	}

	for i := range mktBids {
		mktBids[i] /= 100.0
	}
	for i := range mktAsks {
		mktAsks[i] /= 100.0
	}

	for i := range smiles {
		//		smiles[i].SetParams([]float64{atm, rr, fly})
		smiles[i].Solve(spot, forward, expiryDays, mktStrikes, mktBids, mktAsks)
	}

	atm, _, _, _, _ := smiles[0].AtmRRFly()

	//	var deltas, strikes [99]float64
	deltas := make([]float64, 99)
	strikes := make([]float64, len(deltas))
	for i := range deltas {
		deltas[i] = float64(i)/100.0 + 0.01
		strikes[i] = risk.SimpleDeltaToStrike(expiryDays, deltas[i], spot, forward, atm)
	}

	vols := make([][]float64, len(smiles))
	for i := range smiles {
		vols[i] = smiles[i].Interp(spot, forward, expiryDays, strikes)
	}

	fmt.Printf("Delta,Strike,5Pt,C5Pt,7Pt,Bid,Ask\n")
	for i := range deltas {
		fmt.Printf("%.1f,%.0f,", deltas[i]*100.0, strikes[i])
		for j := range vols {
			fmt.Printf("%.1f,", vols[j][i]*100.0)
		}
		fmt.Printf("\n")
	}
	for i := range mktStrikes {
		fmt.Printf(",%.0f,,,,%.1f,%.1f\n", mktStrikes[i], mktBids[i]*100.0, mktAsks[i]*100.0)
	}
}
