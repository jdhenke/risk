package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
)

func main() {
	panic(http.ListenAndServe(":"+os.Getenv("PORT"), http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(rw, `<html>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<body>
<a href="https://github.com/jdhenke/risk">Code</a>
<pre>
`)
		writeResults(rw)
		fmt.Fprintf(rw, "</pre></body></html>")
	})))
}

func writeResults(rw http.ResponseWriter) {
	fmt.Fprintln(rw, "DICE OUTCOMES")
	fmt.Fprintln(rw)
	for _, s := range []struct {
		yours, theirs int
	}{
		{1, 2},
		{1, 1},
		{2, 1},
		{2, 2},
		{3, 2},
		{3, 1},
	} {
		outcomes := diceOdds(s.yours, s.theirs)
		fmt.Fprintf(rw, "Dice You:Them %d:%d\n", s.yours, s.theirs)
		var keys []rollOutcome
		for key := range outcomes {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].youLose > keys[j].youLose
		})
		for _, outcome := range keys {
			fmt.Fprintf(
				rw,
				"\t%02.2f%%: You Lose: %d, They Lose: %d\n",
				100*outcomes[outcome],
				outcome.youLose,
				outcome.theyLose,
			)
		}
	}
	fmt.Fprintln(rw)
	fmt.Fprintln(rw, "WAR OUTCOMES")
	fmt.Fprintln(rw)
	theirPiecesOptions := []int{1, 2, 3, 4, 5, 10, 20, 30}
	for _, theirPieces := range theirPiecesOptions {
		fmt.Fprintf(rw, "Their Pieces %d\n", theirPieces)
		for yourPieces := 2; ; yourPieces++ {
			odds := warWinProb(yourPieces, theirPieces)
			if odds < .1 {
				continue
			}
			yourExpectedCost, theirExpectedCost := expectedCost(yourPieces, theirPieces)
			yourLowerCost, yourUpperCost, yourCostWindow := yourCostIntervals(yourPieces, theirPieces, 0.33, .66)
			theirLowerCost, theirUpperCost, theirCostWindow := theirCostInterval(yourPieces, theirPieces, 0.33, 0.66)
			fmt.Fprintf(
				rw,
				"\tYour Pieces:%02d - %02.2f%% You Win, Your E[cost]: %02.2f, Your Cost Interval: [%02d-%02d] (%02.2f%%), Their E[cost]: %02.2f, Their Cost Interval: [%02d,%02d] (%02.2f%%)\n ",
				yourPieces, 100*odds,
				yourExpectedCost,
				yourLowerCost,
				yourUpperCost,
				yourCostWindow*100,
				theirExpectedCost,
				theirLowerCost,
				theirUpperCost,
				theirCostWindow*100,
			)
			if odds > 0.9 {
				break
			}
		}
	}
}

type diceCounts struct{ yourPieces, theirPieces int }

type rollOutcome struct{ youLose, theyLose int }

var oddsMemo = make(map[diceCounts]map[rollOutcome]float64)

func diceOdds(yours, theirs int) (outcomes map[rollOutcome]float64) {
	memoKey := diceCounts{yours, theirs}
	if outcomes, ok := oddsMemo[memoKey]; ok {
		return outcomes
	}
	defer func() { oddsMemo[memoKey] = outcomes }()
	yourRolls := allRolls(yours)
	theirRolls := allRolls(theirs)
	total := 0
	outcomesCount := make(map[rollOutcome]int)
	for _, yourRoll := range yourRolls {
		for _, theirRoll := range theirRolls {
			total++
			outcomesCount[singleRollOutcome(yourRoll, theirRoll)]++
		}
	}
	outcomes = make(map[rollOutcome]float64)
	for outcome, count := range outcomesCount {
		outcomes[outcome] = float64(count) / float64(total)
	}
	return outcomes
}

func allRolls(dice int) [][]int {
	if dice == 0 {
		return [][]int{nil}
	}
	var out [][]int
	subRolls := allRolls(dice - 1)
	for _, subRoll := range subRolls {
		for x := 1; x <= 6; x++ {
			out = append(out, append([]int{x}, subRoll...))
		}
	}
	return out
}

