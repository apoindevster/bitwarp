module bitwarp-client

go 1.23.2

require (
	github.com/apoindevster/bitwarp v0.0.0-unpublished
	github.com/apoindevster/bitwarp/commandclient v0.0.0-unpublished
	github.com/apoindevster/bitwarp/ui/common/commands v0.0.0-unpublished
	github.com/apoindevster/bitwarp/ui/connlist v0.0.0-unpublished
	github.com/apoindevster/bitwarp/ui/newconn v0.0.0-unpublished
	github.com/apoindevster/bitwarp/ui/runall v0.0.0-unpublished
	github.com/apoindevster/bitwarp/ui/shell v0.0.0-unpublished
	github.com/charmbracelet/bubbletea v1.3.6
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.73.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/bubbles v0.21.0 // indirect
	github.com/charmbracelet/colorprofile v0.2.3-0.20250311203215-f60798e515dc // indirect
	github.com/charmbracelet/lipgloss v1.1.0 // indirect
	github.com/charmbracelet/x/ansi v0.9.3 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13-0.20250311204145-2c3ea96c31dd // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sahilm/fuzzy v0.1.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250324211829-b45e905df463 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/apoindevster/bitwarp => ../

replace github.com/apoindevster/bitwarp/commandclient => ../commandclient

replace github.com/apoindevster/bitwarp/ui/shell => ./shell

replace github.com/apoindevster/bitwarp/ui/connlist => ./connlist

replace github.com/apoindevster/bitwarp/ui/newconn => ./newconn

replace github.com/apoindevster/bitwarp/ui/common/commands => ./common/commands

replace github.com/apoindevster/bitwarp/ui/runall => ./runall

replace github.com/apoindevster/bitwarp/ui/db => ./db
