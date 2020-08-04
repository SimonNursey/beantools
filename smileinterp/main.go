package main

import (
	"beanex/risk"
	"fmt"
	"time"
)

func main() {
	spot := 7131.0
	forward := 7225.0
	smile := risk.Smile{&risk.FivePointSmile{}}
	smile1 := risk.Smile{&risk.ClampedFivePointSmile{ClampLeft: 1.0, ClampRight: 1.0}}
	smile2 := risk.Smile{&risk.ClampedFivePointSmile{ClampLeft: 1.1, ClampRight: 1.1}}
	smile3 := risk.Smile{&risk.ClampedFivePointSmile{ClampLeft: 1.2, ClampRight: 1.2}}
	smile.SetParams([]float64{0.6918, 0.0305, 0.0265})
	smile1.SetParams(smile.Params())
	smile2.SetParams(smile.Params())
	smile3.SetParams(smile.Params())

	//	smile1 := risk.NewFivePointSmile(0.80, 0.05, 0.025, map[risk.InterpParam]float64{risk.LeftSlope: -0.00005, risk.RightSlope: 0.00005})
	expirydays := int(time.Date(2020, 6, 26, 0, 0, 0, 0, time.UTC).Sub(time.Now()).Hours() / 24)

	//deltas := []float64{0.001, 0.01, 0.05, 0.10, 0.15, 0.25, 0.50, 0.75, 0.85, 0.90, 0.95, 0.99, 0.999}
	deltas := make([]float64, 99)
	for i := range deltas {
		deltas[i] = 0.01 + float64(i)*0.01
	}

	strikes := make([]float64, len(deltas))
	for i := range strikes {
		strikes[i] = risk.SimpleDeltaToStrike(expirydays, deltas[i], spot, forward, smile.Params()[0])
	}

	vols := smile.Interp(spot, forward, expirydays, strikes)
	vols1 := smile1.Interp(spot, forward, expirydays, strikes)
	vols2 := smile2.Interp(spot, forward, expirydays, strikes)
	vols3 := smile3.Interp(spot, forward, expirydays, strikes)

	for i := range strikes {
		fmt.Printf("%3.0f,%7.0f,%5.1f,%5.1f,%5.1f,%5.1f\n", deltas[i]*100.0, strikes[i], vols[i]*100.0, vols1[i]*100.0, vols2[i]*100.0, vols3[i]*100.0)
	}
}
