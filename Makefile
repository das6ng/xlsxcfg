.PHONY: xlscfg
xlscfg:
	go build -o build/xlscfg ./bin/xlscfg

.PHONY: test_pb
test_pb:
	protoc --proto_path=./tests --go_out=./tests --go_opt=paths=source_relative example.proto deps.proto
