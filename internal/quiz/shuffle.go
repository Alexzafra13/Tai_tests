package quiz

import (
	"fmt"
	"math/rand/v2"
	"regexp"
)

// optionOrder maps display positions to original options: the option shown
// at position i is original option order[i].
type optionOrder [4]int

var identityOrder = optionOrder{0, 1, 2, 3}

// letterReference matches options that name other options by letter, like
// "A y B son correctas", "las respuestas a) y c)" or "Solo la A". Any
// shuffle would break them, so the whole question keeps its order.
var letterReference = regexp.MustCompile(`(?i)(?:^|[^\p{L}])[a-d]\)?\s*(?:,|y|e|o)\s*[a-d]\)?(?:$|[^\p{L}])` +
	`|\b(?:la|las|opci[oó]n|opciones|respuesta|respuestas|apartado|apartados|letra|letras)\s+[a-d]\)?(?:$|[^\p{L}])`)

// relativeReference matches options whose meaning depends on their place
// among the others ("Todas las anteriores", "Ninguna es correcta"). They
// stay where they are; the rest are shuffled around them.
var relativeReference = regexp.MustCompile(`(?i)\b(?:anteriores?|todas|todos|ninguna|ninguno|ambas|ambos)\b`)

// newOptionOrder decides how to show a question's options. Every doubt
// errs on the side of not moving: a false positive only means an option
// stays in place, while a false negative could make a question wrong.
func newOptionOrder(options [4]string, fixed bool, shuffle func(n int, swap func(i, j int))) optionOrder {
	if fixed {
		return identityOrder
	}
	var movable []int
	for i, opt := range options {
		if letterReference.MatchString(opt) {
			return identityOrder
		}
		if !relativeReference.MatchString(opt) {
			movable = append(movable, i)
		}
	}

	order := identityOrder
	targets := append([]int(nil), movable...)
	shuffle(len(targets), func(i, j int) { targets[i], targets[j] = targets[j], targets[i] })
	for k, pos := range movable {
		order[pos] = targets[k]
	}
	return order
}

func randomShuffle(n int, swap func(i, j int)) { rand.Shuffle(n, swap) }

// display returns the options in the order they are shown.
func (o optionOrder) display(options [4]string) [4]string {
	var out [4]string
	for i, orig := range o {
		out[i] = options[orig]
	}
	return out
}

// toDisplay converts an original option index to its display position.
func (o optionOrder) toDisplay(orig int) int {
	for i, v := range o {
		if v == orig {
			return i
		}
	}
	return orig
}

// toOriginal converts a display position to the original option index.
func (o optionOrder) toOriginal(pos int) int { return o[pos] }

func (o optionOrder) String() string {
	return fmt.Sprintf("%d%d%d%d", o[0], o[1], o[2], o[3])
}

// parseOptionOrder reads the stored form ("2031"); anything invalid falls
// back to the original order.
func parseOptionOrder(s string) optionOrder {
	if len(s) != 4 {
		return identityOrder
	}
	var o optionOrder
	seen := [4]bool{}
	for i, c := range s {
		v := int(c - '0')
		if v < 0 || v > 3 || seen[v] {
			return identityOrder
		}
		seen[v] = true
		o[i] = v
	}
	return o
}
