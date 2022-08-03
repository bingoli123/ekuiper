go build -trimpath -modfile extensions.mod --buildmode=plugin -mod=mod -o _build/kuiper-linux-amd64/plugins/sources/Rabbitmq.so extensions/sources/rabbitmq/rabbitmq.go
go build -trimpath -modfile extensions.mod --buildmode=plugin -mod=mod -o _build/kuiper-linux-amd64/plugins/sinks/Rabbitmq.so extensions/sinks/rabbitmq/rabbitmq.go
