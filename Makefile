.PHONY: xlscfg
xlscfg:
	go build -o build/xlsxcfg ./bin/xlsxcfg

.PHONY: constant
constant:
	protoc --proto_path=./constant --go_out=./constant --go_opt=paths=source_relative constant.proto

.PHONY: test_pb
test_pb:
	protoc --proto_path=./tests --go_out=./tests --go_opt=paths=source_relative example.proto deps.proto
