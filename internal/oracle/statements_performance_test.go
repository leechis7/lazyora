package oracle

import (
	"fmt"
	"strings"
	"testing"
)

func BenchmarkStatements(b *testing.B) {
	for _, count := range []int{20000, 40000, 80000} {
		text := strings.Repeat("SELECT 1 FROM dual;\n", count)
		b.Run(fmt.Sprintf("lines_%d", count), func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			for b.Loop() {
				got, err := Statements(text)
				if err != nil || len(got) != count || got[count-1].Line != count {
					b.Fatal("incorrect statements", err)
				}
			}
		})
	}
	for _, count := range []int{20000, 40000} {
		text := "SELECT 1" + strings.Repeat(" / 2", count) + " FROM dual;"
		b.Run(fmt.Sprintf("divisions_%d", count), func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			b.ReportAllocs()
			for b.Loop() {
				got, err := Statements(text)
				if err != nil || len(got) != 1 {
					b.Fatal("division split as slash delimiter", err)
				}
			}
		})
	}
}

func BenchmarkAtStatement(b *testing.B) {
	text := strings.Repeat("SELECT 1 FROM dual;\n", 80000)
	b.SetBytes(int64(len(text)))
	b.ReportAllocs()
	for b.Loop() {
		if got, err := AtStatement(text, len(text)); err != nil || got != "SELECT 1 FROM dual;" {
			b.Fatal("incorrect cursor statement", err)
		}
	}
}
