package quiz

import "testing"

func TestScore(t *testing.T) {
	tests := []struct {
		name                  string
		correct, wrong, blank int
		penalty               float64
		wantNet, wantScore    float64
		wantNoPenalty         float64
	}{
		{"perfect", 100, 0, 0, 1.0 / 3, 100, 10, 10},
		{"third penalty", 60, 30, 10, 1.0 / 3, 50, 5, 6},
		{"no penalty", 60, 30, 10, 0, 60, 6, 6},
		{"quarter penalty", 40, 20, 0, 0.25, 35, 5.83, 6.67},
		{"all blank", 0, 0, 25, 1.0 / 3, 0, 0, 0},
		{"negative", 10, 60, 30, 1.0 / 3, -10, -1, 1},
		{"empty test", 0, 0, 0, 1.0 / 3, 0, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Score(tt.correct, tt.wrong, tt.blank, tt.penalty)
			if r.Net != tt.wantNet || r.Score != tt.wantScore || r.ScoreNoPenalty != tt.wantNoPenalty {
				t.Errorf("Score(%d,%d,%d,%.3f) = net %v score %v nopen %v; want %v %v %v",
					tt.correct, tt.wrong, tt.blank, tt.penalty, r.Net, r.Score, r.ScoreNoPenalty,
					tt.wantNet, tt.wantScore, tt.wantNoPenalty)
			}
			if r.Total != tt.correct+tt.wrong+tt.blank {
				t.Errorf("Total = %d", r.Total)
			}
		})
	}
}
