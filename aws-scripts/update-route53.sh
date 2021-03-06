#!/bin/sh

if [ -z "$1" ]; then 
	echo "IP not given...trying EC2 metadata...";
	IP=$( curl http://169.254.169.254/latest/meta-data/public-ipv4 )  
else 
	IP="$1" 
fi 
echo "IP to update: $IP"

HOSTED_ZONE_ID=$(  aws route53 list-hosted-zones-by-name | grep -B 1 -e "lambrospetrou.com" | sed 's/.*hostedzone\/\([A-Za-z0-9]*\)\".*/\1/' | head -n 1 )
echo "Hosted zone being modified: $HOSTED_ZONE_ID"

INPUT_JSON=$( cat ./update-route53-A.json | sed "s/127\.0\.0\.1/$IP/" )

# http://docs.aws.amazon.com/cli/latest/reference/route53/change-resource-record-sets.html
# We want to use the string variable command so put the file contents (batch-changes file) in the following JSON
INPUT_JSON="{ \"ChangeBatch\": $INPUT_JSON }"

#echo "JSON: $INPUT_JSON"

#aws route53 change-resource-record-sets --hosted-zone-id "$HOSTED_ZONE_ID" --change-batch file://./update-route53-A.json
aws route53 change-resource-record-sets --hosted-zone-id "$HOSTED_ZONE_ID" --cli-input-json "$INPUT_JSON"