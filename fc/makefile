zip:
	rm -rf output && mkdir output &&  GOOS=linux GOARCH=amd64 go build main.go && mv main output && cp -r conf output && zip -r -m link_trace.zip output


