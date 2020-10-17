package main

import (
	"fmt"
	"sort"
)

func main() {
	for _, s := range []struct {
		yours, theirs int
	}{
		{1, 2},
		{1, 1},
		{2, 1},
		{2, 2},
		{3, 2},
	} {
		outcomes := diceOdds(s.yours, s.theirs)
		fmt.Printf("Dice You:Them %d:%d\n", s.yours, s.theirs)
		var keys []RollOutcome
		for key := range outcomes {
			keys = append(keys, key)
		}
		sort.Slice(keys, func(i, j int) bool {
			return keys[i].youLose > keys[j].youLose
		})
		for _, outcome := range keys {
			fmt.Printf(
				"\t%02.2f%%: You Lose: %d, They Lose: %d\n",
				100*outcomes[outcome],
				outcome.youLose,
				outcome.theyLose,
			)
		}
	}
	fmt.Println()
	theirPiecesOptions := []int{1, 2, 3, 4, 5, 10, 20, 30}
	for _, theirPieces := range theirPiecesOptions {
		fmt.Printf("Their Pieces %d\n", theirPieces)
		for yourPieces := 2; ; yourPieces++ {
			odds := piecesOdds(yourPieces, theirPieces)
			if odds < .1 {
				continue
			}
			fmt.Printf("\tYour Pieces:%2d - %02.2f%% you win\n", yourPieces, 100*odds)
			if odds > 0.9 {
				break
			}
		}
	}
}

type dice struct {yours, them int}
var oddsMemo = make(map[dice]map[RollOutcome]float64)

func diceOdds(yours, theirs int) (outcomes map[RollOutcome]float64) {
	memoKey := dice{yours, theirs}
	if outcomes, ok := oddsMemo[memoKey]; ok {
		return outcomes
	}
	defer func() { oddsMemo[memoKey] = outcomes }()
	yourRolls := allRolls(yours)
	theirRolls := allRolls(theirs)
	total := 0
	outcomesCount := make(map[RollOutcome]int)
	for _, yourRoll := range yourRolls {
		for _, theirRoll := range theirRolls {
			total++
			outcomesCount[win(yourRoll, theirRoll)]++
		}
	}
	outcomes = make(map[RollOutcome]float64)
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

type RollOutcome struct {
	youLose, theyLose int
}

func win(yourRoll, theirRoll []int) RollOutcome {
	sort.Sort(sort.Reverse(sort.IntSlice(yourRoll)))
	sort.Sort(sort.Reverse(sort.IntSlice(theirRoll)))
	out := RollOutcome{}
	for i := 0; i < len(yourRoll) && i < len(theirRoll); i++ {
		if yourRoll[i] > theirRoll[i] {
			out.theyLose++
		} else {
			out.youLose++
		}
	}
	return out
}

type pieces struct{ you, them int }

var probsWinMemo = make(map[pieces]float64)

func piecesOdds(you, them int) (retProb float64) {
	memoKey := pieces{you, them}
	if prob, ok := probsWinMemo[memoKey]; ok {
		return prob
	}
	defer func() { probsWinMemo[memoKey] = retProb }()
	if you == 1 {
		return 0
	}
	if them == 0 {
		return 1
	}
	youNow := min(you-1, 3)
	themNow := min(them, 2)
	outcomes := diceOdds(youNow, themNow)
	for outcome, outcomeProb := range outcomes {
		retProb += outcomeProb * piecesOdds(you-outcome.youLose, them - outcome.theyLose)
	}
	return retProb
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
