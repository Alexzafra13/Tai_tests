package quiz

import (
	"context"

	"github.com/alexzafra13/tai_tests/internal/validate"
)

const scoringKey = "scoring"

// ScoringSettings are the user's scoring rules.
type ScoringSettings struct {
	Scale
	// DefaultPenalty pre-fills the penalty when creating a test.
	DefaultPenalty float64 `json:"default_penalty"`
}

// DefaultScoring is provisional until checked against the bases of the
// call: points out of 100, pass mark at half, and -1/3 per wrong answer.
var DefaultScoring = ScoringSettings{
	Scale:          Scale{Max: 100, PassMark: 50},
	DefaultPenalty: 1.0 / 3,
}

func (s ScoringSettings) validate() error {
	v := validate.Errors{}
	if s.Max <= 0 || s.Max > 1000 {
		v["max"] = "La puntuación máxima debe estar entre 1 y 1000"
	}
	if s.PassMark < 0 || s.PassMark > s.Max {
		v["pass_mark"] = "El aprobado debe estar entre 0 y la puntuación máxima"
	}
	if s.DefaultPenalty < 0 || s.DefaultPenalty > 1 {
		v["default_penalty"] = "La penalización debe estar entre 0 y 1"
	}
	if len(v) == 0 {
		return nil
	}
	return v
}

// Scoring returns the stored scoring settings, or the defaults.
func (s *Store) Scoring(ctx context.Context) (ScoringSettings, error) {
	sc := DefaultScoring
	_, err := s.settings.Get(ctx, scoringKey, &sc)
	return sc, err
}

func (s *Store) SetScoring(ctx context.Context, sc ScoringSettings) error {
	if err := sc.validate(); err != nil {
		return err
	}
	return s.settings.Set(ctx, scoringKey, sc)
}
