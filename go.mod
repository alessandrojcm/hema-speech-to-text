module github.com/your-org/hema-replay-system

go 1.24.4

require (
	github.com/andreykaipov/goobs v1.2.3
	github.com/baabaaox/go-webrtcvad v1.1.1
	github.com/dh1tw/gosamplerate v0.1.2
	github.com/ggerganov/whisper.cpp/bindings/go v0.0.0-20250807023745-4245c77b654c
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/transforms v0.0.0-20180121090939-51830ccc35a5
	github.com/go-audio/wav v1.1.0
	github.com/go-skynet/go-llama.cpp v0.0.0-20240314183750-6a8041ef6b46
	github.com/gordonklaus/portaudio v0.0.0-20250206071425-98a94950218b
	github.com/mitchellh/mapstructure v1.5.0
	github.com/orcaman/writerseeker v0.0.0-20200621085525-1d3f536ff85e
	github.com/rs/zerolog v1.32.0
	github.com/spf13/viper v1.18.0
	github.com/stretchr/testify v1.10.0
	gonum.org/v1/gonum v0.16.0
)

require (
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-audio/riff v1.0.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/mmcloughlin/profile v0.1.1 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/pelletier/go-toml/v2 v2.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/go-skynet/go-llama.cpp => ./go-llama.cpp
