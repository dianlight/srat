#!/bin/bash
# Environment setup for SRAT build and test scripts
# Sets common environment variables and functions

# if SUPERVISOR_URL exists compone API_URL as ${SUPERVISOR_URL%/}:3000/
if [ -n "${SUPERVISOR_URL}" ]; then
	export API_URL="${SUPERVISOR_URL%/}:3000/"
else
	echo "WARN: Defaulting API_URL to http://localhost:3000/ - ensure this is correct for your environment"
	export API_URL="http://localhost:3000/"
fi
