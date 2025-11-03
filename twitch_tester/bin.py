import subprocess
import json
from twitch_tester.models import StreamStats


class BinMissing(Exception):
    pass


class TimeoutExpired(Exception):
    pass


def test_twitch_bandwidth(
    rtmp_endpoint: str,
    stream_key: str,
    duration: int = 30,
) -> StreamStats | None:
    cmd = [
        "./bin/rtmp_tester",
        rtmp_endpoint,
        stream_key,
        f'{duration}',
    ]

    try:
        # Start the subprocess
        proc = subprocess.Popen(
            cmd,
            stdout=subprocess.PIPE,
            stderr=None,
            text=True,
            bufsize=1,
            universal_newlines=True,
        )

        # Read stderr line by line
        out = []
        if proc.stdout:
            for line in iter(proc.stdout.readline, ""):
                out.append(line)
                # Parse progress lines (from -progress)

        data = ''.join(out)
        result = json.loads(data)
        # Wait for process completion
        proc.wait(timeout=duration + 1)

        # Calculate average bitrate
        return StreamStats(
            ping=-1,
            target_bitrate_kbps=10000,
            avg_bitrate_kbps=result.get('AvgBitrate', 0),
            peak_bitrate_kbps=0,
            duration_sec=10,
        )

    except subprocess.TimeoutExpired:
        proc.kill()
        raise TimeoutExpired
    except FileNotFoundError:
        raise BinMissing
