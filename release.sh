make

if [ -d "release/kuiperd/service/plugins/sources/Rabbitmq.so" ]; then
  rm release/kuiperd/service/plugins/sources/Rabbitmq.so
fi
if [ -d "release/kuiperd/service/plugins/sinks/Rabbitmq.so" ]; then
  rm release/kuiperd/service/plugins/sinks/Rabbitmq.so
fi

go build -trimpath -modfile extensions.mod --buildmode=plugin -o release/kuiperd/service/plugins/sources/Rabbitmq.so extensions/sources/rabbitmq/rabbitmq.go
go build -trimpath -modfile extensions.mod --buildmode=plugin -o release/kuiperd/service/plugins/sinks/Rabbitmq.so extensions/sinks/rabbitmq/rabbitmq.go
