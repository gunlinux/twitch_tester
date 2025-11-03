#!/bin/sh

#Europe: Frankfurt (10) rtmp://euc10.contribute.live-video.net/app/{stream_key}
#Europe: Sweden, Stockholm (10) rtmp://eun10.contribute.live-video.net/app/{stream_key}

make build
./rtmp_tester rtmp://euc10.contribute.live-video.net/app "${STREAM_KEY}?bandwidthtest" 20


