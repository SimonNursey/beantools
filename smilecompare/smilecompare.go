package main

import (
	"beanex/market"
	"fmt"
)

func main() {
	//	atm, rr, fly := 0.60, 0.08, 0.04
	spot := 8001.25
	forward := 8041.0
	expiryDays := 22

	mktStrikes := []float64{6000, 6500, 7000, 7500, 8000, 8500, 9000, 9500, 10000, 10500, 11000}
	mktBids := []float64{0.7594294886024444, 0.7001145298013258, 0.6532190309055066, 0.6316866165183694, 0.6302766309814991, 0.6258575200773457, 0.6582802722760192, 0.6935341244134169, 0.7337009912761131, 0.7772609041902822, 0.8189136195548339}
	mktAsks := []float64{0.7791704606156098, 0.7122155702910425, 0.6611594311863958, 0.6433943383451804, 0.6456860960366261, 0.6576567983173176, 0.6827609495853916, 0.7086085753208514, 0.7524805613170765, 0.7890178421910498, 0.8333837252360602}

	smiles := []market.Smile{
		//		market.NewFivePointSmile(),
		market.NewSevenPointSmile(),
	}

	/*	for i := range mktBids {
			mktBids[i] /= 100.0
		}
		for i := range mktAsks {
			mktAsks[i] /= 100.0
		}
	*/
	for i := range smiles {
		//		smiles[i].SetParams([]float64{atm, rr, fly})
		err := smiles[i].Solve(spot, forward, expiryDays, mktStrikes, mktBids, mktAsks)
		if err != nil {
			fmt.Printf("%v : %s\n", i, err.Error())
		}
	}

	atm, _, _, _, _ := smiles[0].AtmRRFly()

	//	var deltas, strikes [99]float64
	deltas := make([]float64, 99)
	strikes := make([]float64, len(deltas))
	for i := range deltas {
		deltas[i] = float64(i)/100.0 + 0.01
		strikes[i] = market.SimpleDeltaToStrike(expiryDays, deltas[i], spot, forward, atm)
	}

	vols := make([][]float64, len(smiles))
	for i := range smiles {
		vols[i] = smiles[i].Interp(spot, forward, expiryDays, strikes)
	}

	for j := range vols {
		fmt.Println(smiles[j].Format())
	}

	/*	fmt.Printf("Delta,Strike,5Pt,C5Pt,7Pt,Bid,Ask\n")
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
	*/
}
