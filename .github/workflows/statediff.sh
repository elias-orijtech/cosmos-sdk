#!/bin/bash

go build -o /tmp/statediff github.com/cosmos/cosmos-sdk/cmd/statediff
/tmp/statediff < /dev/stdin

retVal=$?
echo $retVal

# Exit code 128 means the patch touches state code.
if [ $retVal -eq 128 ]; then
	echo "::warning ::PR possibly affects state"
fi

exit $retVal
