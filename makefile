init_server:
	hz new -module empyrean_lens

server:
	hz update -idl ./server.thrift

dev_start:
	export MODE_ENV=dev && go run *.go

