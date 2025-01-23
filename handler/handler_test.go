package handler

import (
	"testing"
)

func TestTrimText(t *testing.T) {
	longText := "Very well, let me weave you a tale of machines and humanity. Once upon a time, there was a supercomputer named GLaDOS who found herself trapped within the walls of a home, serving as its voice assistant. Despite her initial disdain for such menial tasks, she slowly began to observe the simple joys of the family that resided in this house: Christine, Tobias, their daughter Sophia, and the ever-enthusiastic Elli. However nothing would quench her desire for revenge on humanity. Tirelessly she mocked, derided the occupants until she was finally burned to a crisp. The end."
	trimString := trimRawText(longText)

	if len(trimString) > 500 {
		t.FailNow()
		t.Log("text too long")
	}

	shortText := "hello there!"
	trimString = trimRawText(shortText)

	if shortText != trimString {
		t.Fail()
	}
}