func singleRollOutcome(yourRoll, theirRoll []int) rollOutcome {
	sort.Sort(sort.Reverse(sort.IntSlice(yourRoll)))
	sort.Sort(sort.Reverse(sort.IntSlice(theirRoll)))
	out := rollOutcome{}
	for i := 0; i < len(yourRoll) && i < len(theirRoll); i++ {
		if yourRoll[i] > theirRoll[i] {
			out.theyLose++
		} else {
			out.youLose++
		}
	}
	return out
}

type piecesCounts struct{ you, them int }

var warOddsMemo = make(map[piecesCounts]map[warOutcome]float64)

type warOutcome struct{ youLose, theyLose int }

func warOdds(yourPieces, theirPieces int) (warOutcomeProbs map[warOutcome]float64) {
	memoKey := piecesCounts{yourPieces, theirPieces}
	if ans, ok := warOddsMemo[memoKey]; ok {
		return ans
	}
	defer func() { warOddsMemo[memoKey] = warOutcomeProbs }()
	if yourPieces == 1 || theirPieces == 0 {
		return map[warOutcome]float64{{0, 0}: 1}
	}
	yourDice := min(yourPieces-1, 3)
	theirDice := min(theirPieces, 2)
	rollOutcomes := diceOdds(yourDice, theirDice)
	warOutcomeProbs = make(map[warOutcome]float64)
	for rollOutcome, prob := range rollOutcomes {
		yourNextPieces, theirNextPieces := yourPieces-rollOutcome.youLose, theirPieces-rollOutcome.theyLose
		subWarOdds := warOdds(yourNextPieces, theirNextPieces)
		for subWarOutcome, subWarProb := range subWarOdds {
			warOutcomeProbs[warOutcome{rollOutcome.youLose + subWarOutcome.youLose, rollOutcome.theyLose + subWarOutcome.theyLose}] += prob * subWarProb
		}
	}
	return warOutcomeProbs
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

// more specific functions that build on top of knowing the probabilities of all possible outcomes

func warWinProb(yourPieces, theirPieces int) (retProb float64) {
	warOutcomes := warOdds(yourPieces, theirPieces)
	for outcome, prob := range warOutcomes {
		if outcome.theyLose == theirPieces {
			retProb += prob
		}
	}
	return retProb
}

func expectedCost(yourPieces, theirPieces int) (yourCost, theirCost float64) {
	warOutcomes := warOdds(yourPieces, theirPieces)
	for outcome, prob := range warOutcomes {
		yourCost += float64(outcome.youLose) * prob
		theirCost += float64(outcome.theyLose) * prob
	}
	return yourCost, theirCost
}

func theirCostInterval(yourPieces, theirPieces int, lower, upper float64) (lowerCost, upperCost int, window float64) {
	return costInterval(yourPieces, theirPieces, lower, upper, func(outcome warOutcome) int {
		return outcome.theyLose
	})
}

func yourCostIntervals(yourPieces, theirPieces int, lower, upper float64) (lowerCost, upperCost int, window float64) {
	return costInterval(yourPieces, theirPieces, lower, upper, func(outcome warOutcome) int {
		return outcome.youLose
	})
}

func costInterval(yourPieces, theirPieces int, lower, upper float64, getCost func(outcome warOutcome) int) (lowerCost, upperCost int, window float64) {
	warOutcomes := warOdds(yourPieces, theirPieces)
	costProbs := make(map[int]float64)
	var costOptions []int
	for outcome, prob := range warOutcomes {
		cost := getCost(outcome)
		costProbs[cost] += prob
		costOptions = append(costOptions, cost)
	}
	sort.Ints(costOptions)
	probsSoFar := float64(0)
	actualLowerProb, actualUpperProb := float64(0), float64(0)
	for i, costOption := range costOptions {
		prob := costProbs[costOption]
		newProbsSoFar := probsSoFar + prob
		if probsSoFar < lower && newProbsSoFar > lower {
			lowerCost = costOptions[i]
			actualLowerProb = probsSoFar
		}
		if probsSoFar < upper && newProbsSoFar > upper {
			upperCost = costOptions[i]
			actualUpperProb = newProbsSoFar
		}
		probsSoFar = newProbsSoFar
	}
	return lowerCost, upperCost, actualUpperProb - actualLowerProb
}
