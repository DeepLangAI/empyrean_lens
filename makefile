server:
	hz update -idl ./server.thrift

init_server:
	hz new -module empyrean_lens

dev_start:
	export MODE_ENV=dev && go run *.go

prod:
	MODE_ENV=prod bash ./run.sh

pre:
	MODE_ENV=pre go run *.go