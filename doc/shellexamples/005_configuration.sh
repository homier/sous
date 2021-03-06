# This is kind of a hack - in normal operation, Sous would block until its
# services had been accepted, but when bootstrapping, we need to wait for them
# to come up.
for n in {1..50}; do
	cygnus -H --env PORT0 http://192.168.99.100:7099/singularity > ~/server-singularity.txt
  if [ $( grep sous-server ~/server-singularity.txt | wc -l) -ge 2 ]; then
	  break
	fi
  sleep 0.1
done
cat ~/server-singularity.txt >2

leftport=$(grep 'sous-server.*left' ~/server-singularity.txt | awk '{ print $3 }')
rightport=$(grep 'sous-server.*right' ~/server-singularity.txt | awk '{ print $3 }')

serverURL=http://192.168.99.100:$leftport
echo "Determined server url as $serverURL"

until curl -s -I $serverURL; do
  sleep 0.1
done
sous config Server "$serverURL"
echo "Set server URL to: $(sous config Server)"

ETAG=$(curl -s -v http://192.168.99.100:$leftport/servers 2>&1 | sed -n '/Etag:/{s/.*: //; P; }')
echo $ETAG
sed "s/LEFTPORT/$leftport/; s/RIGHTPORT/$rightport/" < ~/templated-configs/servers.json > ~/servers.json
cat ~/servers.json
curl -v -X PUT -H "If-Match: ${ETAG//[$'\t\r\n ']}" -H "Content-Type: application/json" "${serverURL}/servers" --data "$(< ~/servers.json)"
curl -s "${serverURL}/servers"
cygnus --env TASK_HOST --env PORT0 -K -s http://192.168.99.100:7099/singularity
