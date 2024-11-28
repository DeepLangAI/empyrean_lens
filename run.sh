echo "Building the project...";
go build;
PID=$(lsof -t -i:19001);
if [ -n "$PID" ]; then
	echo "Killing process ID: $PID";
	kill -9 $PID;
fi;
echo "Starting the application...";
MODE_ENV=${MODE_ENV-prod} nohup ./empyrean_lens > output.log 2>&1 &
