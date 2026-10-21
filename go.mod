module lazyora

go 1.24.0

replace github.com/rivo/tview => ./third_party/tview

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/gdamore/tcell/v2 v2.13.10
	github.com/rivo/tview v0.42.0
	github.com/rivo/uniseg v0.4.7
	github.com/sijms/go-ora/v3 v3.0.2-0.20260731135026-9197bc12b6fe
	golang.org/x/crypto v0.45.0
)

require (
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/term v0.37.0 // indirect
	golang.org/x/text v0.31.0 // indirect
)
