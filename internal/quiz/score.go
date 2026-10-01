package quiz

import "math"

// Scale expresses scores the way the call does, e.g. out of 100 points with
// a pass mark. It only changes how results are shown: tests store the net
// ratio, so changing the scale re-expresses past results too.
type Scale struct {
	Max      float64 `json:"max"`
	PassMark float64 `json:"pass_mark"`
}

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
	// Ratio is Net / Total. It can be negative, as in the real exam.
	Ratio float64 `json:"ratio"`
	// Score is Ratio on the configured scale; ScoreNoPenalty is what it
	// would be if errors cost nothing, to show what the penalty costs.
	Score          float64 `json:"score"`
	ScoreNoPenalty float64 `json:"score_no_penalty"`
	Max            float64 `json:"max"`
	PassMark       float64 `json:"pass_mark"`
	Passed         bool    `json:"passed"`
}

func Score(correct, wrong, blank int, penalty float64, scale Scale) Result {
	r := Result{Total: correct + wrong + blank, Correct: correct, Wrong: wrong, Blank: blank, Penalty: penalty}
	r.Net = round(float64(correct)-float64(wrong)*penalty, 4)
	if r.Total > 0 {
		r.Ratio = r.Net / float64(r.Total)
		r.ScoreNoPenalty = round(float64(correct)/float64(r.Total)*scale.Max, 2)
	}
	r.applyScale(scale)
	return r
}

func (r *Result) applyScale(scale Scale) {
	r.Max, r.PassMark = scale.Max, scale.PassMark
	r.Score = round(r.Ratio*scale.Max, 2)
	r.Passed = r.Score >= scale.PassMark
}

func round(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(x*p) / p
}
