module commands

go 1.23.2

require github.com/apoindevster/bitwarp v0.0.0-unpublished

require (
	github.com/apoindevster/bitwarp/commandclient v0.0.0-unpublished
	github.com/apoindevster/bitwarp/proto v0.0.0-unpublished
)

replace github.com/apoindevster/bitwarp => ../../../

replace github.com/apoindevster/bitwarp/commandclient => ../../../commandclient

replace github.com/apoindevster/bitwarp/proto => ../../../proto
