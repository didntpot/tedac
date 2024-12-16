module github.com/tedacmc/tedac

go 1.23.4

require (
	git.restartfu.com/restart/gophig.git v1.0.0
	github.com/df-mc/atomic v1.10.0
	github.com/df-mc/dragonfly v0.9.20-0.20241216095337-1c2ac18a5d85
	github.com/df-mc/worldupgrader v1.0.18
	github.com/go-gl/mathgl v1.1.0
	github.com/google/uuid v1.6.0
	github.com/hugolgst/rich-go v0.0.0-20210925091458-d59fb695d9c0
	github.com/samber/lo v1.38.1
	github.com/sandertv/go-raknet v1.14.2
	github.com/sandertv/gophertunnel v1.43.1-0.20241215115351-09b06aef681f
	github.com/segmentio/fasthash v1.0.3
	golang.org/x/oauth2 v0.23.0
)

require (
	github.com/brentp/intintmap v0.0.0-20190211203843-30dc0ade9af9 // indirect
	github.com/df-mc/goleveldb v1.1.9 // indirect
	github.com/go-jose/go-jose/v3 v3.0.3 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/muhammadmuzzammil1998/jsonc v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/restartfu/gophig v0.0.1 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c // indirect
	golang.org/x/image v0.21.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sandertv/go-raknet => github.com/tedacmc/tedac-raknet v0.0.4

replace github.com/sandertv/gophertunnel => github.com/tedacmc/tedac-gophertunnel v0.0.33
