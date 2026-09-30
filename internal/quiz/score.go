package quiz

import "math"

// Result is the outcome of a test under the usual opposition rules: every
// wrong answer subtracts a fraction (the penalty) of a correct one, and
// blank answers neither add nor subtract.
type Result struct {
	Total   int     `json:"total"`
	Correct int     `json:"correct"`
	Wrong   int     `json:"wrong"`
	Blank   int     `json:"blank"`
	Penalty float64 `json:"penalty"`
	// Net is correct - wrong*penalty, in questions.
	Net float64 `json:"net"`
	// Score is Net scaled to 0-10. It can be negative, as in the real exam
	// before minimums are applied.
	Score float64 `json:"score"`
	// ScoreNoPenalty is what the score would be if errors cost nothing, to
	// show how much the penalty is costing.
	ScoreNoPenalty float64 `json:"score_no_penalty"`
}

func Score(correct, wrong, blank int, penalty float64) Result {
	r := Result{Total: correct + wrong + blank, Correct: correct, Wrong: wrong, Blank: blank, Penalty: penalty}
	r.Net = round(float64(correct)-float64(wrong)*penalty, 4)
	if r.Total > 0 {
		r.Score = round(r.Net/float64(r.Total)*10, 2)
		r.ScoreNoPenalty = round(float64(correct)/float64(r.Total)*10, 2)
	}
	return r
}

func round(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(x*p) / p
}
