package quiz

import "testing"

func TestScore(t *testing.T) {
	outOf10 := Scale{Max: 10, PassMark: 5}
	tests := []struct {
		name                  string
		correct, wrong, blank int
		penalty               float64
		wantNet, wantScore    float64
		wantNoPenalty         float64
		wantPassed            bool
	}{
		{"perfect", 100, 0, 0, 1.0 / 3, 100, 10, 10, true},
		{"third penalty", 60, 30, 10, 1.0 / 3, 50, 5, 6, true},
		{"no penalty", 60, 30, 10, 0, 60, 6, 6, true},
		{"quarter penalty", 40, 20, 0, 0.25, 35, 5.83, 6.67, true},
		{"all blank", 0, 0, 25, 1.0 / 3, 0, 0, 0, false},
		{"negative", 10, 60, 30, 1.0 / 3, -10, -1, 1, false},
		{"empty test", 0, 0, 0, 1.0 / 3, 0, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Score(tt.correct, tt.wrong, tt.blank, tt.penalty, outOf10)
			if r.Net != tt.wantNet || r.Score != tt.wantScore || r.ScoreNoPenalty != tt.wantNoPenalty || r.Passed != tt.wantPassed {
				t.Errorf("Score(%d,%d,%d,%.3f) = net %v score %v nopen %v passed %v; want %v %v %v %v",
					tt.correct, tt.wrong, tt.blank, tt.penalty, r.Net, r.Score, r.ScoreNoPenalty, r.Passed,
					tt.wantNet, tt.wantScore, tt.wantNoPenalty, tt.wantPassed)
			}
			if r.Total != tt.correct+tt.wrong+tt.blank {
				t.Errorf("Total = %d", r.Total)
			}
		})
	}
}

func TestScoreOnOtherScale(t *testing.T) {
	// 60 right, 30 wrong, 10 blank at -1/3 is 50 net of 100: half marks.
	r := Score(60, 30, 10, 1.0/3, Scale{Max: 100, PassMark: 50})
	if r.Score != 50 || !r.Passed || r.ScoreNoPenalty != 60 || r.Max != 100 {
		t.Errorf("on 100: %+v", r)
	}
	r = Score(60, 30, 10, 1.0/3, Scale{Max: 50, PassMark: 25.01})
	if r.Score != 25 || r.Passed {
		t.Errorf("on 50 with pass mark 25.01: %+v", r)
	}
}
